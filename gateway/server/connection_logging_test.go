package server

import (
	"bytes"
	"context"
	"encoding/hex"
	"encoding/json"
	"net"
	"testing"
	"time"

	"github.com/LdDl/go-egts/egts/packet"
	"github.com/LdDl/go-egts/egts/subrecord"
	"github.com/LdDl/go-egts/gateway/configuration"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
	"github.com/stretchr/testify/assert"
)

type authenticationLogTestCase struct {
	name        string
	enabled     bool
	identity    bool
	credentials *subrecord.SRAuthInfo
	events      []string
	reason      string
	result      uint8
	queued      int
}

func TestAuthenticationLogs(t *testing.T) {
	identity, err := hex.DecodeString("0100020b001300010001c90800000011161600000101010500161600000095d8")
	assert.NoError(t, err)
	telemetry, err := hex.DecodeString("0100020b003a00020001162f000200111616000002021015001f946c1f36ac1ba4b983e73819ed03100000000000120c00000007000000000000000000140500020000000450a3")
	assert.NoError(t, err)
	sequence := "sequence"
	cases := []authenticationLogTestCase{
		{
			name: "telemetry after identity without credentials", enabled: true, identity: true,
			events: []string{"auth_requested", "packet_rejected"},
			reason: "authentication_required", result: packet.EGTS_PC_AUTH_DENIED,
		},
		{
			name: "telemetry without identity or credentials", enabled: true,
			events: []string{"packet_rejected"},
			reason: "authentication_required", result: packet.EGTS_PC_AUTH_DENIED,
		},
		{
			name: "valid credentials", enabled: true, identity: true,
			credentials: &subrecord.SRAuthInfo{UserName: "terminal", Password: "secret"},
			events:      []string{"auth_requested", "auth_success"}, queued: 2,
		},
		{
			name: "valid credentials without identity", enabled: true,
			credentials: &subrecord.SRAuthInfo{UserName: "terminal", Password: "secret"},
			events:      []string{"auth_success"}, queued: 2,
		},
		{
			name: "wrong password", enabled: true, identity: true,
			credentials: &subrecord.SRAuthInfo{UserName: "terminal", Password: "wrong"},
			events:      []string{"auth_requested", "packet_rejected", "auth_rejected"},
			reason:      "invalid_credentials", result: packet.EGTS_PC_AUTH_DENIED,
		},
		{
			name: "unsupported server sequence", enabled: true, identity: true,
			credentials: &subrecord.SRAuthInfo{UserName: "terminal", Password: "secret", ServerSequence: &sequence},
			events:      []string{"auth_requested", "packet_rejected", "auth_rejected"},
			reason:      "unsupported_server_sequence", result: packet.EGTS_PC_AUTH_DENIED,
		},
		{
			name: "authentication disabled", identity: true,
			events: []string{"auth_recieved"}, queued: 2,
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			var output bytes.Buffer
			previousLogger := log.Logger
			log.Logger = zerolog.New(&output)
			t.Cleanup(func() {
				log.Logger = previousLogger
			})
			cfg := configuration.DefaultConfiguration()
			cfg.AuthCfg = configuration.AuthConf{Enabled: tc.enabled, Password: "secret"}
			s := &server{cfg: *cfg, accepting: true, wake: make(chan struct{}, 1)}
			client, peer := net.Pipe()
			done := make(chan struct{})
			var connectionErr error
			go func() {
				connectionErr = s.handleConnection(context.Background(), peer)
				close(done)
			}()
			t.Cleanup(func() {
				err := client.Close()
				assert.NoError(t, err)
				err = peer.Close()
				assert.NoError(t, err)
				select {
				case <-done:
				case <-time.After(time.Second):
					t.Error("Connection handler did not stop")
				}
			})
			err := client.SetDeadline(time.Now().Add(3 * time.Second))
			assert.NoError(t, err)
			var requests [][]byte
			authPackets := 0
			if tc.identity {
				requests = append(requests, identity)
				authPackets++
			}
			if tc.credentials != nil {
				pkg, err := packet.ReadPacket(identity)
				assert.NoError(t, err)
				if err != nil {
					return
				}
				services := pkg.ServicesFrameData.(*packet.ServicesFrameData)
				(*services)[0].RecordsData = packet.RecordsData{&packet.RecordData{
					SubrecordType: packet.AuthInfo, SubrecordLength: tc.credentials.Len(), SubrecordData: tc.credentials,
				}}
				(*services)[0].RecordLength = (*services)[0].RecordsData.Len()
				pkg.FrameDataLength = services.Len()
				encoded, err := pkg.Encode()
				assert.NoError(t, err)
				if err != nil {
					return
				}
				requests = append(requests, encoded)
				authPackets++
			}
			if tc.credentials == nil || tc.result == packet.EGTS_PC_OK {
				requests = append(requests, telemetry)
			}
			for i, request := range requests {
				_, err = client.Write(request)
				assert.NoError(t, err)
				if err != nil {
					return
				}
				raw, err := packet.ReadFrame(client)
				assert.NoError(t, err)
				if err != nil {
					return
				}
				response, err := packet.ReadPacket(raw)
				assert.NoError(t, err)
				if err != nil {
					return
				}
				expected := packet.EGTS_PC_OK
				if i == len(requests)-1 {
					expected = tc.result
				}
				assert.Equal(t, packet.EGTS_PT_RESPONSE, response.PacketType)
				assert.Equal(t, expected, response.ServicesFrameData.(*packet.PTResponse).ProcessingResult)
				if i < authPackets {
					raw, err = packet.ReadFrame(client)
					assert.NoError(t, err)
					if err != nil {
						return
					}
					response, err = packet.ReadPacket(raw)
					assert.NoError(t, err)
					assert.Equal(t, packet.EGTS_PT_APPDATA, response.PacketType)
				}
			}
			err = client.Close()
			assert.NoError(t, err)
			select {
			case <-done:
			case <-time.After(time.Second):
				t.Fatal("Connection handler did not stop")
			}
			assert.NoError(t, connectionErr)
			assert.Len(t, s.queue, tc.queued)
			var events []string
			lines := bytes.Split(bytes.TrimSuffix(output.Bytes(), []byte("\n")), []byte("\n"))
			for _, line := range lines {
				var entry map[string]interface{}
				err = json.Unmarshal(line, &entry)
				assert.NoError(t, err)
				if err != nil {
					return
				}
				event, ok := entry["event"].(string)
				assert.True(t, ok)
				events = append(events, event)
				assert.Equal(t, "server", entry["scope"])
				assert.Equal(t, "pipe", entry["remote_address"])
				assert.Contains(t, entry, "terminal_id")
				if tc.identity {
					assert.Equal(t, float64(5654), entry["terminal_id"])
				} else {
					assert.Nil(t, entry["terminal_id"])
				}
				assert.NotContains(t, entry, "level")
				assert.NotContains(t, entry, "raw")
				assert.NotContains(t, entry, "packet")
				if event == "packet_rejected" {
					assert.Equal(t, float64(tc.result), entry["result"])
					assert.Equal(t, tc.reason, entry["reason"])
					assert.Equal(t, true, entry["connection_closing"])
					packetID := float64(2)
					if tc.credentials != nil {
						packetID = 1
					}
					assert.Equal(t, packetID, entry["packet_id"])
				} else {
					assert.Equal(t, tc.enabled, entry["auth_enabled"])
					assert.Equal(t, event == "auth_success" || !tc.enabled, entry["authorized"])
				}
			}
			assert.Equal(t, tc.events, events)
			assert.NotContains(t, output.String(), "secret")
			assert.NotContains(t, output.String(), "wrong")
		})
	}
}
