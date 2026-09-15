package packet_test

import (
	"encoding/binary"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"testing"

	"github.com/LdDl/go-egts/crc"
	"github.com/LdDl/go-egts/egts/packet"
	"github.com/LdDl/go-egts/egts/subrecord"
	"github.com/stretchr/testify/assert"
)

type readPacketHeaderTestCase struct {
	name      string
	offset    int
	value     byte
	errorCode uint8
}

type readPacketTruncatedTestCase struct {
	name         string
	data         []byte
	headerLength int
}

type readPacketSizeTestCase struct {
	name          string
	headerLength  int
	responseCount int
}

func TestReadPacketInvalidHeader(t *testing.T) {
	cases := []readPacketHeaderTestCase{
		{name: "zero header length", offset: 3, value: 0, errorCode: packet.EGTS_PC_INC_HEADERFORM},
		{name: "short header length", offset: 3, value: 10, errorCode: packet.EGTS_PC_INC_HEADERFORM},
		{name: "long header length", offset: 3, value: 12, errorCode: packet.EGTS_PC_INC_HEADERFORM},
		{name: "route fields without flag", offset: 3, value: 16, errorCode: packet.EGTS_PC_INC_HEADERFORM},
		{name: "maximum header length", offset: 3, value: 255, errorCode: packet.EGTS_PC_INC_HEADERFORM},
		{name: "route flag without fields", offset: 2, value: 0x20, errorCode: packet.EGTS_PC_INC_HEADERFORM},
		{name: "zero protocol version", offset: 0, value: 0, errorCode: packet.EGTS_PC_UNS_PROTOCOL},
		{name: "unknown protocol version", offset: 0, value: 2, errorCode: packet.EGTS_PC_UNS_PROTOCOL},
		{name: "reserved prefix", offset: 2, value: 0x40, errorCode: packet.EGTS_PC_INC_HEADERFORM},
		{name: "header encoding", offset: 4, value: 1, errorCode: packet.EGTS_PC_INC_HEADERFORM},
		{name: "compression", offset: 2, value: 0x04, errorCode: packet.EGTS_PC_UNS_TYPE},
		{name: "encryption", offset: 2, value: 0x08, errorCode: packet.EGTS_PC_DECRYPT_ERROR},
		{name: "signed packet", offset: 9, value: 2, errorCode: packet.EGTS_PC_UNS_TYPE},
		{name: "unknown packet type", offset: 9, value: 3, errorCode: packet.EGTS_PC_UNS_TYPE},
		{name: "maximum packet type", offset: 9, value: 255, errorCode: packet.EGTS_PC_UNS_TYPE},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			data := []byte{1, 0, 0, 11, 0, 3, 0, 7, 0, 0, 0, 6, 0, 0, 0, 0}
			data[tc.offset] = tc.value
			data[10] = byte(crc.Crc(8, data[:10]))
			binary.LittleEndian.PutUint16(data[14:], uint16(crc.Crc(16, data[11:14])))

			var result packet.Packet
			var err error
			assert.NotPanics(t, func() {
				result, err = packet.ReadPacket(data)
			})
			assert.Error(t, err)
			assert.Equal(t, tc.errorCode, result.ErrorCode)
			assert.Nil(t, result.ServicesFrameData)
		})
	}
}

func TestReadPacketTruncated(t *testing.T) {
	appdata, err := hex.DecodeString(AllDataCheckIncome[0])
	assert.NoError(t, err)
	if err != nil {
		return
	}

	response := []byte{1, 0, 0, 11, 0, 3, 0, 7, 0, 0, 0, 6, 0, 0, 0, 0}
	response[10] = byte(crc.Crc(8, response[:10]))
	binary.LittleEndian.PutUint16(response[14:], uint16(crc.Crc(16, response[11:14])))

	routed := []byte{1, 0, 0x20, 16, 0, 3, 0, 7, 0, 0, 1, 0, 2, 0, 3, 0, 6, 0, 0, 0, 0}
	routed[15] = byte(crc.Crc(8, routed[:15]))
	binary.LittleEndian.PutUint16(routed[19:], uint16(crc.Crc(16, routed[16:19])))

	cases := []readPacketTruncatedTestCase{
		{name: "appdata", data: appdata, headerLength: 11},
		{name: "response", data: response, headerLength: 11},
		{name: "routed response", data: routed, headerLength: 16},
	}

	for _, tc := range cases {
		for length := 0; length < len(tc.data); length++ {
			t.Run(fmt.Sprintf("%s/%d", tc.name, length), func(t *testing.T) {
				result, err := packet.ReadPacket(tc.data[:length])
				assert.Error(t, err)
				expectedCode := packet.EGTS_PC_INVDATALEN
				if length < tc.headerLength {
					expectedCode = packet.EGTS_PC_INC_HEADERFORM
				}
				assert.Equal(t, expectedCode, result.ErrorCode)
				assert.Nil(t, result.ServicesFrameData)
			})
		}
	}
}

func TestReadPacketChecksums(t *testing.T) {
	data, err := hex.DecodeString(AllDataCheckIncome[0])
	assert.NoError(t, err)
	if err != nil {
		return
	}

	t.Run("header", func(t *testing.T) {
		corrupted := append([]byte(nil), data...)
		corrupted[10] ^= 1
		result, err := packet.ReadPacket(corrupted)
		assert.Error(t, err)
		assert.Equal(t, packet.EGTS_PC_HEADERCRC_ERROR, result.ErrorCode)
		assert.Nil(t, result.ServicesFrameData)
	})

	t.Run("body before decoding", func(t *testing.T) {
		corrupted := append([]byte(nil), data...)
		corrupted[11] = 0
		corrupted[12] = 0
		result, err := packet.ReadPacket(corrupted)
		assert.Error(t, err)
		assert.Equal(t, packet.EGTS_PC_DATACRC_ERROR, result.ErrorCode)
		assert.Nil(t, result.ServicesFrameData)
	})

	t.Run("body checksum", func(t *testing.T) {
		corrupted := append([]byte(nil), data...)
		corrupted[len(corrupted)-1] ^= 1
		result, err := packet.ReadPacket(corrupted)
		assert.Error(t, err)
		assert.Equal(t, packet.EGTS_PC_DATACRC_ERROR, result.ErrorCode)
		assert.Nil(t, result.ServicesFrameData)
	})

	t.Run("valid checksum with invalid body", func(t *testing.T) {
		corrupted := append([]byte(nil), data...)
		corrupted[11] = 0
		corrupted[12] = 0
		binary.LittleEndian.PutUint16(corrupted[len(corrupted)-2:], uint16(crc.Crc(16, corrupted[11:len(corrupted)-2])))
		result, err := packet.ReadPacket(corrupted)
		assert.Error(t, err)
		assert.Equal(t, packet.EGTS_PC_INC_DATAFORM, result.ErrorCode)
	})
}

func TestReadPacketFrameLength(t *testing.T) {
	for _, length := range []uint16{0, 1, 2, 4, 256, 65535} {
		t.Run(fmt.Sprintf("declared/%d", length), func(t *testing.T) {
			data := []byte{1, 0, 0, 11, 0, 3, 0, 7, 0, 0, 0, 6, 0, 0, 0, 0}
			binary.LittleEndian.PutUint16(data[5:7], length)
			data[10] = byte(crc.Crc(8, data[:10]))
			binary.LittleEndian.PutUint16(data[14:], uint16(crc.Crc(16, data[11:14])))
			result, err := packet.ReadPacket(data)
			assert.Error(t, err)
			assert.Equal(t, packet.EGTS_PC_INVDATALEN, result.ErrorCode)
		})
	}

	t.Run("trailing bytes", func(t *testing.T) {
		data, err := hex.DecodeString(AllDataCheckIncome[0])
		assert.NoError(t, err)
		if err != nil {
			return
		}
		data = append(data, 0)
		result, err := packet.ReadPacket(data)
		assert.Error(t, err)
		assert.Equal(t, packet.EGTS_PC_INVDATALEN, result.ErrorCode)
	})

	t.Run("uint16 overflow", func(t *testing.T) {
		data := make([]byte, 11+65535+2)
		data[0] = 1
		data[3] = 11
		data[9] = packet.EGTS_PT_APPDATA
		binary.LittleEndian.PutUint16(data[5:7], 65535)
		data[10] = byte(crc.Crc(8, data[:10]))
		binary.LittleEndian.PutUint16(data[len(data)-2:], uint16(crc.Crc(16, data[11:len(data)-2])))
		var result packet.Packet
		var err error
		assert.NotPanics(t, func() {
			result, err = packet.ReadPacket(data)
		})
		assert.Error(t, err)
		assert.Equal(t, packet.EGTS_PC_INVDATALEN, result.ErrorCode)
	})
}

func TestReadPacketEmptyFrame(t *testing.T) {
	for _, packetType := range []uint8{packet.EGTS_PT_APPDATA, packet.EGTS_PT_RESPONSE} {
		t.Run(fmt.Sprintf("type/%d", packetType), func(t *testing.T) {
			data := []byte{1, 0, 0, 11, 0, 0, 0, 7, 0, packetType, 0}
			data[10] = byte(crc.Crc(8, data[:10]))
			result, err := packet.ReadPacket(data)
			if packetType == packet.EGTS_PT_RESPONSE {
				assert.Error(t, err)
				assert.Equal(t, packet.EGTS_PC_INC_DATAFORM, result.ErrorCode)
				return
			}
			assert.NoError(t, err)
			assert.Equal(t, packet.EGTS_PC_OK, result.ErrorCode)
			assert.IsType(t, &packet.ServicesFrameData{}, result.ServicesFrameData)
			assert.Zero(t, result.ServicesFrameDataCheckSum)
			encoded, err := json.Marshal(result)
			assert.NoError(t, err)
			assert.Contains(t, string(encoded), `"SFRD":null`)
		})
	}
}

func TestReadPacketLargeFrame(t *testing.T) {
	cases := []readPacketSizeTestCase{
		{name: "260 bytes", headerLength: 11, responseCount: 40},
		{name: "512 bytes", headerLength: 11, responseCount: 82},
		{name: "near packet limit", headerLength: 16, responseCount: 10918},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			body := make([]byte, 7)
			body[5] = packet.SERVICE_DATA
			body[6] = packet.SERVICE_DATA
			for i := 0; i < tc.responseCount; i++ {
				body = append(body, packet.RecordResponse, 3, 0, byte(i), byte(i>>8), 0)
			}
			binary.LittleEndian.PutUint16(body[:2], uint16(len(body)-7))
			data := make([]byte, tc.headerLength)
			data[0] = 1
			data[3] = byte(tc.headerLength)
			data[9] = packet.EGTS_PT_APPDATA
			if tc.headerLength == 16 {
				data[2] = 0x20
				binary.LittleEndian.PutUint16(data[10:12], 123)
				binary.LittleEndian.PutUint16(data[12:14], 456)
				data[14] = 7
			}
			binary.LittleEndian.PutUint16(data[5:7], uint16(len(body)))
			data[tc.headerLength-1] = byte(crc.Crc(8, data[:tc.headerLength-1]))
			data = append(data, body...)
			checksum := make([]byte, 2)
			binary.LittleEndian.PutUint16(checksum, uint16(crc.Crc(16, body)))
			data = append(data, checksum...)

			result, err := packet.ReadPacket(data)
			assert.NoError(t, err)
			if err != nil {
				return
			}
			assert.Equal(t, packet.EGTS_PC_OK, result.ErrorCode)
			assert.Equal(t, uint16(len(body)), result.FrameDataLength)
			records, ok := result.ServicesFrameData.(*packet.ServicesFrameData)
			assert.True(t, ok)
			if !ok {
				return
			}
			assert.Len(t, *records, 1)
			assert.Len(t, (*records)[0].RecordsData, tc.responseCount)
			assert.Equal(t, uint16(tc.responseCount-1), (*records)[0].RecordsData[tc.responseCount-1].SubrecordData.(*subrecord.SRRecordResponse).ConfirmedRecordNumber)
			if tc.headerLength == 16 {
				assert.Equal(t, uint16(123), result.PeerAddress)
				assert.Equal(t, uint16(456), result.RecipientAddress)
				assert.Equal(t, uint8(7), result.TimeToLive)
			}
		})
	}
}
