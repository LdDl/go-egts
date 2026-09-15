package server

import (
	"context"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"net"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/LdDl/go-egts/egts/packet"
	"github.com/LdDl/go-egts/gateway/configuration"
	"github.com/gomodule/redigo/redis"
	"github.com/stretchr/testify/assert"
)

func TestValkeyIndependentDeliveryAndRestoreIntegration(t *testing.T) {
	if os.Getenv("GO_EGTS_VALKEY_INTEGRATION") != "1" {
		t.Skip("Set GO_EGTS_VALKEY_INTEGRATION=1 with the example Valkey infrastructure running")
	}
	cfg, err := configuration.PrepareFileConfiguration("../examples/valkey/gateway.toml")
	assert.NoError(t, err)
	if err != nil {
		return
	}
	cfg.DeliveryCfg.AckMode = "queued"
	cfg.DeliveryCfg.QueueCapacity = 2
	cfg.DeliveryCfg.DumpDirectory = t.TempDir()
	connections := make(map[string]redis.Conn)
	streamName := fmt.Sprintf("egts.test.%d", time.Now().UnixNano())
	for i := range cfg.DestinationsCfg.Valkey {
		valkey := cfg.DestinationsCfg.Valkey[i]
		cfg.DestinationsCfg.Valkey[i].StreamName = streamName
		connection, err := redis.Dial("tcp", fmt.Sprintf("%s:%d", valkey.Host, valkey.Port),
			redis.DialUsername(valkey.UserName), redis.DialPassword(valkey.Password),
			redis.DialConnectTimeout(time.Second), redis.DialReadTimeout(time.Second), redis.DialWriteTimeout(time.Second))
		assert.NoError(t, err)
		if err != nil {
			return
		}
		t.Cleanup(func() {
			_, err := connection.Do("DEL", streamName)
			assert.NoError(t, err)
			err = connection.Close()
			assert.NoError(t, err)
		})
		connections[valkey.ID] = connection
	}
	blocked, err := net.Listen("tcp", "127.0.0.1:0")
	assert.NoError(t, err)
	if err != nil {
		return
	}
	t.Cleanup(func() {
		err := blocked.Close()
		assert.NoError(t, err)
	})
	backupPort := cfg.DestinationsCfg.Valkey[1].Port
	cfg.DestinationsCfg.Valkey[1].Port = blocked.Addr().(*net.TCPAddr).Port
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
	stopped := make(chan struct{})
	go func() {
		done <- s.serve(ctx, listener)
		close(stopped)
	}()
	t.Cleanup(func() {
		cancel()
		<-stopped
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
	err = conn.SetDeadline(time.Now().Add(5 * time.Second))
	assert.NoError(t, err)
	raw, err := hex.DecodeString("0100000b002300000001991800000001ef0000000202101500d2312b104fba3a9ed227bc35030000b200000000006a8d")
	assert.NoError(t, err)
	for i := 0; i < 3; i++ {
		_, err = conn.Write(raw)
		assert.NoError(t, err)
		answer, err := packet.ReadFrame(conn)
		assert.NoError(t, err)
		if err != nil {
			return
		}
		response, err := packet.ReadPacket(answer)
		assert.NoError(t, err)
		if i < 2 {
			assert.Zero(t, response.ServicesFrameData.(*packet.PTResponse).ProcessingResult)
		} else {
			assert.Equal(t, packet.EGTS_PC_NO_RES_AVAIL, response.ServicesFrameData.(*packet.PTResponse).ProcessingResult)
		}
	}
	assert.Eventually(t, func() bool {
		s.mu.Lock()
		defer s.mu.Unlock()
		return len(s.queue) == 2 && !s.queue[0].valkey["monitoring"] && !s.queue[1].valkey["monitoring"]
	}, 3*time.Second, 10*time.Millisecond)
	cancel()
	err = <-done
	assert.NoError(t, err)
	cfg.DestinationsCfg.Valkey[1].Port = backupPort
	cfg.DestinationsCfg.Valkey[0], cfg.DestinationsCfg.Valkey[1] = cfg.DestinationsCfg.Valkey[1], cfg.DestinationsCfg.Valkey[0]
	restored, err := newServer(cfg)
	assert.NoError(t, err)
	if err != nil {
		return
	}
	assert.Len(t, restored.queue, 2)
	for _, item := range restored.queue {
		assert.Equal(t, map[string]bool{"backup": true}, item.valkey)
	}
	last := restored.queue[1]
	listener, err = net.Listen("tcp", "127.0.0.1:0")
	assert.NoError(t, err)
	if err != nil {
		return
	}
	recovery, stopRecovery := context.WithCancel(context.Background())
	recovered := make(chan error, 1)
	finished := make(chan struct{})
	go func() {
		recovered <- restored.serve(recovery, listener)
		close(finished)
	}()
	t.Cleanup(func() {
		stopRecovery()
		<-finished
	})
	select {
	case <-last.done:
	case <-time.After(5 * time.Second):
		t.Error("Restored Valkey packets were not delivered")
	}
	stopRecovery()
	err = <-recovered
	assert.NoError(t, err)
	streams := make(map[string][]map[string]string)
	for id, connection := range connections {
		reply, err := connection.Do("XRANGE", streamName, "-", "+")
		assert.NoError(t, err)
		entries, err := redis.Values(reply, nil)
		assert.NoError(t, err)
		assert.Len(t, entries, 2)
		for _, entry := range entries {
			values, err := redis.Values(entry, nil)
			assert.NoError(t, err)
			var entryID string
			var fields []interface{}
			_, err = redis.Scan(values, &entryID, &fields)
			assert.NoError(t, err)
			data, err := redis.StringMap(fields, nil)
			assert.NoError(t, err)
			var event receivedEvent
			err = json.Unmarshal([]byte(data["payload"]), &event)
			assert.NoError(t, err)
			assert.Equal(t, raw, event.Record.Raw)
			streams[id] = append(streams[id], data)
		}
	}
	assert.Equal(t, streams["monitoring"], streams["backup"])
	if len(streams["monitoring"]) == 2 {
		assert.NotEqual(t, streams["monitoring"][0]["message_id"], streams["monitoring"][1]["message_id"])
	}
	data, err := os.ReadFile(filepath.Join(cfg.DeliveryCfg.DumpDirectory, configuration.DUMP_FILENAME))
	assert.NoError(t, err)
	assert.JSONEq(t, `{"version":5}`, string(data))
}
