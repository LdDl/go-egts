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
	"github.com/stretchr/testify/assert"
)

func TestRelayDestinationsDeliverIndependently(t *testing.T) {
	cfg := configuration.DefaultConfiguration()
	cfg.DeliveryCfg.QueueCapacity = 3
	cfg.DeliveryCfg.DumpDirectory = filepath.Join(t.TempDir(), "queue")
	cfg.DestinationsCfg.Stdout = false
	listeners := make(map[string]*net.TCPListener)
	for _, id := range []string{"slow", "fast"} {
		listener, err := net.Listen("tcp", "127.0.0.1:0")
		assert.NoError(t, err)
		if err != nil {
			return
		}
		t.Cleanup(func() {
			err := listener.Close()
			assert.NoError(t, err)
		})
		listeners[id] = listener.(*net.TCPListener)
		relay := configuration.DefaultEGTSDestination()
		relay.ID = id
		relay.Enabled = true
		relay.Port = listener.Addr().(*net.TCPAddr).Port
		cfg.DestinationsCfg.EGTS = append(cfg.DestinationsCfg.EGTS, relay)
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
	err = conn.SetDeadline(time.Now().Add(5 * time.Second))
	assert.NoError(t, err)
	raw, err := hex.DecodeString("0100000b002300000001991800000001ef0000000202101500d2312b104fba3a9ed227bc35030000b200000000006a8d")
	assert.NoError(t, err)
	var frames [][]byte
	for i := 0; i < 3; i++ {
		pkg, err := packet.ReadPacket(raw)
		assert.NoError(t, err)
		pkg.PacketID = uint16(i)
		encoded, err := pkg.Encode()
		assert.NoError(t, err)
		frames = append(frames, encoded)
		_, err = conn.Write(encoded)
		assert.NoError(t, err)
		answer, err := packet.ReadFrame(conn)
		assert.NoError(t, err)
		if err != nil {
			return
		}
		response, err := packet.ReadPacket(answer)
		assert.NoError(t, err)
		assert.Zero(t, response.ServicesFrameData.(*packet.PTResponse).ProcessingResult)
	}
	peers := make(map[string]net.Conn)
	for _, id := range []string{"slow", "fast"} {
		err = listeners[id].SetDeadline(time.Now().Add(3 * time.Second))
		assert.NoError(t, err)
		peer, err := listeners[id].Accept()
		assert.NoError(t, err)
		if err != nil {
			return
		}
		peers[id] = peer
		t.Cleanup(func() {
			err := peer.Close()
			assert.NoError(t, err)
		})
		err = peer.SetDeadline(time.Now().Add(3 * time.Second))
		assert.NoError(t, err)
	}
	first, err := packet.ReadFrame(peers["slow"])
	assert.NoError(t, err)
	assert.Equal(t, frames[0], first)
	for _, expected := range frames {
		forwarded, err := packet.ReadFrame(peers["fast"])
		assert.NoError(t, err)
		if err != nil {
			return
		}
		assert.Equal(t, expected, forwarded)
		pkg, err := packet.ReadPacket(forwarded)
		assert.NoError(t, err)
		answer := pkg.PrepareAnswer(1, 2)
		encoded, err := answer.Encode()
		assert.NoError(t, err)
		_, err = peers["fast"].Write(encoded)
		assert.NoError(t, err)
	}
	assert.Eventually(t, func() bool {
		s.mu.Lock()
		defer s.mu.Unlock()
		if len(s.queue) != 3 {
			return false
		}
		for _, item := range s.queue {
			if item.egts["fast"] || !item.egts["slow"] {
				return false
			}
		}
		return true
	}, time.Second, 5*time.Millisecond)
	_, err = conn.Write(raw)
	assert.NoError(t, err)
	answer, err := packet.ReadFrame(conn)
	assert.NoError(t, err)
	response, err := packet.ReadPacket(answer)
	assert.NoError(t, err)
	assert.Equal(t, packet.EGTS_PC_NO_RES_AVAIL, response.ServicesFrameData.(*packet.PTResponse).ProcessingResult)
	cancel()
	select {
	case err = <-done:
		assert.NoError(t, err)
	case <-time.After(3 * time.Second):
		t.Fatal("Relay did not stop")
	}
	name := filepath.Join(cfg.DeliveryCfg.DumpDirectory, configuration.DUMP_FILENAME)
	data, err := os.ReadFile(name)
	assert.NoError(t, err)
	lines := bytes.Split(bytes.TrimSpace(data), []byte("\n"))
	assert.Len(t, lines, 4)
	for i, line := range lines[1:] {
		var saved dumpRecord
		err = json.Unmarshal(line, &saved)
		assert.NoError(t, err)
		assert.Equal(t, []string{"slow"}, saved.EGTSIDs)
		assert.Equal(t, frames[i], saved.Raw)
	}
	cfg.DestinationsCfg.EGTS[0], cfg.DestinationsCfg.EGTS[1] = cfg.DestinationsCfg.EGTS[1], cfg.DestinationsCfg.EGTS[0]
	added := cfg.DestinationsCfg.EGTS[0]
	added.ID = "new_destination"
	cfg.DestinationsCfg.EGTS = append(cfg.DestinationsCfg.EGTS, added)
	restored, err := newServer(cfg)
	assert.NoError(t, err)
	if err != nil {
		return
	}
	last := restored.queue[2]
	ctx, stop := context.WithCancel(context.Background())
	defer stop()
	go func() {
		done <- restored.deliver(ctx)
	}()
	err = listeners["slow"].SetDeadline(time.Now().Add(3 * time.Second))
	assert.NoError(t, err)
	peer, err := listeners["slow"].Accept()
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
	for _, expected := range frames {
		forwarded, err := packet.ReadFrame(peer)
		assert.NoError(t, err)
		if err != nil {
			return
		}
		assert.Equal(t, expected, forwarded)
		pkg, err := packet.ReadPacket(forwarded)
		assert.NoError(t, err)
		answer := pkg.PrepareAnswer(1, 2)
		encoded, err := answer.Encode()
		assert.NoError(t, err)
		_, err = peer.Write(encoded)
		assert.NoError(t, err)
	}
	select {
	case <-last.done:
	case <-time.After(3 * time.Second):
		t.Error("Restored relay packets were not delivered")
	}
	stop()
	err = <-done
	assert.NoError(t, err)
	assert.Empty(t, restored.queue)
	assert.Empty(t, restored.relays)
	err = listeners["fast"].SetDeadline(time.Now().Add(50 * time.Millisecond))
	assert.NoError(t, err)
	duplicate, err := listeners["fast"].Accept()
	assert.Error(t, err)
	assert.Nil(t, duplicate)
	data, err = os.ReadFile(name)
	assert.NoError(t, err)
	assert.JSONEq(t, `{"version":3}`, string(data))
}

func TestDeliveredWaitsForEveryRelay(t *testing.T) {
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
		relay := configuration.DefaultEGTSDestination()
		relay.ID = id
		relay.Enabled = true
		relay.Port = listener.Addr().(*net.TCPAddr).Port
		cfg.DestinationsCfg.EGTS = append(cfg.DestinationsCfg.EGTS, relay)
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
			t.Error("Relay did not stop")
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
		forwarded, err := packet.ReadFrame(peer)
		assert.NoError(t, err)
		assert.Equal(t, raw, forwarded)
		pkg, err := packet.ReadPacket(forwarded)
		assert.NoError(t, err)
		answer := pkg.PrepareAnswer(1, 2)
		encoded, err := answer.Encode()
		assert.NoError(t, err)
		_, err = peer.Write(encoded)
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
