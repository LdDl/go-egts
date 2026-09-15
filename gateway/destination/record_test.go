package destination

import (
	"bytes"
	"encoding/hex"
	"encoding/json"
	"math"
	"testing"
	"time"

	"github.com/LdDl/go-egts/crc"
	"github.com/LdDl/go-egts/egts/packet"
	"github.com/LdDl/go-egts/egts/subrecord"
	"github.com/LdDl/go-egts/gateway/logger"
	"github.com/stretchr/testify/assert"
)

type packetRecordTestCase struct {
	name string
	hex  string
}

type encodedRecord struct {
	ReceivedAt time.Time       `json:"received_at"`
	Source     Source          `json:"source"`
	Packet     json.RawMessage `json:"packet"`
	Raw        []byte          `json:"raw"`
}

type encodedEvent struct {
	Application string        `json:"application"`
	Scope       string        `json:"scope"`
	Event       string        `json:"event"`
	Record      encodedRecord `json:"record"`
}

func TestPacketRecord(t *testing.T) {
	cases := []packetRecordTestCase{
		{name: "position", hex: "0100000b002300000001991800000001ef0000000202101500d2312b104fba3a9ed227bc35030000b200000000006a8d"},
		{name: "authentication", hex: "0100020b0020000000014f1900000010010101160000000000523836363130343032393639303030380004417f"},
		{name: "response", hex: "0100030b001000000000b3000000060000005802020003000000002ec1"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			raw, err := hex.DecodeString(tc.hex)
			assert.NoError(t, err)
			if err != nil {
				return
			}
			receivedAt := time.Date(2026, 9, 15, 12, 30, 45, 123456789, time.FixedZone("MSK", 3*60*60))
			terminalID := uint32(42)
			source := Source{RemoteAddress: "[::1]:12345\n\"escaped\"", TerminalID: &terminalID}
			record, err := NewRecord(receivedAt, source, raw)
			assert.NoError(t, err)
			if err != nil {
				return
			}
			data, err := record.Encode()
			assert.NoError(t, err)
			assert.True(t, bytes.HasSuffix(data, []byte("\n")))
			assert.Equal(t, 1, bytes.Count(data, []byte("\n")))
			assert.NotContains(t, string(data), `"level":`)
			var event encodedEvent
			err = json.Unmarshal(data, &event)
			assert.NoError(t, err)
			assert.Equal(t, "egts_gateway", event.Application)
			assert.Equal(t, logger.SCOPE_PACKETS, event.Scope)
			assert.Equal(t, logger.EVENT_PACKET_RECEIVED, event.Event)
			assert.Equal(t, receivedAt.UTC(), event.Record.ReceivedAt)
			assert.Equal(t, source, event.Record.Source)
			assert.Equal(t, raw, event.Record.Raw)
			decoded, err := packet.ReadPacket(raw)
			assert.NoError(t, err)
			expectedPacket, err := json.Marshal(decoded)
			assert.NoError(t, err)
			assert.JSONEq(t, string(expectedPacket), string(event.Record.Packet))
			// The receive buffer and connection metadata can be reused after creating the record.
			raw[0] = 255
			terminalID = 99
			again, err := record.Encode()
			assert.NoError(t, err)
			assert.Equal(t, data, again)
		})
	}
}

func TestPacketRecordWithoutIdentityOrBody(t *testing.T) {
	raw := []byte{1, 0, 0, 11, 0, 0, 0, 7, 0, 1, 0}
	raw[10] = byte(crc.Crc(8, raw[:10]))
	record, err := NewRecord(time.Now(), Source{RemoteAddress: "127.0.0.1:12345"}, raw)
	assert.NoError(t, err)
	if err != nil {
		return
	}
	data, err := record.Encode()
	assert.NoError(t, err)
	assert.Contains(t, string(data), `"terminal_id":null`)
	assert.Contains(t, string(data), `"SFRD":null`)
	assert.Equal(t, raw, record.Raw)
}

func TestPacketRecordErrors(t *testing.T) {
	raw, err := hex.DecodeString("0100000b002300000001991800000001ef0000000202101500d2312b104fba3a9ed227bc35030000b200000000006a8d")
	assert.NoError(t, err)
	if err != nil {
		return
	}
	for length := 0; length < len(raw); length++ {
		record, err := NewRecord(time.Now(), Source{}, raw[:length])
		assert.Error(t, err)
		assert.Nil(t, record)
	}
	trailing := append(append([]byte(nil), raw...), 0)
	record, err := NewRecord(time.Now(), Source{}, trailing)
	assert.Error(t, err)
	assert.Nil(t, record)
	corrupted := append([]byte(nil), raw...)
	corrupted[len(corrupted)-1] ^= 1
	record, err = NewRecord(time.Now(), Source{}, corrupted)
	assert.Error(t, err)
	assert.Nil(t, record)
	data, err := record.Encode()
	assert.Error(t, err)
	assert.Nil(t, data)

	record, err = NewRecord(time.Now(), Source{}, raw)
	assert.NoError(t, err)
	if err != nil {
		return
	}
	services, ok := record.Packet.ServicesFrameData.(*packet.ServicesFrameData)
	assert.True(t, ok)
	if !ok {
		return
	}
	position, ok := (*services)[0].RecordsData[0].SubrecordData.(*subrecord.SRPosData)
	assert.True(t, ok)
	if !ok {
		return
	}
	position.Latitude = math.NaN()
	data, err = record.Encode()
	assert.Error(t, err)
	assert.Nil(t, data)
	var jsonError *json.UnsupportedValueError
	assert.ErrorAs(t, err, &jsonError)
}
