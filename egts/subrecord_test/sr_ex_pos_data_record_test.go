package subrecord

import (
	"encoding/binary"
	"encoding/hex"
	"fmt"
	"io"
	"testing"

	"github.com/LdDl/go-egts/egts/packet"
	"github.com/LdDl/go-egts/egts/subrecord"
	"github.com/stretchr/testify/assert"
)

var (
	SRExPosDataRecordCheckIncome = []string{"18110000"}
)

func TestSRExPosDataRecordDecoding(t *testing.T) {
	for i := range SRExPosDataRecordCheckIncome {
		pkgHex := SRExPosDataRecordCheckIncome[i]
		pkgBytes, err := hex.DecodeString(pkgHex)
		if err != nil {
			t.Errorf("Error: %s", err.Error())
		}
		subr := subrecord.SRExPosDataRecord{}
		err = subr.Decode(pkgBytes)
		if err != nil {
			t.Errorf("Error: %s", err.Error())
		}
		hexed, err := subr.Encode()
		if err != nil {
			t.Errorf("Error: %s", err.Error())
		}
		if hex.EncodeToString(hexed) != SRExPosDataRecordCheckIncome[i] {
			t.Errorf("Have to be %s, but got %s", SRExPosDataRecordCheckIncome[i], hex.EncodeToString(hexed))
		}
	}
}

type exPosDataReadTestCase struct {
	name     string
	data     []byte
	flagBits string
	expected subrecord.SRExPosDataRecord
}

func TestSRExPosDataFields(t *testing.T) {
	cases := []exPosDataReadTestCase{
		{name: "no fields", data: []byte{0}, flagBits: "00000"},
		{
			name:     "VDOP",
			data:     []byte{1, 0x34, 0x12},
			flagBits: "00001",
			expected: subrecord.SRExPosDataRecord{VerticalDiluptionOfPrecision: 0x1234},
		},
		{
			name:     "HDOP",
			data:     []byte{2, 0x45, 0x23},
			flagBits: "00010",
			expected: subrecord.SRExPosDataRecord{HorizontalDiluptionOfPrecision: 0x2345},
		},
		{
			name:     "PDOP",
			data:     []byte{4, 0x56, 0x34},
			flagBits: "00100",
			expected: subrecord.SRExPosDataRecord{PositionDiluptionOfPrecision: 0x3456},
		},
		{
			name:     "satellites",
			data:     []byte{8, 17},
			flagBits: "01000",
			expected: subrecord.SRExPosDataRecord{Satellites: 17},
		},
		{
			name:     "navigation systems",
			data:     []byte{16, 0x81, 0},
			flagBits: "10000",
			expected: subrecord.SRExPosDataRecord{NavigationSystem: 0x81},
		},
		{
			name:     "all fields",
			data:     []byte{0x1f, 0x34, 0x12, 0x45, 0x23, 0x56, 0x34, 17, 0x81, 0},
			flagBits: "11111",
			expected: subrecord.SRExPosDataRecord{
				VerticalDiluptionOfPrecision:   0x1234,
				HorizontalDiluptionOfPrecision: 0x2345,
				PositionDiluptionOfPrecision:   0x3456,
				Satellites:                     17,
				NavigationSystem:               0x81,
			},
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			result := subrecord.SRExPosDataRecord{}
			err := result.Decode(tc.data)
			assert.NoError(t, err)
			if err != nil {
				return
			}
			assert.Equal(t, tc.flagBits, result.NavigationSystemExists+result.SatellitesExists+result.PositionDiluptionOfPrecisionExists+result.HorizontalDiluptionOfPrecisionExists+result.VerticalDiluptionOfPrecisionExists)
			assert.Equal(t, tc.expected.VerticalDiluptionOfPrecision, result.VerticalDiluptionOfPrecision)
			assert.Equal(t, tc.expected.HorizontalDiluptionOfPrecision, result.HorizontalDiluptionOfPrecision)
			assert.Equal(t, tc.expected.PositionDiluptionOfPrecision, result.PositionDiluptionOfPrecision)
			assert.Equal(t, tc.expected.Satellites, result.Satellites)
			assert.Equal(t, tc.expected.NavigationSystem, result.NavigationSystem)
			encoded, err := result.Encode()
			assert.NoError(t, err)
			assert.Equal(t, tc.data, encoded)

			for length := 0; length < len(tc.data); length++ {
				t.Run(fmt.Sprintf("truncated/%d", length), func(t *testing.T) {
					truncated := subrecord.SRExPosDataRecord{}
					err := truncated.Decode(tc.data[:length])
					assert.Error(t, err)
					assert.Equal(t, subrecord.SRExPosDataRecord{}, truncated)
				})
			}
			trailing := append(append([]byte(nil), tc.data...), 0)
			before := result
			err = result.Decode(trailing)
			assert.Error(t, err)
			assert.Equal(t, before, result)
		})
	}
}

func TestSRExPosDataRepeatedDecode(t *testing.T) {
	first := []byte{0x1f, 0x34, 0x12, 0x45, 0x23, 0x56, 0x34, 17, 0x81, 0}
	result := subrecord.SRExPosDataRecord{}
	err := result.Decode(first)
	assert.NoError(t, err)
	if err != nil {
		return
	}
	previous := result
	err = result.Decode([]byte{1, 0x56})
	assert.ErrorIs(t, err, io.ErrUnexpectedEOF)
	assert.Equal(t, previous, result)
	err = result.Decode(nil)
	assert.ErrorIs(t, err, io.EOF)
	assert.Equal(t, previous, result)

	err = result.Decode([]byte{0})
	assert.NoError(t, err)
	expected := subrecord.SRExPosDataRecord{
		VerticalDiluptionOfPrecisionExists:   "0",
		HorizontalDiluptionOfPrecisionExists: "0",
		PositionDiluptionOfPrecisionExists:   "0",
		SatellitesExists:                     "0",
		NavigationSystemExists:               "0",
	}
	assert.Equal(t, expected, result)
}

func TestSRExPosDataInvalidRecord(t *testing.T) {
	cases := []exPosDataReadTestCase{
		{name: "short VDOP", data: []byte{1, 0x12}},
		{name: "short NS", data: []byte{16, 0x81}},
		{name: "trailing data", data: []byte{0, 1}},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			data := []byte{packet.ExtPosData, 0, 0}
			binary.LittleEndian.PutUint16(data[1:3], uint16(len(tc.data)))
			data = append(data, tc.data...)
			var result packet.RecordsData
			err := result.Decode(data)
			assert.Error(t, err)
			assert.Nil(t, result)
		})
	}
}
