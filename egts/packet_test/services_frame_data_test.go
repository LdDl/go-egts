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

var (
	ServicesFrameDataCheckIncome = []string{"9d001100977c8e5702241100009edd050f02021018009edd050f5fb4b49e8d7da2359b00009bc8550f030012010011040018110000120c000000070000000000000000001307000300000000000014050002860014041b0700400000fbff00001b0700000100000000001b0700010100000000001b07000201006c6300001b0700030100000000001b0700040100000000001b0700050100000000001b0700000200000000001b070001020000000000"}
)

func TestServicesFrameDataDecoding(t *testing.T) {
	for i := range ServicesFrameDataCheckIncome {
		pkgHex := ServicesFrameDataCheckIncome[i]
		pkgBytes, err := hex.DecodeString(pkgHex)
		if err != nil {
			t.Errorf("Error: %s", err.Error())
		}
		subr := packet.ServicesFrameData{}
		err = subr.Decode(pkgBytes)
		if err != nil {
			t.Errorf("Error: %s", err.Error())
		}
		hexed, err := subr.Encode()
		if err != nil {
			t.Errorf("Error: %s", err.Error())
		}
		if hex.EncodeToString(hexed) != ServicesFrameDataCheckIncome[i] {
			t.Errorf("Have to be %s, but got %s", ServicesFrameDataCheckIncome[i], hex.EncodeToString(hexed))
		}
	}
}

type serviceFrameFieldsTestCase struct {
	name     string
	flags    byte
	objectID uint32
	eventID  uint32
	time     uint32
}

type recordDecodeErrorTestCase struct {
	name string
	data []byte
}

func TestServicesFrameDataOptionalFields(t *testing.T) {
	cases := []serviceFrameFieldsTestCase{
		{name: "none", flags: 0},
		{name: "object", flags: 1, objectID: 0x12345678},
		{name: "event", flags: 2, eventID: 0x23456789},
		{name: "object and event", flags: 3, objectID: 0x12345678, eventID: 0x23456789},
		{name: "time", flags: 4, time: 0x3456789a},
		{name: "object and time", flags: 5, objectID: 0x12345678, time: 0x3456789a},
		{name: "event and time", flags: 6, eventID: 0x23456789, time: 0x3456789a},
		{name: "all", flags: 7, objectID: 0x12345678, eventID: 0x23456789, time: 0x3456789a},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			data := []byte{6, 0, 0x34, 0x12, tc.flags}
			if tc.flags&1 != 0 {
				data = append(data, 0x78, 0x56, 0x34, 0x12)
			}
			if tc.flags&2 != 0 {
				data = append(data, 0x89, 0x67, 0x45, 0x23)
			}
			if tc.flags&4 != 0 {
				data = append(data, 0x9a, 0x78, 0x56, 0x34)
			}
			data = append(data, packet.SERVICE_DATA, packet.SERVICE_DATA)
			data = append(data, packet.RecordResponse, 3, 0, 0x45, 0x23, packet.EGTS_PC_OK)

			var result packet.ServicesFrameData
			err := result.Decode(data)
			assert.NoError(t, err)
			if err != nil {
				return
			}
			assert.Len(t, result, 1)
			assert.Equal(t, uint16(6), result[0].RecordLength)
			assert.Equal(t, uint16(0x1234), result[0].RecordNumber)
			assert.Equal(t, tc.objectID, result[0].ObjectIdentifier)
			assert.Equal(t, tc.eventID, result[0].EventIdentifier)
			assert.Equal(t, tc.time, result[0].Time)
			assert.Equal(t, packet.SERVICE_DATA, result[0].SourceServiceType)
			assert.Equal(t, packet.SERVICE_DATA, result[0].RecipientServiceType)
			assert.Len(t, result[0].RecordsData, 1)
			response := result[0].RecordsData[0].SubrecordData.(*subrecord.SRRecordResponse)
			assert.Equal(t, uint16(0x2345), response.ConfirmedRecordNumber)

			encoded, err := result.Encode()
			assert.NoError(t, err)
			assert.Equal(t, data, encoded)

			for length := 1; length < len(data); length++ {
				t.Run(fmt.Sprintf("truncated/%d", length), func(t *testing.T) {
					var truncated packet.ServicesFrameData
					err := truncated.Decode(data[:length])
					assert.Error(t, err)
					assert.Nil(t, truncated)
				})
			}
		})
	}
}

func TestServicesFrameDataInvalidRecordLength(t *testing.T) {
	cases := []recordDecodeErrorTestCase{
		{name: "zero length", data: []byte{0, 0, 1, 0, 0, 2, 2}},
		{name: "missing body", data: []byte{6, 0, 1, 0, 0, 2, 2}},
		{name: "short body", data: []byte{6, 0, 1, 0, 0, 2, 2, 0, 3, 0, 1, 0}},
		{name: "maximum length", data: []byte{255, 255, 1, 0, 0, 2, 2, 0, 3, 0, 1, 0, 0}},
		{name: "short subrecord header", data: []byte{2, 0, 1, 0, 0, 2, 2, 0, 3}},
		{name: "subrecord exceeds record", data: []byte{6, 0, 1, 0, 0, 2, 2, 0, 4, 0, 1, 0, 0}},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			var result packet.ServicesFrameData
			err := result.Decode(tc.data)
			assert.Error(t, err)
			assert.Nil(t, result)

			data := make([]byte, 11)
			data[0] = 1
			data[3] = 11
			data[9] = packet.EGTS_PT_APPDATA
			binary.LittleEndian.PutUint16(data[5:7], uint16(len(tc.data)))
			data[10] = byte(crc.Crc(8, data[:10]))
			data = append(data, tc.data...)
			checksum := make([]byte, 2)
			binary.LittleEndian.PutUint16(checksum, uint16(crc.Crc(16, tc.data)))
			data = append(data, checksum...)
			pkg, err := packet.ReadPacket(data)
			assert.Error(t, err)
			assert.Equal(t, packet.EGTS_PC_INC_DATAFORM, pkg.ErrorCode)
		})
	}
}

func TestServicesFrameDataMultipleRecords(t *testing.T) {
	data := []byte{
		6, 0, 1, 0, 0, 1, 1, 0, 3, 0, 11, 0, 0,
		6, 0, 2, 0, 0, 2, 2, 0, 3, 0, 22, 0, 0,
	}
	var result packet.ServicesFrameData
	err := result.Decode(data)
	assert.NoError(t, err)
	if err != nil {
		return
	}
	assert.Len(t, result, 2)
	assert.Equal(t, uint16(1), result[0].RecordNumber)
	assert.Equal(t, uint16(2), result[1].RecordNumber)
	assert.Equal(t, packet.SERVICE_AUTH, result[0].SourceServiceType)
	assert.Equal(t, packet.SERVICE_DATA, result[1].SourceServiceType)
	first := result[0].RecordsData[0].SubrecordData.(*subrecord.SRRecordResponse)
	second := result[1].RecordsData[0].SubrecordData.(*subrecord.SRRecordResponse)
	assert.Equal(t, uint16(11), first.ConfirmedRecordNumber)
	assert.Equal(t, uint16(22), second.ConfirmedRecordNumber)
	assert.NotSame(t, result[0], result[1])
	encoded, err := result.Encode()
	assert.NoError(t, err)
	assert.Equal(t, data, encoded)
}

func TestServicesFrameDataRepeatedDecode(t *testing.T) {
	first := []byte{6, 0, 1, 0, 0, 2, 2, 0, 3, 0, 11, 0, 0}
	second := []byte{6, 0, 2, 0, 0, 2, 2, 0, 3, 0, 22, 0, 0}
	var result packet.ServicesFrameData
	err := result.Decode(first)
	assert.NoError(t, err)
	err = result.Decode(second)
	assert.NoError(t, err)
	assert.Len(t, result, 1)
	assert.Equal(t, uint16(2), result[0].RecordNumber)

	previous := result
	invalid := append(append([]byte(nil), first...), 1)
	err = result.Decode(invalid)
	assert.Error(t, err)
	assert.Equal(t, previous, result)

	var empty packet.ServicesFrameData
	err = empty.Decode(invalid)
	assert.Error(t, err)
	assert.Nil(t, empty)

	for _, data := range [][]byte{nil, {}} {
		err = result.Decode(data)
		assert.NoError(t, err)
		assert.Nil(t, result)
		encoded, err := json.Marshal(result)
		assert.NoError(t, err)
		assert.Equal(t, "null", string(encoded))
	}
}
