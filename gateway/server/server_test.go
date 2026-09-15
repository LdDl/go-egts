package server

import (
	"bytes"
	"context"
	"encoding/hex"
	"encoding/json"
	"io"
	"net"
	"os"
	"path/filepath"
	"strconv"
	"testing"
	"time"

	"github.com/LdDl/go-egts/crc"
	"github.com/LdDl/go-egts/egts/packet"
	"github.com/LdDl/go-egts/egts/subrecord"
	"github.com/LdDl/go-egts/gateway/configuration"
	"github.com/LdDl/go-egts/gateway/destination"
	"github.com/stretchr/testify/assert"
)

type connectionTestCase struct {
	name    string
	chunks  [][]byte
	codes   []uint8
	records int
}

type receivedRecord struct {
	ReceivedAt time.Time          `json:"received_at"`
	Source     destination.Source `json:"source"`
	Raw        []byte             `json:"raw"`
}

type receivedEvent struct {
	Record receivedRecord `json:"record"`
}

func TestGatewayConnections(t *testing.T) {
	telemetry, err := hex.DecodeString("0100000b002300000001991800000001ef0000000202101500d2312b104fba3a9ed227bc35030000b200000000006a8d")
	assert.NoError(t, err)
	confirmation, err := hex.DecodeString("0100030b001000000000b3000000060000005802020003000000002ec1")
	assert.NoError(t, err)
	corrupted := append([]byte(nil), telemetry...)
	corrupted[len(corrupted)-1] ^= 1
	empty := []byte{1, 0, 0, 11, 0, 0, 0, 7, 0, 1, 0}
	empty[10] = byte(crc.Crc(8, empty[:10]))
	cases := []connectionTestCase{
		{name: "telemetry without authentication", chunks: [][]byte{telemetry}, codes: []uint8{0}, records: 1},
		{name: "fragmented packet", chunks: [][]byte{telemetry[:1], telemetry[1:10], telemetry[10:11], telemetry[11:]}, codes: []uint8{0}, records: 1},
		{name: "coalesced packets", chunks: [][]byte{bytes.Repeat(telemetry, 2)}, codes: []uint8{0, 0}, records: 2},
		{name: "no confirmation loop", chunks: [][]byte{append(confirmation, telemetry...)}, codes: []uint8{0}, records: 1},
		{name: "bad CRC then good packet", chunks: [][]byte{append(corrupted, telemetry...)}, codes: []uint8{packet.EGTS_PC_DATACRC_ERROR, 0}, records: 1},
		{name: "empty appdata", chunks: [][]byte{empty}, codes: []uint8{0}, records: 1},
	}
	for _, mode := range []string{"queued", "delivered"} {
		for _, tc := range cases {
			t.Run(mode+"/"+tc.name, func(t *testing.T) {
				root := t.TempDir()
				cfg := configuration.DefaultConfiguration()
				cfg.DeliveryCfg.AckMode = mode
				cfg.DeliveryCfg.DumpDirectory = filepath.Join(root, "queue")
				cfg.DestinationsCfg.Stdout = false
				cfg.DestinationsCfg.File.Enabled = true
				cfg.DestinationsCfg.File.Directory = filepath.Join(root, "packets")
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
				err = conn.SetDeadline(time.Now().Add(2 * time.Second))
				assert.NoError(t, err)
				for _, chunk := range tc.chunks {
					n, err := conn.Write(chunk)
					assert.NoError(t, err)
					assert.Equal(t, len(chunk), n)
				}
				for i, code := range tc.codes {
					raw, err := readPacket(conn)
					assert.NoError(t, err)
					if err != nil {
						return
					}
					answer, err := packet.ReadPacket(raw)
					assert.NoError(t, err)
					assert.Equal(t, packet.EGTS_PT_RESPONSE, answer.PacketType)
					assert.Equal(t, uint16(i+1), answer.PacketID)
					result := answer.ServicesFrameData.(*packet.PTResponse)
					assert.Equal(t, code, result.ProcessingResult)
					if mode == "delivered" && code == 0 {
						data, err := os.ReadFile(filepath.Join(cfg.DestinationsCfg.File.Directory, configuration.PACKETS_FILENAME))
						assert.NoError(t, err)
						assert.NotEmpty(t, data)
					}
				}
				assert.Eventually(t, func() bool {
					s.mu.Lock()
					defer s.mu.Unlock()
					return len(s.queue) == 0
				}, 2*time.Second, 5*time.Millisecond)
				data, err := os.ReadFile(filepath.Join(cfg.DestinationsCfg.File.Directory, configuration.PACKETS_FILENAME))
				assert.NoError(t, err)
				lines := bytes.Split(bytes.TrimSuffix(data, []byte("\n")), []byte("\n"))
				assert.Len(t, lines, tc.records)
				for _, line := range lines {
					var event receivedEvent
					err = json.Unmarshal(line, &event)
					assert.NoError(t, err)
					assert.Nil(t, event.Record.Source.TerminalID)
					assert.Equal(t, conn.LocalAddr().String(), event.Record.Source.RemoteAddress)
					assert.False(t, event.Record.ReceivedAt.IsZero())
					_, err = packet.ReadPacket(event.Record.Raw)
					assert.NoError(t, err)
				}
			})
		}
	}
}

func TestGatewayAuthentication(t *testing.T) {
	for _, password := range []string{"secret", "wrong"} {
		t.Run(password, func(t *testing.T) {
			root := t.TempDir()
			cfg := configuration.DefaultConfiguration()
			cfg.AuthCfg = configuration.AuthConf{Enabled: true, Password: "secret"}
			cfg.DeliveryCfg.AckMode = "delivered"
			cfg.DeliveryCfg.DumpDirectory = filepath.Join(root, "queue")
			cfg.DestinationsCfg.Stdout = false
			cfg.DestinationsCfg.File.Enabled = true
			cfg.DestinationsCfg.File.Directory = filepath.Join(root, "packets")
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
			conn, err := net.Dial("tcp", listener.Addr().String())
			assert.NoError(t, err)
			if err != nil {
				return
			}
			t.Cleanup(func() {
				err := conn.Close()
				assert.NoError(t, err)
			})
			err = conn.SetDeadline(time.Now().Add(2 * time.Second))
			assert.NoError(t, err)
			identity, err := hex.DecodeString("0100020b0020000000014f1900000010010101160000000000523836363130343032393639303030380004417f")
			assert.NoError(t, err)
			_, err = conn.Write(identity)
			assert.NoError(t, err)
			for _, kind := range []uint8{packet.EGTS_PT_RESPONSE, packet.EGTS_PT_APPDATA} {
				raw, err := readPacket(conn)
				assert.NoError(t, err)
				if err != nil {
					return
				}
				response, err := packet.ReadPacket(raw)
				assert.NoError(t, err)
				assert.Equal(t, kind, response.PacketType)
				if kind == packet.EGTS_PT_APPDATA {
					services := response.ServicesFrameData.(*packet.ServicesFrameData)
					assert.IsType(t, &subrecord.SRAuthParams{}, (*services)[0].RecordsData[0].SubrecordData)
				} else {
					assert.Zero(t, response.ServicesFrameData.(*packet.PTResponse).ProcessingResult)
				}
			}
			pkg, err := packet.ReadPacket(identity)
			assert.NoError(t, err)
			services := pkg.ServicesFrameData.(*packet.ServicesFrameData)
			auth := &subrecord.SRAuthInfo{UserName: "terminal", Password: password}
			(*services)[0].RecordsData = packet.RecordsData{&packet.RecordData{SubrecordType: packet.AuthInfo, SubrecordLength: auth.Len(), SubrecordData: auth}}
			(*services)[0].RecordLength = (*services)[0].RecordsData.Len()
			pkg.FrameDataLength = services.Len()
			raw, err := pkg.Encode()
			assert.NoError(t, err)
			_, err = conn.Write(raw)
			assert.NoError(t, err)
			expected := packet.EGTS_PC_OK
			if password == "wrong" {
				expected = packet.EGTS_PC_AUTH_DENIED
			}
			for _, kind := range []uint8{packet.EGTS_PT_RESPONSE, packet.EGTS_PT_APPDATA} {
				raw, err := readPacket(conn)
				assert.NoError(t, err)
				if err != nil {
					return
				}
				response, err := packet.ReadPacket(raw)
				assert.NoError(t, err)
				assert.Equal(t, kind, response.PacketType)
				if kind == packet.EGTS_PT_RESPONSE {
					assert.Equal(t, expected, response.ServicesFrameData.(*packet.PTResponse).ProcessingResult)
				} else {
					services := response.ServicesFrameData.(*packet.ServicesFrameData)
					assert.Equal(t, expected, (*services)[0].RecordsData[0].SubrecordData.(*subrecord.SRResultCode).RCD)
				}
			}
			if password == "wrong" {
				_, err = readPacket(conn)
				assert.ErrorIs(t, err, io.EOF)
			} else {
				telemetry, err := hex.DecodeString("0100000b002300000001991800000001ef0000000202101500d2312b104fba3a9ed227bc35030000b200000000006a8d")
				assert.NoError(t, err)
				_, err = conn.Write(telemetry)
				assert.NoError(t, err)
				raw, err = readPacket(conn)
				assert.NoError(t, err)
				response, err := packet.ReadPacket(raw)
				assert.NoError(t, err)
				assert.Zero(t, response.ServicesFrameData.(*packet.PTResponse).ProcessingResult)
			}
			data, err := os.ReadFile(filepath.Join(cfg.DestinationsCfg.File.Directory, configuration.PACKETS_FILENAME))
			assert.NoError(t, err)
			if password == "wrong" {
				assert.Empty(t, data)
			} else {
				lines := bytes.Split(bytes.TrimSuffix(data, []byte("\n")), []byte("\n"))
				assert.Len(t, lines, 2)
				for _, line := range lines {
					var event receivedEvent
					err = json.Unmarshal(line, &event)
					assert.NoError(t, err)
					assert.NotNil(t, event.Record.Source.TerminalID)
					assert.Equal(t, uint32(0), *event.Record.Source.TerminalID)
				}
			}
		})
	}
}

func TestReadPacketErrors(t *testing.T) {
	raw, err := hex.DecodeString("0100000b002300000001991800000001ef0000000202101500d2312b104fba3a9ed227bc35030000b200000000006a8d")
	assert.NoError(t, err)
	for length := 0; length < len(raw); length++ {
		data, err := readPacket(bytes.NewReader(raw[:length]))
		assert.Error(t, err)
		assert.Nil(t, data)
	}
	for _, offset := range []int{0, 2, 3, 10} {
		corrupted := append([]byte(nil), raw...)
		corrupted[offset] ^= 1
		if offset == 2 {
			corrupted[offset] ^= 0x21
		}
		data, err := readPacket(bytes.NewReader(corrupted))
		assert.Error(t, err)
		assert.Nil(t, data)
	}
}

func TestAuthenticationRequired(t *testing.T) {
	cfg := configuration.DefaultConfiguration()
	cfg.AuthCfg = configuration.AuthConf{Enabled: true, Password: "secret"}
	s := &server{cfg: *cfg, accepting: true, wake: make(chan struct{}, 1)}
	client, peer := net.Pipe()
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	done := make(chan error, 1)
	go func() {
		done <- s.handleConnection(ctx, peer)
	}()
	t.Cleanup(func() {
		err := client.Close()
		assert.NoError(t, err)
		err = peer.Close()
		assert.NoError(t, err)
	})
	err := client.SetDeadline(time.Now().Add(time.Second))
	assert.NoError(t, err)
	raw, err := hex.DecodeString("0100000b002300000001991800000001ef0000000202101500d2312b104fba3a9ed227bc35030000b200000000006a8d")
	assert.NoError(t, err)
	_, err = client.Write(raw)
	assert.NoError(t, err)
	answer, err := readPacket(client)
	assert.NoError(t, err)
	if err != nil {
		return
	}
	response, err := packet.ReadPacket(answer)
	assert.NoError(t, err)
	assert.Equal(t, packet.EGTS_PC_AUTH_DENIED, response.ServicesFrameData.(*packet.PTResponse).ProcessingResult)
	select {
	case err = <-done:
		assert.NoError(t, err)
	case <-time.After(time.Second):
		t.Error("Unauthenticated connection was not rejected")
	}
	assert.Empty(t, s.queue)
}

func TestRunOccupiedPort(t *testing.T) {
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	assert.NoError(t, err)
	if err != nil {
		return
	}
	t.Cleanup(func() {
		err := listener.Close()
		assert.NoError(t, err)
	})
	host, portString, err := net.SplitHostPort(listener.Addr().String())
	assert.NoError(t, err)
	port, err := strconv.Atoi(portString)
	assert.NoError(t, err)
	cfg := configuration.DefaultConfiguration()
	cfg.ServerCfg = configuration.ServerConf{Host: host, Port: port}
	cfg.DeliveryCfg.DumpDirectory = filepath.Join(t.TempDir(), "queue")
	err = Run(context.Background(), cfg)
	assert.ErrorContains(t, err, "Can't listen for EGTS")
	_, err = os.Stat(cfg.DeliveryCfg.DumpDirectory)
	assert.ErrorIs(t, err, os.ErrNotExist)
}
