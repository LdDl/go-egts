package server

import (
	"context"
	"encoding/hex"
	"encoding/json"
	"net"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/LdDl/go-egts/egts/packet"
	"github.com/LdDl/go-egts/egts/subrecord"
	"github.com/LdDl/go-egts/gateway/configuration"
	"github.com/LdDl/go-egts/gateway/destination"
	"github.com/stretchr/testify/assert"
)

const (
	v2AuthHex     = "0100000b006a000a00016b53006400071122334455667788785634122a0000000101010b008877665544332211003032031e003132333435363738394142434445464748010000000300000045585452410c2100004131323342430000000000000000000000000000000000000000000000000000282c"
	v2PositionHex = "0100000b003e000b00018f27006500071122334455667788785634122a00000002021024002a000000ffffffffffffffff837bc02c563412a5042143057856341285ff57e80300feffc582"
)

func v2GatewayConfig(t *testing.T) *configuration.Configuration {
	t.Helper()
	cfg := configuration.DefaultConfiguration()
	root := t.TempDir()
	cfg.ServerCfg.ServiceVersion = 2
	cfg.DestinationsCfg.Stdout = false
	cfg.DestinationsCfg.File.Enabled = true
	cfg.DestinationsCfg.File.Directory = filepath.Join(root, "packets")
	cfg.DeliveryCfg.DumpDirectory = filepath.Join(root, "queue")
	cfg.DeliveryCfg.AckMode = "delivered"
	return cfg
}

func startV2Gateway(t *testing.T, cfg *configuration.Configuration) (net.Listener, *server) {
	t.Helper()
	s, err := newServer(cfg)
	if !assert.NoError(t, err) {
		return nil, nil
	}
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if !assert.NoError(t, err) {
		return nil, nil
	}
	ctx, cancel := context.WithCancel(context.Background())
	done := make(chan error, 1)
	go func() { done <- s.serve(ctx, listener) }()
	t.Cleanup(func() {
		cancel()
		select {
		case err := <-done:
			if !assert.NoError(t, err) {
				return
			}
		case <-time.After(3 * time.Second):
			t.Error("Gateway did not stop")
		}
	})
	return listener, s
}

func TestV2GatewayAuthenticationAndRelay(t *testing.T) {
	auth, err := hex.DecodeString(v2AuthHex)
	if !assert.NoError(t, err) {
		return
	}
	position, err := hex.DecodeString(v2PositionHex)
	if !assert.NoError(t, err) {
		return
	}
	collectorCfg := v2GatewayConfig(t)
	collectorCfg.AuthCfg.Enabled = true
	collectorCfg.AuthCfg.Password = "upstream"
	collector, _ := startV2Gateway(t, collectorCfg)
	if collector == nil {
		return
	}
	cfg := v2GatewayConfig(t)
	cfg.AuthCfg.Enabled = true
	cfg.AuthCfg.Password = "terminal"
	upstream := configuration.DefaultEGTSDestination()
	upstream.ID = "v2"
	upstream.Enabled = true
	upstream.Port = collector.Addr().(*net.TCPAddr).Port
	upstream.Auth.Enabled = true
	upstream.Auth.Password = "upstream"
	cfg.DestinationsCfg.EGTS = []configuration.EGTSDestinationConf{upstream}
	listener, _ := startV2Gateway(t, cfg)
	if listener == nil {
		return
	}
	conn, err := net.Dial("tcp", listener.Addr().String())
	if !assert.NoError(t, err) {
		return
	}
	defer conn.Close()
	if !assert.NoError(t, conn.SetDeadline(time.Now().Add(5*time.Second))) {
		return
	}
	_, err = conn.Write(auth)
	if !assert.NoError(t, err) {
		return
	}
	raw, err := packet.ReadFrame(conn)
	if !assert.NoError(t, err) {
		return
	}
	ack, err := packet.ReadPacketVersion(raw, 2)
	if !assert.NoError(t, err) {
		return
	}
	assert.Equal(t, uint8(0), ack.ServicesFrameData.(*packet.PTResponse).ProcessingResult)
	raw, err = packet.ReadFrame(conn)
	if !assert.NoError(t, err) {
		return
	}
	challenge, err := packet.ReadPacketVersion(raw, 2)
	if !assert.NoError(t, err) {
		return
	}
	services, ok := challenge.ServicesFrameData.(*packet.ServicesFrameData)
	if !assert.True(t, ok) {
		return
	}
	assert.IsType(t, &subrecord.SRAuthParams{}, (*services)[0].RecordsData[0].SubrecordData)
	credentials := &subrecord.SRAuthInfo{Password: "terminal"}
	r := (*services)[0]
	r.RecordsData = packet.RecordsData{&packet.RecordData{SubrecordType: packet.AuthInfo, SubrecordLength: credentials.Len(), SubrecordData: credentials}}
	r.RecordLength = r.RecordsData.Len()
	challenge.FrameDataLength = services.Len()
	b, err := challenge.Encode()
	if !assert.NoError(t, err) {
		return
	}
	_, err = conn.Write(b)
	if !assert.NoError(t, err) {
		return
	}
	raw, err = packet.ReadFrame(conn)
	if !assert.NoError(t, err) {
		return
	}
	ack, err = packet.ReadPacketVersion(raw, 2)
	if !assert.NoError(t, err) {
		return
	}
	assert.Zero(t, ack.ServicesFrameData.(*packet.PTResponse).ProcessingResult)
	raw, err = packet.ReadFrame(conn)
	if !assert.NoError(t, err) {
		return
	}
	result, err := packet.ReadPacketVersion(raw, 2)
	if !assert.NoError(t, err) {
		return
	}
	services = result.ServicesFrameData.(*packet.ServicesFrameData)
	assert.Zero(t, (*services)[0].RecordsData[0].SubrecordData.(*subrecord.SRResultCode).RCD)
	_, err = conn.Write(position)
	if !assert.NoError(t, err) {
		return
	}
	raw, err = packet.ReadFrame(conn)
	if !assert.NoError(t, err) {
		return
	}
	ack, err = packet.ReadPacketVersion(raw, 2)
	if !assert.NoError(t, err) {
		return
	}
	assert.Zero(t, ack.ServicesFrameData.(*packet.PTResponse).ProcessingResult)
	// Delivery acknowledgement is sent after the collector has persisted the position.
	files, err := filepath.Glob(filepath.Join(collectorCfg.DestinationsCfg.File.Directory, "*"))
	if !assert.NoError(t, err) {
		return
	}
	found := false
	for _, name := range files {
		data, err := os.ReadFile(name)
		if !assert.NoError(t, err) {
			return
		}
		for _, line := range strings.Split(strings.TrimSpace(string(data)), "\n") {
			var event receivedEvent
			if !assert.NoError(t, json.Unmarshal([]byte(line), &event)) {
				return
			}
			if string(event.Record.Raw) == string(position) {
				found = true
				assert.Equal(t, uint8(2), event.Record.Source.ServiceVersion)
				if !assert.NotNil(t, event.Record.Source.TerminalIDV2) {
					return
				}
				assert.Equal(t, uint64(0x1122334455667788), *event.Record.Source.TerminalIDV2)
			}
		}
	}
	assert.True(t, found, "collector did not receive V2 telemetry")
}

func TestV2DumpPreservesServiceVersion(t *testing.T) {
	position, err := hex.DecodeString(v2PositionHex)
	if !assert.NoError(t, err) {
		return
	}
	cfg := v2GatewayConfig(t)
	s, err := newServer(cfg)
	if !assert.NoError(t, err) {
		return
	}
	s.accepting = true
	id := uint64(0x1122334455667788)
	record, err := destination.NewRecord(time.Now(), destination.Source{ServiceVersion: 2, TerminalIDV2: &id}, position)
	if !assert.NoError(t, err) {
		return
	}
	_, err = s.enqueue(record, nil)
	if !assert.NoError(t, err) {
		return
	}
	if !assert.NoError(t, s.saveDump()) {
		return
	}
	if !assert.NoError(t, s.file.Close()) {
		return
	}
	saved, err := os.ReadFile(filepath.Join(cfg.DeliveryCfg.DumpDirectory, configuration.DUMP_FILENAME))
	if !assert.NoError(t, err) {
		return
	}
	assert.Contains(t, string(saved), `"version":6`)
	// Restoring does not depend on the current listener version.
	cfg.ServerCfg.ServiceVersion = 0
	restored, err := newServer(cfg)
	if !assert.NoError(t, err) {
		return
	}
	defer restored.file.Close()
	if !assert.Len(t, restored.queue, 1) {
		return
	}
	p := restored.queue[0].record.Packet
	assert.Equal(t, uint8(2), p.ServiceVersion)
	r := (*p.ServicesFrameData.(*packet.ServicesFrameData))[0]
	assert.Equal(t, uint64(0x8877665544332211), r.ObjectIdentifierV2)
	assert.IsType(t, &subrecord.SRPosDataV2{}, r.RecordsData[0].SubrecordData)
}
