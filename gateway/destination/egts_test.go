package destination

import (
	"context"
	"encoding/hex"
	"net"
	"testing"
	"time"

	"github.com/LdDl/go-egts/egts/packet"
	"github.com/LdDl/go-egts/egts/subrecord"
	"github.com/LdDl/go-egts/gateway/configuration"
	"github.com/stretchr/testify/assert"
)

type egtsConfirmationTestCase struct {
	name string
	mode string
	fail bool
}

func TestEGTSConfirmations(t *testing.T) {
	cases := []egtsConfirmationTestCase{
		{name: "combined confirmation", mode: "combined"},
		{name: "separate confirmation", mode: "separate"},
		{name: "records before transport", mode: "reversed"},
		{name: "wrong packet then correct", mode: "wrong then correct"},
		{name: "transport rejected", mode: "transport rejected", fail: true},
		{name: "record rejected", mode: "record rejected", fail: true},
		{name: "missing record confirmation", mode: "transport only", fail: true},
		{name: "wrong packet confirmation", mode: "wrong packet", fail: true},
		{name: "wrong record confirmation", mode: "wrong record", fail: true},
		{name: "wrong service confirmation", mode: "wrong service", fail: true},
		{name: "record still in progress", mode: "progress", fail: true},
		{name: "invalid checksum", mode: "checksum", fail: true},
		{name: "closed connection", mode: "closed", fail: true},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			raw, err := hex.DecodeString("0100000b002300000001991800000001ef0000000202101500d2312b104fba3a9ed227bc35030000b200000000006a8d")
			assert.NoError(t, err)
			record, err := NewRecord(time.Now(), Source{}, raw)
			assert.NoError(t, err)
			if err != nil {
				return
			}
			client, peer := net.Pipe()
			cfg := configuration.DefaultConfiguration()
			cfg.DestinationsCfg.EGTS.Enabled = true
			writer, err := PrepareEGTS(cfg)
			assert.NoError(t, err)
			writer.conn = client
			t.Cleanup(func() {
				err := writer.Close()
				assert.NoError(t, err)
				err = peer.Close()
				assert.NoError(t, err)
			})
			ctx, cancel := context.WithTimeout(context.Background(), 150*time.Millisecond)
			defer cancel()
			done := make(chan error, 1)
			go func() {
				done <- writer.WriteContext(ctx, record, nil)
			}()
			err = peer.SetDeadline(time.Now().Add(time.Second))
			assert.NoError(t, err)
			actual, err := packet.ReadFrame(peer)
			assert.NoError(t, err)
			assert.Equal(t, raw, actual)
			answer := record.Packet.PrepareAnswer(10, 20)
			confirmation := answer.ServicesFrameData.(*packet.PTResponse)
			services := confirmation.SDR.(*packet.ServicesFrameData)
			response := (*services)[0].RecordsData[0].SubrecordData.(*subrecord.SRRecordResponse)
			var frames []packet.Packet
			switch tc.mode {
			case "transport rejected":
				confirmation.ProcessingResult = packet.EGTS_PC_NO_RES_AVAIL
			case "record rejected":
				response.RecordStatus = packet.EGTS_PC_IO_ERROR
			case "progress":
				response.RecordStatus = packet.EGTS_PC_IN_PROGRESS
			case "wrong packet", "wrong then correct":
				confirmation.ResponsePacketID++
			case "wrong record":
				response.ConfirmedRecordNumber++
			case "wrong service":
				(*services)[0].SourceServiceType = packet.SERVICE_AUTH
			case "transport only", "separate", "reversed":
				confirmation.SDR = nil
				answer.FrameDataLength = 3
			}
			frames = append(frames, answer)
			if tc.mode == "separate" || tc.mode == "reversed" {
				appdata := answer
				appdata.PacketType = packet.EGTS_PT_APPDATA
				appdata.PacketID = 21
				appdata.ServicesFrameData = services
				appdata.FrameDataLength = services.Len()
				if tc.mode == "separate" {
					frames = append(frames, appdata)
				} else {
					frames = []packet.Packet{appdata, answer}
				}
			}
			if tc.mode == "wrong then correct" {
				frames = append(frames, record.Packet.PrepareAnswer(10, 22))
			}
			if tc.mode == "closed" {
				err = peer.Close()
				assert.NoError(t, err)
				frames = nil
			}
			for _, frame := range frames {
				encoded, err := frame.Encode()
				assert.NoError(t, err)
				if tc.mode == "checksum" {
					encoded[len(encoded)-1] ^= 1
				}
				_, err = peer.Write(encoded[:5])
				assert.NoError(t, err)
				_, err = peer.Write(encoded[5:])
				assert.NoError(t, err)
				if frame.PacketType == packet.EGTS_PT_APPDATA {
					ack, err := packet.ReadFrame(peer)
					assert.NoError(t, err)
					parsed, err := packet.ReadPacket(ack)
					assert.NoError(t, err)
					assert.Equal(t, frame.PacketID, parsed.ServicesFrameData.(*packet.PTResponse).ResponsePacketID)
				}
			}
			select {
			case err = <-done:
				if tc.fail {
					assert.Error(t, err)
					assert.Nil(t, writer.conn)
				} else {
					assert.NoError(t, err)
				}
			case <-time.After(time.Second):
				t.Error("EGTS delivery did not finish")
			}
			assert.Equal(t, raw, record.Raw)
		})
	}
}

func TestEGTSCancellation(t *testing.T) {
	raw, err := hex.DecodeString("0100000b002300000001991800000001ef0000000202101500d2312b104fba3a9ed227bc35030000b200000000006a8d")
	assert.NoError(t, err)
	record, err := NewRecord(time.Now(), Source{}, raw)
	assert.NoError(t, err)
	client, peer := net.Pipe()
	cfg := configuration.DefaultConfiguration()
	cfg.DestinationsCfg.EGTS.Enabled = true
	writer, err := PrepareEGTS(cfg)
	assert.NoError(t, err)
	writer.conn = client
	ctx, cancel := context.WithCancel(context.Background())
	done := make(chan error, 1)
	go func() {
		done <- writer.WriteContext(ctx, record, nil)
	}()
	_, err = packet.ReadFrame(peer)
	assert.NoError(t, err)
	cancel()
	select {
	case err = <-done:
		assert.Error(t, err)
	case <-time.After(time.Second):
		t.Error("EGTS confirmation wait ignored cancellation")
	}
	err = writer.Close()
	assert.NoError(t, err)
	err = peer.Close()
	assert.NoError(t, err)
	err = writer.WriteContext(context.Background(), record, nil)
	assert.ErrorIs(t, err, net.ErrClosed)
}

func TestEGTSReconnectAuthenticatesAgain(t *testing.T) {
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	assert.NoError(t, err)
	if err != nil {
		return
	}
	t.Cleanup(func() {
		err := listener.Close()
		assert.NoError(t, err)
	})
	cfg := configuration.DefaultConfiguration()
	cfg.DestinationsCfg.EGTS.Enabled = true
	cfg.DestinationsCfg.EGTS.Port = listener.Addr().(*net.TCPAddr).Port
	cfg.DestinationsCfg.EGTS.Auth = configuration.EGTSAuthConf{Enabled: true, UserName: "relay", Password: "remote-password"}
	writer, err := PrepareEGTS(cfg)
	assert.NoError(t, err)
	t.Cleanup(func() {
		err := writer.Close()
		assert.NoError(t, err)
	})
	raw, err := hex.DecodeString("0100000b002300000001991800000001ef0000000202101500d2312b104fba3a9ed227bc35030000b200000000006a8d")
	assert.NoError(t, err)
	record, err := NewRecord(time.Now(), Source{}, raw)
	assert.NoError(t, err)
	term := subrecord.SRTermIdentity{TerminalIdentifier: 123, HDIDE: "0", IMEIE: "0", IMSIE: "0", LNGCE: "0", SSRA: "0", NIDE: "0", BSE: "0", MNE: "0"}
	identity, err := term.Encode()
	assert.NoError(t, err)
	for attempt := 0; attempt < 2; attempt++ {
		ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
		done := make(chan error, 1)
		go func() {
			done <- writer.WriteContext(ctx, record, identity)
		}()
		err = listener.(*net.TCPListener).SetDeadline(time.Now().Add(3 * time.Second))
		assert.NoError(t, err)
		peer, err := listener.Accept()
		assert.NoError(t, err)
		if err != nil {
			cancel()
			return
		}
		err = peer.SetDeadline(time.Now().Add(3 * time.Second))
		assert.NoError(t, err)
		for _, kind := range []uint8{packet.TermIdentity, packet.AuthInfo, packet.PosData} {
			frame, err := packet.ReadFrame(peer)
			assert.NoError(t, err)
			if err != nil {
				cancel()
				err = peer.Close()
				assert.NoError(t, err)
				return
			}
			pkg, err := packet.ReadPacket(frame)
			assert.NoError(t, err)
			services := pkg.ServicesFrameData.(*packet.ServicesFrameData)
			assert.Equal(t, kind, (*services)[0].RecordsData[0].SubrecordType)
			switch kind {
			case packet.TermIdentity:
				assert.Equal(t, uint32(123), (*services)[0].RecordsData[0].SubrecordData.(*subrecord.SRTermIdentity).TerminalIdentifier)
			case packet.AuthInfo:
				auth := (*services)[0].RecordsData[0].SubrecordData.(*subrecord.SRAuthInfo)
				assert.Equal(t, "relay", auth.UserName)
				assert.Equal(t, "remote-password", auth.Password)
			case packet.PosData:
				assert.Equal(t, raw, frame)
				if attempt == 0 {
					continue
				}
			}
			answer := pkg.PrepareAnswer(1, 2)
			encoded, err := answer.Encode()
			assert.NoError(t, err)
			_, err = peer.Write(encoded)
			assert.NoError(t, err)
			if kind != packet.PosData {
				result := pkg.PrepareSRResultCode(packet.EGTS_PC_OK, 2, 3)
				if kind == packet.TermIdentity {
					services := result.ServicesFrameData.(*packet.ServicesFrameData)
					(*services)[0].RecordsData[0] = &packet.RecordData{SubrecordType: packet.AuthParams, SubrecordLength: 1, SubrecordData: &subrecord.SRAuthParams{}}
				}
				encoded, err = result.Encode()
				assert.NoError(t, err)
				_, err = peer.Write(encoded)
				assert.NoError(t, err)
				ack, err := packet.ReadFrame(peer)
				assert.NoError(t, err)
				parsed, err := packet.ReadPacket(ack)
				assert.NoError(t, err)
				assert.Equal(t, result.PacketID, parsed.ServicesFrameData.(*packet.PTResponse).ResponsePacketID)
			}
		}
		err = peer.Close()
		assert.NoError(t, err)
		select {
		case err = <-done:
			if attempt == 0 {
				assert.Error(t, err)
				assert.Nil(t, writer.conn)
			} else {
				assert.NoError(t, err)
			}
		case <-time.After(3 * time.Second):
			t.Error("Relay attempt did not finish")
		}
		cancel()
	}
}

func TestEGTSRejectsSelfConnection(t *testing.T) {
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	assert.NoError(t, err)
	if err != nil {
		return
	}
	t.Cleanup(func() {
		err := listener.Close()
		assert.NoError(t, err)
	})
	cfg := configuration.DefaultConfiguration()
	cfg.ServerCfg.Port = listener.Addr().(*net.TCPAddr).Port
	cfg.DestinationsCfg.EGTS.Enabled = true
	cfg.DestinationsCfg.EGTS.Port = cfg.ServerCfg.Port
	writer, err := PrepareEGTS(cfg)
	assert.NoError(t, err)
	raw, err := hex.DecodeString("0100000b002300000001991800000001ef0000000202101500d2312b104fba3a9ed227bc35030000b200000000006a8d")
	assert.NoError(t, err)
	record, err := NewRecord(time.Now(), Source{}, raw)
	assert.NoError(t, err)
	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()
	err = writer.WriteContext(ctx, record, nil)
	assert.ErrorContains(t, err, "points back to this gateway")
	err = writer.Close()
	assert.NoError(t, err)
}
