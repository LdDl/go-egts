package packet_test

import (
	"encoding/binary"
	"testing"

	"github.com/LdDl/go-egts/crc"
	"github.com/LdDl/go-egts/egts/packet"
	"github.com/LdDl/go-egts/egts/subrecord"
	"github.com/stretchr/testify/assert"
)

type prepareResultCodeTestCase struct {
	name         string
	code         uint8
	recordNumber uint16
	packetID     uint16
}

func TestPrepareSRResultCodePacket(t *testing.T) {
	cases := []prepareResultCodeTestCase{
		{name: "success", code: packet.EGTS_PC_OK, recordNumber: 0x1234, packetID: 0xabcd},
		{name: "auth denied", code: packet.EGTS_PC_AUTH_DENIED, recordNumber: 0xabcd, packetID: 0x1234},
		{name: "zero identifiers", code: packet.EGTS_PC_IN_PROGRESS},
		{name: "maximum", code: 255, recordNumber: 65535, packetID: 65535},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			incoming := packet.Packet{PacketType: packet.EGTS_PT_APPDATA, PacketID: 7}
			previous := incoming
			response := incoming.PrepareSRResultCode(tc.code, tc.recordNumber, tc.packetID)
			assert.Equal(t, previous, incoming)
			assert.Equal(t, packet.EGTS_PT_APPDATA, response.PacketType)
			assert.Equal(t, tc.packetID, response.PacketID)
			assert.Equal(t, uint16(11), response.FrameDataLength)
			encoded, err := response.Encode()
			assert.NoError(t, err)
			if !assert.Len(t, encoded, 24) {
				return
			}
			assert.Equal(t, byte(1), encoded[9])
			assert.Equal(t, []byte{tc.code}, encoded[21:22])
			decoded, err := packet.ReadPacket(encoded)
			assert.NoError(t, err)
			if err != nil {
				return
			}
			assert.Equal(t, packet.EGTS_PT_APPDATA, decoded.PacketType)
			assert.Equal(t, tc.packetID, decoded.PacketID)
			assert.Equal(t, packet.EGTS_PC_OK, decoded.ErrorCode)
			records, ok := decoded.ServicesFrameData.(*packet.ServicesFrameData)
			assert.True(t, ok)
			if !ok {
				return
			}
			if !assert.Len(t, *records, 1) {
				return
			}
			record := (*records)[0]
			assert.Equal(t, "1", record.RSOD)
			assert.Equal(t, "0", record.GRP)
			assert.Equal(t, tc.recordNumber, record.RecordNumber)
			assert.Equal(t, packet.SERVICE_AUTH, record.SourceServiceType)
			assert.Equal(t, packet.SERVICE_AUTH, record.RecipientServiceType)
			assert.Equal(t, uint16(4), record.RecordLength)
			if !assert.Len(t, record.RecordsData, 1) {
				return
			}
			assert.Equal(t, packet.ResultCode, record.RecordsData[0].SubrecordType)
			assert.Equal(t, uint16(1), record.RecordsData[0].SubrecordLength)
			assert.Equal(t, &subrecord.SRResultCode{RCD: tc.code}, record.RecordsData[0].SubrecordData)
			reencoded, err := decoded.Encode()
			assert.NoError(t, err)
			assert.Equal(t, encoded, reencoded)
		})
	}
}

func TestReadPacketInvalidResultCodeLength(t *testing.T) {
	for _, value := range [][]byte{nil, {0, 0}} {
		body := []byte{byte(3 + len(value)), 0, 1, 0, 0x20, 1, 1, 9, byte(len(value)), 0}
		body = append(body, value...)
		data := []byte{1, 0, 0, 11, 0, byte(len(body)), 0, 1, 0, 1, 0}
		data[10] = byte(crc.Crc(8, data[:10]))
		data = append(data, body...)
		checksum := make([]byte, 2)
		binary.LittleEndian.PutUint16(checksum, uint16(crc.Crc(16, body)))
		data = append(data, checksum...)
		result, err := packet.ReadPacket(data)
		assert.Error(t, err)
		assert.Equal(t, packet.EGTS_PC_INC_DATAFORM, result.ErrorCode)
	}
}
