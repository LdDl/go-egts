package packet_test

import (
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"testing"

	"github.com/LdDl/go-egts/egts/packet"
	"github.com/LdDl/go-egts/egts/subrecord"
	"github.com/stretchr/testify/assert"
)

var (
	PTResponseCheckIncome = []string{"00000006000000580101000300000000"}
)

func TestPTResponseDecoding(t *testing.T) {
	for i := range PTResponseCheckIncome {
		pkgHex := PTResponseCheckIncome[i]
		pkgBytes, err := hex.DecodeString(pkgHex)
		if err != nil {
			t.Errorf("Error: %s", err.Error())
		}
		subr := packet.PTResponse{}
		err = subr.Decode(pkgBytes)
		if err != nil {
			log.Println(pkgBytes)
			t.Errorf("Error: %s", err.Error())
		}
		hexed, err := subr.Encode()
		if err != nil {
			t.Errorf("Error: %s", err.Error())
		}
		if hex.EncodeToString(hexed) != PTResponseCheckIncome[i] {
			t.Errorf("Have to be %s, but got %s", PTResponseCheckIncome[i], hex.EncodeToString(hexed))
		}
	}
}

type ptResponseValueTestCase struct {
	name     string
	data     []byte
	expected packet.PTResponse
}

var ptResponseWithRecord = []byte{
	0x34, 0x12, 0,
	6, 0, 0x78, 0x56, 0x58, 1, 1,
	0, 3, 0, 0xbc, 0x9a, 0,
}

func TestPTResponseWithoutRecords(t *testing.T) {
	cases := []ptResponseValueTestCase{
		{name: "zero", data: []byte{0, 0, 0}},
		{
			name:     "byte order",
			data:     []byte{0x34, 0x12, 138},
			expected: packet.PTResponse{ResponsePacketID: 0x1234, ProcessingResult: 138},
		},
		{
			name:     "maximum",
			data:     []byte{0xff, 0xff, 0xff},
			expected: packet.PTResponse{ResponsePacketID: 65535, ProcessingResult: 255},
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			result := packet.PTResponse{}
			err := result.Decode(tc.data)
			assert.NoError(t, err)
			assert.Equal(t, tc.expected, result)
			assert.Nil(t, result.SDR)
			encoded, err := tc.expected.Encode()
			assert.NoError(t, err)
			assert.Equal(t, tc.data, encoded)
			assert.Equal(t, uint16(3), result.Len())
		})
	}
}

func TestPTResponseMultipleRecords(t *testing.T) {
	data := append([]byte(nil), ptResponseWithRecord...)
	data = append(data, 6, 0, 0xff, 0xff, 0x58, 2, 2, 0, 3, 0, 0xef, 0xcd, 151)
	result := packet.PTResponse{}
	err := result.Decode(data)
	assert.NoError(t, err)
	assert.Equal(t, uint16(0x1234), result.ResponsePacketID)
	assert.Equal(t, uint8(0), result.ProcessingResult)
	expected := packet.ServicesFrameData{
		&packet.ServiceDataRecord{
			RecordLength:         6,
			RecordNumber:         0x5678,
			SSOD:                 "0",
			RSOD:                 "1",
			GRP:                  "0",
			RPP:                  "11",
			TMFE:                 "0",
			EVFE:                 "0",
			OBFE:                 "0",
			SourceServiceType:    1,
			RecipientServiceType: 1,
			RecordsData: packet.RecordsData{
				&packet.RecordData{
					SubrecordType:   packet.RecordResponse,
					SubrecordLength: 3,
					SubrecordData:   &subrecord.SRRecordResponse{ConfirmedRecordNumber: 0x9abc, RecordStatus: 0},
				},
			},
		},
		&packet.ServiceDataRecord{
			RecordLength:         6,
			RecordNumber:         0xffff,
			SSOD:                 "0",
			RSOD:                 "1",
			GRP:                  "0",
			RPP:                  "11",
			TMFE:                 "0",
			EVFE:                 "0",
			OBFE:                 "0",
			SourceServiceType:    2,
			RecipientServiceType: 2,
			RecordsData: packet.RecordsData{
				&packet.RecordData{
					SubrecordType:   packet.RecordResponse,
					SubrecordLength: 3,
					SubrecordData:   &subrecord.SRRecordResponse{ConfirmedRecordNumber: 0xcdef, RecordStatus: 151},
				},
			},
		},
	}
	assert.Equal(t, &expected, result.SDR)
	encoded, err := result.Encode()
	assert.NoError(t, err)
	assert.Equal(t, data, encoded)
	assert.Equal(t, uint16(len(data)), result.Len())
}

func TestPTResponseTruncated(t *testing.T) {
	for length := 0; length < len(ptResponseWithRecord); length++ {
		if length == 3 {
			continue
		}
		t.Run(fmt.Sprintf("length/%d", length), func(t *testing.T) {
			result := packet.PTResponse{}
			err := result.Decode(ptResponseWithRecord[:length])
			assert.Error(t, err)
			assert.Equal(t, packet.PTResponse{}, result)
			if length == 1 || length == 4 {
				assert.ErrorIs(t, err, io.ErrUnexpectedEOF)
			}
			if length == 0 || length == 2 {
				assert.ErrorIs(t, err, io.EOF)
			}
		})
	}
}

func TestPTResponseRepeatedDecode(t *testing.T) {
	result := packet.PTResponse{}
	err := result.Decode(ptResponseWithRecord)
	assert.NoError(t, err)
	previous := result
	for _, data := range [][]byte{
		nil,
		{1},
		{1, 0},
		{1, 0, 0, 6},
		append(append([]byte(nil), ptResponseWithRecord...), 1),
	} {
		err = result.Decode(data)
		assert.Error(t, err)
		assert.Equal(t, previous, result)
	}
	err = result.Decode([]byte{0, 0, 138})
	assert.NoError(t, err)
	assert.Equal(t, packet.PTResponse{ProcessingResult: 138}, result)
	assert.Nil(t, result.SDR)
	encoded, err := result.Encode()
	assert.NoError(t, err)
	assert.Equal(t, []byte{0, 0, 138}, encoded)
	assert.Equal(t, uint16(3), result.Len())
	encoded, err = json.Marshal(result)
	assert.NoError(t, err)
	assert.JSONEq(t, `{"RPID":0,"PR":138,"SFRD":null}`, string(encoded))
	err = result.Decode(ptResponseWithRecord)
	assert.NoError(t, err)
	assert.Equal(t, previous, result)
}
