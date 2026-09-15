package server

import (
	"bytes"
	"context"
	"encoding/hex"
	"encoding/json"
	"net"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/LdDl/go-egts/egts/packet"
	"github.com/LdDl/go-egts/gateway/configuration"
	"github.com/LdDl/go-egts/gateway/destination"
	"github.com/gomodule/redigo/redis"
	"github.com/stretchr/testify/assert"
)

type valkeyDumpTestCase struct {
	name      string
	version   int
	pending   []string
	disabled  string
	errorText string
}

func TestValkeyDumpDestinations(t *testing.T) {
	cases := []valkeyDumpTestCase{
		{name: "restore only pending destination", version: 5, pending: []string{"backup"}},
		{name: "missing destination", version: 5, pending: []string{"missing"}, errorText: "unavailable destination"},
		{name: "disabled destination", version: 5, pending: []string{"backup"}, disabled: "backup", errorText: "unavailable destination"},
		{name: "duplicate destination", version: 5, pending: []string{"backup", "backup"}, errorText: "Duplicate"},
		{name: "Valkey in old dump", version: 4, pending: []string{"backup"}, errorText: "legacy packet dump"},
		{name: "legacy local destination", version: 1},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			cfg := configuration.DefaultConfiguration()
			cfg.DeliveryCfg.DumpDirectory = t.TempDir()
			cfg.DestinationsCfg.Stdout = false
			cfg.DestinationsCfg.File.Enabled = true
			cfg.DestinationsCfg.File.Directory = t.TempDir()
			for _, id := range []string{"backup", "monitoring"} {
				valkey := configuration.DefaultValkeyDestination()
				valkey.ID = id
				valkey.Enabled = id != tc.disabled
				valkey.UserName = "user"
				valkey.Password = "secret"
				cfg.DestinationsCfg.Valkey = append(cfg.DestinationsCfg.Valkey, valkey)
			}
			raw, err := hex.DecodeString("0100000b002300000001991800000001ef0000000202101500d2312b104fba3a9ed227bc35030000b200000000006a8d")
			assert.NoError(t, err)
			saved := dumpRecord{ReceivedAt: time.Now().UTC(), Raw: raw, ValkeyIDs: tc.pending, File: true}
			var buffer bytes.Buffer
			encoder := json.NewEncoder(&buffer)
			err = encoder.Encode(dumpHeader{Version: tc.version})
			assert.NoError(t, err)
			err = encoder.Encode(saved)
			assert.NoError(t, err)
			name := filepath.Join(cfg.DeliveryCfg.DumpDirectory, configuration.DUMP_FILENAME)
			err = os.WriteFile(name, buffer.Bytes(), 0600)
			assert.NoError(t, err)
			s, err := newServer(cfg)
			if tc.errorText != "" {
				assert.ErrorContains(t, err, tc.errorText)
				assert.Nil(t, s)
			} else {
				assert.NoError(t, err)
				if err != nil {
					return
				}
				assert.Len(t, s.queue, 1)
				item := s.queue[0]
				assert.Equal(t, raw, item.record.Raw)
				assert.Equal(t, saved.ReceivedAt, item.record.ReceivedAt)
				assert.True(t, item.file)
				assert.False(t, item.stdout)
				assert.Nil(t, item.relay)
				assert.Empty(t, item.egts)
				if tc.version == 1 {
					assert.Empty(t, item.valkey)
				} else {
					assert.Equal(t, map[string]bool{"backup": true}, item.valkey)
				}
				err = s.file.Close()
				assert.NoError(t, err)
				for _, writer := range s.valkeys {
					err = writer.Close()
					assert.NoError(t, err)
				}
			}
			data, err := os.ReadFile(name)
			assert.NoError(t, err)
			assert.Equal(t, buffer.Bytes(), data)
		})
	}
}

func TestValkeyEnqueueAndSave(t *testing.T) {
	cfg := configuration.DefaultConfiguration()
	cfg.DeliveryCfg.DumpDirectory = t.TempDir()
	cfg.DestinationsCfg.Stdout = false
	for _, id := range []string{"monitoring", "backup", "disabled"} {
		valkey := configuration.DefaultValkeyDestination()
		valkey.ID = id
		valkey.Enabled = id != "disabled"
		valkey.UserName = "user"
		valkey.Password = "secret"
		cfg.DestinationsCfg.Valkey = append(cfg.DestinationsCfg.Valkey, valkey)
	}
	s, err := newServer(cfg)
	assert.NoError(t, err)
	if err != nil {
		return
	}
	t.Cleanup(func() {
		for _, writer := range s.valkeys {
			err := writer.Close()
			assert.NoError(t, err)
		}
	})
	s.accepting = true
	raw, err := hex.DecodeString("0100000b002300000001991800000001ef0000000202101500d2312b104fba3a9ed227bc35030000b200000000006a8d")
	assert.NoError(t, err)
	record, err := destination.NewRecord(time.Now(), destination.Source{}, raw)
	assert.NoError(t, err)
	item, err := s.enqueue(record, nil)
	assert.NoError(t, err)
	assert.Equal(t, map[string]bool{"monitoring": true, "backup": true}, item.valkey)
	assert.Nil(t, item.relay)
	delete(item.valkey, "monitoring")
	err = s.saveDump()
	assert.NoError(t, err)
	data, err := os.ReadFile(filepath.Join(cfg.DeliveryCfg.DumpDirectory, configuration.DUMP_FILENAME))
	assert.NoError(t, err)
	decoder := json.NewDecoder(bytes.NewReader(data))
	var header dumpHeader
	err = decoder.Decode(&header)
	assert.NoError(t, err)
	assert.Equal(t, 5, header.Version)
	var saved dumpRecord
	err = decoder.Decode(&saved)
	assert.NoError(t, err)
	assert.Equal(t, []string{"backup"}, saved.ValkeyIDs)
	assert.Equal(t, record.Raw, saved.Raw)
	assert.False(t, saved.Stdout)
	assert.Nil(t, saved.SessionID)
	assert.Contains(t, string(data), `"identity":null`)
}

func TestDeliveredWaitsForEveryValkey(t *testing.T) {
	cfg := configuration.DefaultConfiguration()
	cfg.DeliveryCfg.AckMode = "delivered"
	cfg.DeliveryCfg.DumpDirectory = filepath.Join(t.TempDir(), "queue")
	cfg.DestinationsCfg.Stdout = false
	var listeners []*net.TCPListener
	for _, id := range []string{"first", "second"} {
		listener, err := net.Listen("tcp", "127.0.0.1:0")
		assert.NoError(t, err)
		if err != nil {
			return
		}
		t.Cleanup(func() {
			err := listener.Close()
			assert.NoError(t, err)
		})
		listeners = append(listeners, listener.(*net.TCPListener))
		valkey := configuration.DefaultValkeyDestination()
		valkey.ID = id
		valkey.Enabled = true
		valkey.Port = listener.Addr().(*net.TCPAddr).Port
		cfg.DestinationsCfg.Valkey = append(cfg.DestinationsCfg.Valkey, valkey)
	}
	s, err := newServer(cfg)
	assert.NoError(t, err)
	if err != nil {
		return
	}
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	assert.NoError(t, err)
	if err != nil {
		return
	}
	ctx, cancel := context.WithCancel(context.Background())
	done := make(chan error, 1)
	go func() {
		done <- s.serve(ctx, listener)
	}()
	t.Cleanup(func() {
		cancel()
		select {
		case err := <-done:
			assert.NoError(t, err)
		case <-time.After(3 * time.Second):
			t.Error("Gateway did not stop")
		}
	})
	conn, err := net.DialTimeout("tcp", listener.Addr().String(), time.Second)
	assert.NoError(t, err)
	if err != nil {
		return
	}
	t.Cleanup(func() {
		err := conn.Close()
		assert.NoError(t, err)
	})
	err = conn.SetDeadline(time.Now().Add(3 * time.Second))
	assert.NoError(t, err)
	raw, err := hex.DecodeString("0100000b002300000001991800000001ef0000000202101500d2312b104fba3a9ed227bc35030000b200000000006a8d")
	assert.NoError(t, err)
	_, err = conn.Write(raw)
	assert.NoError(t, err)
	for i, listener := range listeners {
		err = listener.SetDeadline(time.Now().Add(3 * time.Second))
		assert.NoError(t, err)
		peer, err := listener.Accept()
		assert.NoError(t, err)
		if err != nil {
			return
		}
		t.Cleanup(func() {
			err := peer.Close()
			assert.NoError(t, err)
		})
		err = peer.SetDeadline(time.Now().Add(3 * time.Second))
		assert.NoError(t, err)
		reader := redis.NewConn(peer, 3*time.Second, 3*time.Second)
		request, err := reader.Receive()
		assert.NoError(t, err)
		if err != nil {
			return
		}
		command, err := redis.Strings(request, nil)
		assert.NoError(t, err)
		assert.Len(t, command, 7)
		if len(command) != 7 {
			return
		}
		assert.Equal(t, []string{"XADD", "egts.packets", "*", "message_id"}, command[:4])
		assert.Equal(t, "payload", command[5])
		var event receivedEvent
		err = json.Unmarshal([]byte(command[6]), &event)
		assert.NoError(t, err)
		assert.Equal(t, raw, event.Record.Raw)
		_, err = peer.Write([]byte("$3\r\n1-0\r\n"))
		assert.NoError(t, err)
		if i == 0 {
			err = conn.SetReadDeadline(time.Now().Add(50 * time.Millisecond))
			assert.NoError(t, err)
			_, err = packet.ReadFrame(conn)
			var timeout net.Error
			assert.ErrorAs(t, err, &timeout)
			if timeout != nil {
				assert.True(t, timeout.Timeout())
			}
			err = conn.SetReadDeadline(time.Now().Add(3 * time.Second))
			assert.NoError(t, err)
		}
	}
	answer, err := packet.ReadFrame(conn)
	assert.NoError(t, err)
	response, err := packet.ReadPacket(answer)
	assert.NoError(t, err)
	assert.Zero(t, response.ServicesFrameData.(*packet.PTResponse).ProcessingResult)
}
