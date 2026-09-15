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
	"github.com/stretchr/testify/assert"
)

func TestRelayDumpRestoresSessions(t *testing.T) {
	root := t.TempDir()
	upstream := configuration.DefaultConfiguration()
	upstream.DeliveryCfg.AckMode = "delivered"
	upstream.DeliveryCfg.DumpDirectory = filepath.Join(root, "upstream-queue")
	upstream.DestinationsCfg.Stdout = false
	upstream.DestinationsCfg.File.Enabled = true
	upstream.DestinationsCfg.File.Directory = filepath.Join(root, "upstream-packets")
	collector, err := newServer(upstream)
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
	collected := make(chan error, 1)
	go func() {
		collected <- collector.serve(ctx, listener)
	}()
	t.Cleanup(func() {
		cancel()
		select {
		case err := <-collected:
			assert.NoError(t, err)
		case <-time.After(3 * time.Second):
			t.Error("Destination gateway did not stop")
		}
	})
	cfg := configuration.DefaultConfiguration()
	cfg.DeliveryCfg.DumpDirectory = filepath.Join(root, "relay-queue")
	cfg.DestinationsCfg.Stdout = false
	cfg.DestinationsCfg.File.Enabled = true
	cfg.DestinationsCfg.File.Directory = filepath.Join(root, "relay-packets")
	cfg.DestinationsCfg.EGTS.Enabled = true
	cfg.DestinationsCfg.EGTS.Port = listener.Addr().(*net.TCPAddr).Port
	s, err := newServer(cfg)
	assert.NoError(t, err)
	if err != nil {
		return
	}
	s.accepting = true
	first, err := s.newRelaySession()
	assert.NoError(t, err)
	second, err := s.newRelaySession()
	assert.NoError(t, err)
	raw, err := hex.DecodeString("0100000b002300000001991800000001ef0000000202101500d2312b104fba3a9ed227bc35030000b200000000006a8d")
	assert.NoError(t, err)
	for i, session := range []*relaySession{first, second, first} {
		identity := []byte{101, 0, 0, 0, 0}
		if session == second {
			identity[0] = 202
		}
		record, err := destination.NewRecord(time.Now().UTC(), destination.Source{RemoteAddress: "original-terminal"}, raw)
		assert.NoError(t, err)
		item, err := s.enqueue(record, &relayPacket{session: session, identity: identity})
		assert.NoError(t, err)
		if i == 0 {
			err = s.file.Write(record)
			assert.NoError(t, err)
			item.file = false
		}
	}
	err = s.endRelaySession(first)
	assert.NoError(t, err)
	err = s.endRelaySession(second)
	assert.NoError(t, err)
	assert.Len(t, s.relays, 2)
	assert.Equal(t, 2, first.pending)
	assert.Equal(t, 1, second.pending)
	err = s.saveDump()
	assert.NoError(t, err)
	err = s.file.Close()
	assert.NoError(t, err)
	for _, session := range s.relays {
		err = session.writer.Close()
		assert.NoError(t, err)
	}
	name := filepath.Join(cfg.DeliveryCfg.DumpDirectory, configuration.DUMP_FILENAME)
	data, err := os.ReadFile(name)
	assert.NoError(t, err)
	lines := bytes.Split(bytes.TrimSuffix(data, []byte("\n")), []byte("\n"))
	assert.Len(t, lines, 4)
	assert.JSONEq(t, `{"version":2}`, string(lines[0]))
	cfg.DestinationsCfg.EGTS.Enabled = false
	invalid, err := newServer(cfg)
	assert.ErrorContains(t, err, "unavailable destination")
	assert.Nil(t, invalid)
	unchanged, err := os.ReadFile(name)
	assert.NoError(t, err)
	assert.Equal(t, data, unchanged)
	cfg.DestinationsCfg.EGTS.Enabled = true
	restored, err := newServer(cfg)
	assert.NoError(t, err)
	if err != nil {
		return
	}
	t.Cleanup(func() {
		err := restored.file.Close()
		assert.NoError(t, err)
		for _, session := range restored.relays {
			err = session.writer.Close()
			assert.NoError(t, err)
		}
	})
	assert.Len(t, restored.queue, 3)
	assert.Len(t, restored.relays, 2)
	assert.Same(t, restored.queue[0].relay.session, restored.queue[2].relay.session)
	assert.NotSame(t, restored.queue[0].relay.session, restored.queue[1].relay.session)
	assert.Equal(t, first.id, restored.queue[0].relay.session.id)
	assert.Equal(t, second.id, restored.queue[1].relay.session.id)
	assert.True(t, restored.queue[0].relay.session.closed)
	assert.False(t, restored.queue[0].file)
	assert.True(t, restored.queue[0].egts)
	last := restored.queue[2]
	deliveryCtx, stopDelivery := context.WithCancel(context.Background())
	done := make(chan error, 1)
	go func() {
		done <- restored.deliver(deliveryCtx)
	}()
	select {
	case <-last.done:
	case <-time.After(5 * time.Second):
		t.Error("Restored relay packets were not delivered")
	}
	stopDelivery()
	err = <-done
	assert.NoError(t, err)
	assert.Empty(t, restored.queue)
	assert.Empty(t, restored.relays)
	data, err = os.ReadFile(name)
	assert.NoError(t, err)
	assert.JSONEq(t, `{"version":2}`, string(data))
	data, err = os.ReadFile(filepath.Join(cfg.DestinationsCfg.File.Directory, configuration.PACKETS_FILENAME))
	assert.NoError(t, err)
	lines = bytes.Split(bytes.TrimSuffix(data, []byte("\n")), []byte("\n"))
	assert.Len(t, lines, 3)
	for _, line := range lines {
		var event receivedEvent
		err = json.Unmarshal(line, &event)
		assert.NoError(t, err)
		assert.Equal(t, raw, event.Record.Raw)
		assert.Equal(t, "original-terminal", event.Record.Source.RemoteAddress)
	}
	data, err = os.ReadFile(filepath.Join(upstream.DestinationsCfg.File.Directory, configuration.PACKETS_FILENAME))
	assert.NoError(t, err)
	lines = bytes.Split(bytes.TrimSuffix(data, []byte("\n")), []byte("\n"))
	assert.Len(t, lines, 5)
	var terminals []uint32
	connections := make(map[uint32]string)
	for _, line := range lines {
		var event receivedEvent
		err = json.Unmarshal(line, &event)
		assert.NoError(t, err)
		assert.NotNil(t, event.Record.Source.TerminalID)
		if event.Record.Source.TerminalID == nil {
			continue
		}
		terminalID := *event.Record.Source.TerminalID
		previous, exists := connections[terminalID]
		if exists {
			assert.Equal(t, previous, event.Record.Source.RemoteAddress)
		}
		connections[terminalID] = event.Record.Source.RemoteAddress
		if bytes.Equal(raw, event.Record.Raw) {
			terminals = append(terminals, terminalID)
		}
	}
	assert.Equal(t, []uint32{101, 202, 101}, terminals)
	assert.Len(t, connections, 2)
	assert.NotEqual(t, connections[101], connections[202])
}

func TestRelayStallSavesDump(t *testing.T) {
	for _, mode := range []string{"queued", "delivered"} {
		t.Run(mode, func(t *testing.T) {
			remote, err := net.Listen("tcp", "127.0.0.1:0")
			assert.NoError(t, err)
			if err != nil {
				return
			}
			t.Cleanup(func() {
				err := remote.Close()
				assert.NoError(t, err)
			})
			cfg := configuration.DefaultConfiguration()
			cfg.DeliveryCfg.AckMode = mode
			cfg.DeliveryCfg.QueueCapacity = 1
			cfg.DeliveryCfg.DumpAfterSeconds = 1
			cfg.DeliveryCfg.DumpDirectory = filepath.Join(t.TempDir(), "queue")
			cfg.DestinationsCfg.Stdout = false
			cfg.DestinationsCfg.EGTS.Enabled = true
			cfg.DestinationsCfg.EGTS.Port = remote.Addr().(*net.TCPAddr).Port
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
			defer cancel()
			done := make(chan error, 1)
			go func() {
				done <- s.serve(ctx, listener)
			}()
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
			err = remote.(*net.TCPListener).SetDeadline(time.Now().Add(3 * time.Second))
			assert.NoError(t, err)
			peer, err := remote.Accept()
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
			forwarded, err := packet.ReadFrame(peer)
			assert.NoError(t, err)
			assert.Equal(t, raw, forwarded)
			if mode == "queued" {
				for _, expected := range []uint8{packet.EGTS_PC_OK, packet.EGTS_PC_NO_RES_AVAIL} {
					answer, err := packet.ReadFrame(conn)
					assert.NoError(t, err)
					if err != nil {
						return
					}
					response, err := packet.ReadPacket(answer)
					assert.NoError(t, err)
					assert.Equal(t, expected, response.ServicesFrameData.(*packet.PTResponse).ProcessingResult)
					if expected == packet.EGTS_PC_OK {
						_, err = conn.Write(raw)
						assert.NoError(t, err)
					}
				}
			} else {
				err = conn.SetReadDeadline(time.Now().Add(50 * time.Millisecond))
				assert.NoError(t, err)
				_, err = packet.ReadFrame(conn)
				var timeout net.Error
				assert.ErrorAs(t, err, &timeout)
				if timeout != nil {
					assert.True(t, timeout.Timeout())
				}
			}
			select {
			case err = <-done:
				assert.ErrorContains(t, err, "delivery stalled")
			case <-time.After(3 * time.Second):
				t.Fatal("Gateway did not save stalled relay queue")
			}
			data, err := os.ReadFile(filepath.Join(cfg.DeliveryCfg.DumpDirectory, configuration.DUMP_FILENAME))
			assert.NoError(t, err)
			lines := bytes.Split(bytes.TrimSuffix(data, []byte("\n")), []byte("\n"))
			assert.Len(t, lines, 2)
			if len(lines) != 2 {
				return
			}
			assert.JSONEq(t, `{"version":2}`, string(lines[0]))
			var saved dumpRecord
			err = json.Unmarshal(lines[1], &saved)
			assert.NoError(t, err)
			assert.Equal(t, raw, saved.Raw)
			assert.True(t, saved.EGTS)
			assert.False(t, saved.Stdout)
			assert.False(t, saved.File)
			assert.NotNil(t, saved.SessionID)
			assert.Nil(t, saved.Identity)
		})
	}
}
