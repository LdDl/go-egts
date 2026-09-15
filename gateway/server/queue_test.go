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

func TestQueueFailureAndRestore(t *testing.T) {
	for _, mode := range []string{"queued", "delivered"} {
		t.Run(mode, func(t *testing.T) {
			root := t.TempDir()
			cfg := configuration.DefaultConfiguration()
			cfg.DeliveryCfg.AckMode = mode
			cfg.DeliveryCfg.QueueCapacity = 1
			cfg.DeliveryCfg.DumpAfterSeconds = 1
			cfg.DeliveryCfg.DumpDirectory = filepath.Join(root, "queue")
			cfg.DestinationsCfg.Stdout = false
			cfg.DestinationsCfg.File.Enabled = true
			cfg.DestinationsCfg.File.Directory = filepath.Join(root, "packets")
			cfg.DestinationsCfg.File.Rotation.MaxFileSizeBytes = 64
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
			conn, err := net.Dial("tcp", listener.Addr().String())
			assert.NoError(t, err)
			if err != nil {
				return
			}
			t.Cleanup(func() {
				err := conn.Close()
				assert.NoError(t, err)
			})
			raw, err := hex.DecodeString("0100000b002300000001991800000001ef0000000202101500d2312b104fba3a9ed227bc35030000b200000000006a8d")
			assert.NoError(t, err)
			_, err = conn.Write(raw)
			assert.NoError(t, err)
			if mode == "queued" {
				err = conn.SetDeadline(time.Now().Add(2 * time.Second))
				assert.NoError(t, err)
				answer, err := packet.ReadFrame(conn)
				assert.NoError(t, err)
				if err != nil {
					return
				}
				response, err := packet.ReadPacket(answer)
				assert.NoError(t, err)
				assert.Zero(t, response.ServicesFrameData.(*packet.PTResponse).ProcessingResult)
				_, err = conn.Write(raw)
				assert.NoError(t, err)
				answer, err = packet.ReadFrame(conn)
				assert.NoError(t, err)
				response, err = packet.ReadPacket(answer)
				assert.NoError(t, err)
				assert.Equal(t, packet.EGTS_PC_NO_RES_AVAIL, response.ServicesFrameData.(*packet.PTResponse).ProcessingResult)
			} else {
				err = conn.SetReadDeadline(time.Now().Add(100 * time.Millisecond))
				assert.NoError(t, err)
				_, err = packet.ReadFrame(conn)
				var timeout net.Error
				assert.ErrorAs(t, err, &timeout)
				if timeout != nil {
					assert.True(t, timeout.Timeout())
				}
				cancel()
			}
			select {
			case err = <-done:
				assert.Error(t, err)
			case <-time.After(4 * time.Second):
				cancel()
				t.Fatal("Gateway did not save stalled queue")
			}
			data, err := os.ReadFile(filepath.Join(cfg.DeliveryCfg.DumpDirectory, configuration.DUMP_FILENAME))
			assert.NoError(t, err)
			lines := bytes.Split(bytes.TrimSuffix(data, []byte("\n")), []byte("\n"))
			assert.Len(t, lines, 2)
			if len(lines) != 2 {
				return
			}
			var saved dumpRecord
			err = json.Unmarshal(lines[1], &saved)
			assert.NoError(t, err)
			assert.Equal(t, raw, saved.Raw)
			assert.True(t, saved.File)
			assert.False(t, saved.Stdout)
			cfg.DestinationsCfg.File.Rotation.MaxFileSizeBytes = 4096
			restored, err := newServer(cfg)
			assert.NoError(t, err)
			if err != nil {
				return
			}
			assert.Len(t, restored.queue, 1)
			item := restored.queue[0]
			ctx, cancelDelivery := context.WithCancel(context.Background())
			defer cancelDelivery()
			delivered := make(chan error, 1)
			go func() {
				delivered <- restored.deliver(ctx)
			}()
			select {
			case <-item.done:
			case <-time.After(2 * time.Second):
				t.Error("Restored packet was not delivered")
			}
			cancelDelivery()
			err = <-delivered
			assert.NoError(t, err)
			err = restored.file.Close()
			assert.NoError(t, err)
			data, err = os.ReadFile(filepath.Join(cfg.DestinationsCfg.File.Directory, configuration.PACKETS_FILENAME))
			assert.NoError(t, err)
			var event receivedEvent
			err = json.Unmarshal(data, &event)
			assert.NoError(t, err)
			assert.Equal(t, saved.Raw, event.Record.Raw)
			assert.Equal(t, saved.ReceivedAt, event.Record.ReceivedAt)
			assert.Equal(t, saved.Source, event.Record.Source)
			data, err = os.ReadFile(filepath.Join(cfg.DeliveryCfg.DumpDirectory, configuration.DUMP_FILENAME))
			assert.NoError(t, err)
			assert.JSONEq(t, `{"version":1}`, string(data))
		})
	}
}

func TestDumpPreservesDestinationProgress(t *testing.T) {
	root := t.TempDir()
	cfg := configuration.DefaultConfiguration()
	cfg.DeliveryCfg.DumpDirectory = filepath.Join(root, "queue")
	cfg.DestinationsCfg.File.Enabled = true
	cfg.DestinationsCfg.File.Directory = filepath.Join(root, "packets")
	err := os.MkdirAll(cfg.DeliveryCfg.DumpDirectory, 0750)
	assert.NoError(t, err)
	raw, err := hex.DecodeString("0100000b002300000001991800000001ef0000000202101500d2312b104fba3a9ed227bc35030000b200000000006a8d")
	assert.NoError(t, err)
	record, err := destination.NewRecord(time.Now(), destination.Source{}, raw)
	assert.NoError(t, err)
	s := &server{cfg: *cfg, queue: []*pendingPacket{&pendingPacket{record: record, file: true, done: make(chan struct{})}}}
	err = s.saveDump()
	assert.NoError(t, err)
	output, err := os.Create(filepath.Join(root, "stdout.ndjson"))
	assert.NoError(t, err)
	if err != nil {
		return
	}
	t.Cleanup(func() {
		err := output.Close()
		assert.NoError(t, err)
	})
	previousStdout := os.Stdout
	os.Stdout = output
	restored, err := newServer(cfg)
	os.Stdout = previousStdout
	assert.NoError(t, err)
	if err != nil {
		return
	}
	assert.Len(t, restored.queue, 1)
	item := restored.queue[0]
	assert.False(t, item.stdout)
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	done := make(chan error, 1)
	go func() {
		done <- restored.deliver(ctx)
	}()
	select {
	case <-item.done:
	case <-time.After(2 * time.Second):
		t.Error("Restored packet was not delivered")
	}
	cancel()
	err = <-done
	assert.NoError(t, err)
	err = restored.stdout.Close()
	assert.NoError(t, err)
	err = restored.file.Close()
	assert.NoError(t, err)
	data, err := os.ReadFile(output.Name())
	assert.NoError(t, err)
	assert.Empty(t, data)
	data, err = os.ReadFile(filepath.Join(cfg.DestinationsCfg.File.Directory, configuration.PACKETS_FILENAME))
	assert.NoError(t, err)
	assert.True(t, json.Valid(data))
}

func TestInvalidDumpIsPreserved(t *testing.T) {
	for _, data := range []string{"", "{", "{\"version\":99}\n", "{\"version\":1}\n{", "{\"version\":1}\n{}\n", "{\"version\":1}\n{\"pending_file\":true,\"raw\":\"bad base64\"}\n"} {
		root := t.TempDir()
		cfg := configuration.DefaultConfiguration()
		cfg.DeliveryCfg.DumpDirectory = filepath.Join(root, "queue")
		cfg.DestinationsCfg.Stdout = false
		cfg.DestinationsCfg.File.Enabled = true
		cfg.DestinationsCfg.File.Directory = filepath.Join(root, "packets")
		err := os.MkdirAll(cfg.DeliveryCfg.DumpDirectory, 0750)
		assert.NoError(t, err)
		name := filepath.Join(cfg.DeliveryCfg.DumpDirectory, configuration.DUMP_FILENAME)
		err = os.WriteFile(name, []byte(data), 0600)
		assert.NoError(t, err)
		s, err := newServer(cfg)
		assert.Error(t, err)
		assert.Nil(t, s)
		actual, err := os.ReadFile(name)
		assert.NoError(t, err)
		assert.Equal(t, data, string(actual))
		_, err = os.Stat(cfg.DestinationsCfg.File.Directory)
		assert.ErrorIs(t, err, os.ErrNotExist)
	}
}

func TestQueueBoundaries(t *testing.T) {
	cfg := configuration.DefaultConfiguration()
	cfg.DeliveryCfg.QueueCapacity = 1
	s := &server{cfg: *cfg, accepting: true, wake: make(chan struct{}, 1)}
	record := &destination.Record{}
	first, err := s.enqueue(record, nil)
	assert.NoError(t, err)
	assert.NotNil(t, first)
	second, err := s.enqueue(record, nil)
	assert.ErrorIs(t, err, ErrQueueFull)
	assert.Nil(t, second)
	assert.Len(t, s.queue, 1)
	s.accepting = false
	third, err := s.enqueue(record, nil)
	assert.ErrorIs(t, err, ErrStopped)
	assert.Nil(t, third)
}
