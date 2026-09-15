package packet_test

import (
	"encoding/binary"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"testing"

	"github.com/LdDl/go-egts/egts/packet"
	"github.com/LdDl/go-egts/egts/subrecord"
	"github.com/stretchr/testify/assert"
)

var (
	RecordsDataCheckIncome = []string{"1018009edd050f5fb4b49e8d7da2359b00009bc8550f030012010011040018110000120c000000070000000000000000001307000300000000000014050002860014041b0700400000fbff00001b0700000100000000001b0700010100000000001b07000201006c6300001b0700030100000000001b0700040100000000001b0700050100000000001b0700000200000000001b07000102000000000015150002d854311539300a1a611eb82239300a1a611eb822"}
)

func TestRecordsDataDecoding(t *testing.T) {
	for i := range RecordsDataCheckIncome {
		pkgHex := RecordsDataCheckIncome[i]
		pkgBytes, err := hex.DecodeString(pkgHex)
		if err != nil {
			t.Errorf("Error: %s", err.Error())
		}
		subr := packet.RecordsData{}
		err = subr.Decode(pkgBytes)
		if err != nil {
			t.Errorf("Error: %s", err.Error())
		}
		hexed, err := subr.Encode()
		if err != nil {
			t.Errorf("Error: %s", err.Error())
		}
		if hex.EncodeToString(hexed) != RecordsDataCheckIncome[i] {
			t.Errorf("Have to be %s, but got %s", RecordsDataCheckIncome[i], hex.EncodeToString(hexed))
		}
	}
}

type subrecordReadTestCase struct {
	name          string
	subrecordType uint8
	data          []byte
}

func TestRecordsDataTruncated(t *testing.T) {
	cases := []subrecordReadTestCase{
		{name: "response", subrecordType: packet.RecordResponse, data: []byte{1, 0, 0}},
		{name: "result code", subrecordType: packet.ResultCode, data: []byte{151}},
		{name: "terminal identity", subrecordType: packet.TermIdentity, data: []byte{1, 0, 0, 0, 1, 2, 0}},
		{name: "extended position", subrecordType: packet.ExtPosData, data: []byte{1, 2, 0}},
		{name: "counter", subrecordType: packet.CountersData, data: []byte{1, 2, 0, 0}},
		{name: "liquid sensor", subrecordType: packet.LiquidLevelSensor, data: []byte{0, 0, 0, 1, 2, 3, 4}},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			data := []byte{tc.subrecordType, 0, 0}
			binary.LittleEndian.PutUint16(data[1:3], uint16(len(tc.data)))
			data = append(data, tc.data...)
			var result packet.RecordsData
			err := result.Decode(data)
			assert.NoError(t, err)
			assert.Len(t, result, 1)
			if err != nil {
				return
			}
			assert.Equal(t, tc.subrecordType, result[0].SubrecordType)
			assert.Equal(t, uint16(len(tc.data)), result[0].SubrecordLength)

			for length := 1; length < len(data); length++ {
				t.Run(fmt.Sprintf("length/%d", length), func(t *testing.T) {
					var truncated packet.RecordsData
					err := truncated.Decode(data[:length])
					assert.Error(t, err)
					assert.Nil(t, truncated)
				})
			}
		})
	}
}

func TestRecordsDataResultCodes(t *testing.T) {
	data := []byte{9, 1, 0, 0, 0, 3, 0, 0x34, 0x12, 0, 9, 1, 0, 151}
	var result packet.RecordsData
	err := result.Decode(data)
	assert.NoError(t, err)
	if !assert.Len(t, result, 3) {
		return
	}
	assert.Equal(t, &subrecord.SRResultCode{RCD: 0}, result[0].SubrecordData)
	assert.Equal(t, &subrecord.SRRecordResponse{ConfirmedRecordNumber: 0x1234}, result[1].SubrecordData)
	assert.Equal(t, &subrecord.SRResultCode{RCD: 151}, result[2].SubrecordData)
	assert.NotSame(t, result[0].SubrecordData, result[2].SubrecordData)
	encoded, err := result.Encode()
	assert.NoError(t, err)
	assert.Equal(t, data, encoded)
	previous := result
	for _, invalid := range [][]byte{
		{9, 0, 0},
		{9, 2, 0, 0, 0},
		{0, 4, 0, 0x34, 0x12, 0, 0},
	} {
		err = result.Decode(invalid)
		assert.Error(t, err)
		assert.Equal(t, previous, result)
	}
}

func TestRecordsDataInvalidSubrecordLength(t *testing.T) {
	cases := []recordDecodeErrorTestCase{
		{name: "zero length", data: []byte{0, 0, 0}},
		{name: "missing body", data: []byte{0, 3, 0}},
		{name: "short body", data: []byte{0, 4, 0, 1, 0, 0}},
		{name: "maximum length", data: []byte{0, 255, 255, 1, 0, 0}},
		{name: "unknown subrecord", data: []byte{255, 1, 0, 0}},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			var result packet.RecordsData
			err := result.Decode(tc.data)
			assert.Error(t, err)
			assert.Nil(t, result)
		})
	}
}

func TestRecordsDataMultipleSubrecords(t *testing.T) {
	data := []byte{0, 3, 0, 11, 0, 0, 0, 3, 0, 22, 0, 1}
	var result packet.RecordsData
	err := result.Decode(data)
	assert.NoError(t, err)
	if err != nil {
		return
	}
	assert.Len(t, result, 2)
	first := result[0].SubrecordData.(*subrecord.SRRecordResponse)
	second := result[1].SubrecordData.(*subrecord.SRRecordResponse)
	assert.Equal(t, uint16(11), first.ConfirmedRecordNumber)
	assert.Equal(t, uint16(22), second.ConfirmedRecordNumber)
	assert.Equal(t, uint8(0), first.RecordStatus)
	assert.Equal(t, uint8(1), second.RecordStatus)
	assert.NotSame(t, result[0], result[1])
	assert.NotSame(t, first, second)
	encoded, err := result.Encode()
	assert.NoError(t, err)
	assert.Equal(t, data, encoded)
}

func TestRecordsDataRepeatedDecode(t *testing.T) {
	first := []byte{0, 3, 0, 11, 0, 0}
	second := []byte{0, 3, 0, 22, 0, 0}
	var result packet.RecordsData
	err := result.Decode(first)
	assert.NoError(t, err)
	err = result.Decode(second)
	assert.NoError(t, err)
	assert.Len(t, result, 1)
	response := result[0].SubrecordData.(*subrecord.SRRecordResponse)
	assert.Equal(t, uint16(22), response.ConfirmedRecordNumber)

	previous := result
	invalid := append(append([]byte(nil), first...), 1)
	err = result.Decode(invalid)
	assert.Error(t, err)
	assert.Equal(t, previous, result)

	var empty packet.RecordsData
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
