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
	"github.com/LdDl/go-egts/egts/subrecord"
	"github.com/LdDl/go-egts/gateway/configuration"
	"github.com/LdDl/go-egts/gateway/destination"
	"github.com/stretchr/testify/assert"
)

type relayAuthenticationTestCase struct {
	name     string
	auth     bool
	identity bool
}

func TestGatewayRelay(t *testing.T) {
	cases := []relayAuthenticationTestCase{
		{name: "without authentication", identity: true},
		{name: "separate passwords", auth: true, identity: true},
		{name: "authentication without identity", auth: true},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			root := t.TempDir()
			upstream := configuration.DefaultConfiguration()
			upstream.AuthCfg = configuration.AuthConf{Enabled: tc.auth, Password: "destination-password"}
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
			remote, err := net.Listen("tcp", "127.0.0.1:0")
			assert.NoError(t, err)
			if err != nil {
				return
			}
			ctx, cancel := context.WithCancel(context.Background())
			collected := make(chan error, 1)
			go func() {
				collected <- collector.serve(ctx, remote)
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
			cfg.AuthCfg = configuration.AuthConf{Enabled: tc.auth, Password: "incoming-password"}
			cfg.DeliveryCfg.AckMode = "delivered"
			cfg.DeliveryCfg.DumpDirectory = filepath.Join(root, "relay-queue")
			cfg.DestinationsCfg.Stdout = false
			if len(cfg.DestinationsCfg.EGTS) == 0 {
				cfg.DestinationsCfg.EGTS = []configuration.EGTSDestinationConf{configuration.DefaultEGTSDestination()}
				cfg.DestinationsCfg.EGTS[0].ID = "primary"
			}
			cfg.DestinationsCfg.EGTS[0].Enabled = true
			cfg.DestinationsCfg.EGTS[0].Port = remote.Addr().(*net.TCPAddr).Port
			cfg.DestinationsCfg.EGTS[0].Auth = configuration.EGTSAuthConf{Enabled: tc.auth, UserName: "relay", Password: "destination-password"}
			relay, err := newServer(cfg)
			assert.NoError(t, err)
			if err != nil {
				return
			}
			listener, err := net.Listen("tcp", "127.0.0.1:0")
			assert.NoError(t, err)
			if err != nil {
				return
			}
			relayCtx, stopRelay := context.WithCancel(context.Background())
			relayed := make(chan error, 1)
			go func() {
				relayed <- relay.serve(relayCtx, listener)
			}()
			t.Cleanup(func() {
				stopRelay()
				select {
				case err := <-relayed:
					assert.NoError(t, err)
				case <-time.After(3 * time.Second):
					t.Error("Relay gateway did not stop")
				}
			})
			telemetry, err := hex.DecodeString("0100000b002300000001991800000001ef0000000202101500d2312b104fba3a9ed227bc35030000b200000000006a8d")
			assert.NoError(t, err)
			for _, terminalID := range []uint32{101, 202} {
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
				identity, err := hex.DecodeString("0100020b0020000000014f1900000010010101160000000000523836363130343032393639303030380004417f")
				assert.NoError(t, err)
				control, err := packet.ReadPacket(identity)
				assert.NoError(t, err)
				services := control.ServicesFrameData.(*packet.ServicesFrameData)
				(*services)[0].RecordsData[0].SubrecordData.(*subrecord.SRTermIdentity).TerminalIdentifier = terminalID
				var frames [][]byte
				if tc.identity {
					identity, err = control.Encode()
					assert.NoError(t, err)
					frames = append(frames, identity)
				}
				if tc.auth {
					auth := &subrecord.SRAuthInfo{UserName: "terminal", Password: "incoming-password"}
					(*services)[0].RecordsData = packet.RecordsData{&packet.RecordData{SubrecordType: packet.AuthInfo, SubrecordLength: auth.Len(), SubrecordData: auth}}
					(*services)[0].RecordLength = (*services)[0].RecordsData.Len()
					control.FrameDataLength = services.Len()
					raw, err := control.Encode()
					assert.NoError(t, err)
					frames = append(frames, raw)
				}
				for _, raw := range frames {
					_, err = conn.Write(raw)
					assert.NoError(t, err)
					for _, kind := range []uint8{packet.EGTS_PT_RESPONSE, packet.EGTS_PT_APPDATA} {
						answer, err := packet.ReadFrame(conn)
						assert.NoError(t, err)
						if err != nil {
							return
						}
						response, err := packet.ReadPacket(answer)
						assert.NoError(t, err)
						assert.Equal(t, kind, response.PacketType)
						if kind == packet.EGTS_PT_RESPONSE {
							assert.Zero(t, response.ServicesFrameData.(*packet.PTResponse).ProcessingResult)
						}
					}
				}
				for i := 0; i < 2; i++ {
					_, err = conn.Write(telemetry)
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
			}
			data, err := os.ReadFile(filepath.Join(upstream.DestinationsCfg.File.Directory, configuration.PACKETS_FILENAME))
			assert.NoError(t, err)
			lines := bytes.Split(bytes.TrimSuffix(data, []byte("\n")), []byte("\n"))
			assert.Len(t, lines, 6)
			connections := make(map[string]int)
			identities := make(map[uint32]int)
			controls := 0
			for _, line := range lines {
				var event receivedEvent
				err = json.Unmarshal(line, &event)
				assert.NoError(t, err)
				pkg, err := packet.ReadPacket(event.Record.Raw)
				assert.NoError(t, err)
				services := pkg.ServicesFrameData.(*packet.ServicesFrameData)
				if (*services)[0].SourceServiceType == packet.SERVICE_AUTH {
					controls++
					if tc.auth {
						auth := (*services)[0].RecordsData[0].SubrecordData.(*subrecord.SRAuthInfo)
						assert.Equal(t, "relay", auth.UserName)
						assert.Equal(t, "destination-password", auth.Password)
					}
					continue
				}
				assert.Equal(t, telemetry, event.Record.Raw)
				connections[event.Record.Source.RemoteAddress]++
				if tc.identity {
					assert.NotNil(t, event.Record.Source.TerminalID)
					if event.Record.Source.TerminalID != nil {
						identities[*event.Record.Source.TerminalID]++
					}
				} else {
					assert.Nil(t, event.Record.Source.TerminalID)
				}
			}
			assert.Equal(t, 2, controls)
			assert.Len(t, connections, 2)
			for _, count := range connections {
				assert.Equal(t, 2, count)
			}
			if tc.identity {
				assert.Equal(t, map[uint32]int{101: 2, 202: 2}, identities)
			}
		})
	}
}

func TestRelayRejectsMixedCredentials(t *testing.T) {
	raw, err := hex.DecodeString("0100000b002300000001991800000001ef0000000202101500d2312b104fba3a9ed227bc35030000b200000000006a8d")
	assert.NoError(t, err)
	pkg, err := packet.ReadPacket(raw)
	assert.NoError(t, err)
	services := pkg.ServicesFrameData.(*packet.ServicesFrameData)
	auth := &subrecord.SRAuthInfo{Password: "incoming-password"}
	*services = append(*services, &packet.ServiceDataRecord{
		SourceServiceType: packet.SERVICE_AUTH, RecipientServiceType: packet.SERVICE_AUTH,
		RecordsData: packet.RecordsData{&packet.RecordData{SubrecordType: packet.AuthInfo, SubrecordData: auth}},
	})
	cfg := configuration.DefaultConfiguration()
	if len(cfg.DestinationsCfg.EGTS) == 0 {
		cfg.DestinationsCfg.EGTS = []configuration.EGTSDestinationConf{configuration.DefaultEGTSDestination()}
		cfg.DestinationsCfg.EGTS[0].ID = "primary"
	}
	cfg.DestinationsCfg.EGTS[0].Enabled = true
	s := &server{cfg: *cfg, accepting: true, wake: make(chan struct{}, 1)}
	item, err := s.enqueue(&destination.Record{Packet: pkg}, nil)
	assert.ErrorIs(t, err, ErrRelayCredentials)
	assert.Nil(t, item)
	assert.Empty(t, s.queue)
	*services = (*services)[1:]
	item, err = s.enqueue(&destination.Record{Packet: pkg}, nil)
	assert.NoError(t, err)
	assert.Empty(t, item.egts)
}
