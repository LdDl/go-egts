package subrecord

import (
	"encoding/binary"
	"encoding/hex"
	"fmt"
	"io"
	"testing"
	"time"

	"github.com/LdDl/go-egts/egts/packet"
	"github.com/LdDl/go-egts/egts/subrecord"
	"github.com/stretchr/testify/assert"
)

var (
	SRAccelerationDataCheckIncome = []string{"39300a1a611eb822"}
)

func TestSRAccelerationDataDecoding(t *testing.T) {
	for i := range SRAccelerationDataCheckIncome {
		pkgHex := SRAccelerationDataCheckIncome[i]
		pkgBytes, err := hex.DecodeString(pkgHex)
		if err != nil {
			t.Errorf("Error: %s", err.Error())
		}
		subr := subrecord.SRAccelerationData{}
		err = subr.Decode(pkgBytes)
		if err != nil {
			t.Errorf("Error: %s", err.Error())
		}

		hexed, err := subr.Encode()
		if err != nil {
			t.Errorf("Error: %s", err.Error())
		}
		if hex.EncodeToString(hexed) != SRAccelerationDataCheckIncome[i] {
			t.Errorf("Have to be %s, but got %s", SRAccelerationDataCheckIncome[i], hex.EncodeToString(hexed))
		}
	}
}

var (
	SRAccelerationHeaderCheckIncome = []string{
		"02d854311539300a1a611eb82239300a1a611eb822",
	}
)

func TestSRAccelerationHeaderDecoding(t *testing.T) {
	for i := range SRAccelerationHeaderCheckIncome {
		pkgHex := SRAccelerationHeaderCheckIncome[i]
		pkgBytes, err := hex.DecodeString(pkgHex)
		if err != nil {
			t.Errorf("Error: %s", err.Error())
		}
		subr := subrecord.SRAccelerationHeader{}
		err = subr.Decode(pkgBytes)
		if err != nil {
			t.Errorf("Error: %s", err.Error())
		}

		hexed, err := subr.Encode()
		if err != nil {
			t.Errorf("Error: %s", err.Error())
		}
		if hex.EncodeToString(hexed) != SRAccelerationHeaderCheckIncome[i] {
			t.Errorf("Have to be %s, but got %s", SRAccelerationHeaderCheckIncome[i], hex.EncodeToString(hexed))
		}
	}
}

type accelerationDataTestCase struct {
	name     string
	data     []byte
	expected subrecord.SRAccelerationData
}

type accelerationReadErrorTestCase struct {
	name string
	data []byte
}

type accelerationEncodeErrorTestCase struct {
	name  string
	value subrecord.SRAccelerationHeader
}

type accelerationTimeTestCase struct {
	name     string
	seconds  uint32
	expected time.Time
}

var accelerationMeasurementsData = []byte{
	2, 1, 0, 0, 0,
	1, 0, 0x34, 0x12, 0xfe, 0xff, 3, 0,
	20, 0, 0, 0x80, 0xff, 0x7f, 0xff, 0xff,
}

func TestSRAccelerationDataValues(t *testing.T) {
	cases := []accelerationDataTestCase{
		{name: "zero", data: make([]byte, 8)},
		{
			name:     "negative values",
			data:     []byte{1, 0, 0xff, 0xff, 0xf6, 0xff, 0, 0x80},
			expected: subrecord.SRAccelerationData{RTM: 1, XAAV: -1, YAAV: -10, ZAAV: -32768},
		},
		{
			name:     "signed limits",
			data:     []byte{0x34, 0x12, 0xff, 0x7f, 0, 0x80, 0xff, 0xff},
			expected: subrecord.SRAccelerationData{RTM: 0x1234, XAAV: 32767, YAAV: -32768, ZAAV: -1},
		},
		{
			name:     "maximum relative time",
			data:     []byte{0xff, 0xff, 1, 0, 0xff, 0xff, 0, 0},
			expected: subrecord.SRAccelerationData{RTM: 65535, XAAV: 1, YAAV: -1},
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			result := subrecord.SRAccelerationData{}
			err := result.Decode(tc.data)
			assert.NoError(t, err)
			assert.Equal(t, tc.expected, result)
			encoded, err := result.Encode()
			assert.NoError(t, err)
			assert.Equal(t, tc.data, encoded)
			assert.Equal(t, uint16(8), result.Len())
		})
	}
}

func TestSRAccelerationDataInvalidLength(t *testing.T) {
	data := []byte{0x34, 0x12, 0xff, 0x7f, 0, 0x80, 0xff, 0xff}
	for length := 0; length < len(data); length++ {
		t.Run(fmt.Sprintf("length/%d", length), func(t *testing.T) {
			result := subrecord.SRAccelerationData{}
			err := result.Decode(data[:length])
			assert.Error(t, err)
			assert.Equal(t, subrecord.SRAccelerationData{}, result)
			if length%2 != 0 {
				assert.ErrorIs(t, err, io.ErrUnexpectedEOF)
			}
		})
	}
	result := subrecord.SRAccelerationData{}
	err := result.Decode(append(append([]byte(nil), data...), 0))
	assert.Error(t, err)
	assert.Equal(t, subrecord.SRAccelerationData{}, result)
}

func TestSRAccelerationDataRepeatedDecode(t *testing.T) {
	data := []byte{0x34, 0x12, 0xff, 0x7f, 0, 0x80, 0xff, 0xff}
	result := subrecord.SRAccelerationData{}
	err := result.Decode(data)
	assert.NoError(t, err)
	previous := result
	err = result.Decode([]byte{1, 0, 2, 0, 3, 0, 4})
	assert.ErrorIs(t, err, io.ErrUnexpectedEOF)
	assert.Equal(t, previous, result)
	err = result.Decode(append(append([]byte(nil), data...), 0))
	assert.Error(t, err)
	assert.Equal(t, previous, result)
	err = result.Decode(make([]byte, 8))
	assert.NoError(t, err)
	assert.Equal(t, subrecord.SRAccelerationData{}, result)
}

func TestSRAccelerationHeaderMeasurements(t *testing.T) {
	result := subrecord.SRAccelerationHeader{}
	err := result.Decode(accelerationMeasurementsData)
	assert.NoError(t, err)
	if err != nil {
		return
	}
	expected := subrecord.SRAccelerationHeader{
		StructuresAmount: 2,
		AbsoluteTimeUint: 1,
		AbsoluteTime:     time.Date(2010, 1, 1, 0, 0, 1, 0, time.UTC),
		AccelerationData: subrecord.SRAccelerationsData{
			&subrecord.SRAccelerationData{RTM: 1, XAAV: 0x1234, YAAV: -2, ZAAV: 3},
			&subrecord.SRAccelerationData{RTM: 20, XAAV: -32768, YAAV: 32767, ZAAV: -1},
		},
	}
	assert.Equal(t, expected, result)
	if !assert.Len(t, result.AccelerationData, 2) {
		return
	}
	assert.NotSame(t, result.AccelerationData[0], result.AccelerationData[1])
	encoded, err := result.Encode()
	assert.NoError(t, err)
	assert.Equal(t, accelerationMeasurementsData, encoded)
	assert.Equal(t, uint16(len(accelerationMeasurementsData)), result.Len())
}

func TestSRAccelerationHeaderAmounts(t *testing.T) {
	for _, amount := range []int{1, 2, 3, 127, 128, 255} {
		t.Run(fmt.Sprintf("amount/%d", amount), func(t *testing.T) {
			data := make([]byte, 5+8*amount)
			data[0] = byte(amount)
			for i := 0; i < amount; i++ {
				offset := 5 + 8*i
				binary.LittleEndian.PutUint16(data[offset:offset+2], uint16(i))
				binary.LittleEndian.PutUint16(data[offset+2:offset+4], uint16(int16(i-128)))
				binary.LittleEndian.PutUint16(data[offset+4:offset+6], uint16(32767-i))
				binary.LittleEndian.PutUint16(data[offset+6:offset+8], uint16(int16(-i)))
			}
			result := subrecord.SRAccelerationHeader{}
			err := result.Decode(data)
			assert.NoError(t, err)
			if err != nil {
				return
			}
			assert.Equal(t, uint8(amount), result.StructuresAmount)
			assert.Len(t, result.AccelerationData, amount)
			for i, sample := range result.AccelerationData {
				assert.Equal(t, uint16(i), sample.RTM)
				assert.Equal(t, int16(i-128), sample.XAAV)
				assert.Equal(t, int16(32767-i), sample.YAAV)
				assert.Equal(t, int16(-i), sample.ZAAV)
			}
			encoded, err := result.Encode()
			assert.NoError(t, err)
			assert.Equal(t, data, encoded)
			assert.Equal(t, uint16(len(data)), result.Len())
		})
	}
}

func TestSRAccelerationHeaderTruncated(t *testing.T) {
	for length := 0; length < len(accelerationMeasurementsData); length++ {
		t.Run(fmt.Sprintf("length/%d", length), func(t *testing.T) {
			result := subrecord.SRAccelerationHeader{}
			err := result.Decode(accelerationMeasurementsData[:length])
			assert.Error(t, err)
			assert.Equal(t, subrecord.SRAccelerationHeader{}, result)
		})
	}
}

func TestSRAccelerationHeaderInvalidAmount(t *testing.T) {
	cases := []accelerationReadErrorTestCase{
		{name: "zero", data: []byte{0, 0, 0, 0, 0}},
		{name: "missing sample", data: []byte{1, 0, 0, 0, 0}},
		{name: "too few samples", data: append([]byte{2, 0, 0, 0, 0}, make([]byte, 8)...)},
		{name: "too many samples", data: append([]byte{1, 0, 0, 0, 0}, make([]byte, 16)...)},
		{name: "trailing byte", data: append([]byte{1, 0, 0, 0, 0}, make([]byte, 9)...)},
		{name: "zero with sample", data: append([]byte{0, 0, 0, 0, 0}, make([]byte, 8)...)},
		{name: "maximum with one sample", data: append([]byte{255, 0, 0, 0, 0}, make([]byte, 8)...)},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			result := subrecord.SRAccelerationHeader{}
			err := result.Decode(tc.data)
			assert.Error(t, err)
			assert.Equal(t, subrecord.SRAccelerationHeader{}, result)
			data := []byte{packet.AccelerationData, 0, 0}
			binary.LittleEndian.PutUint16(data[1:3], uint16(len(tc.data)))
			data = append(data, tc.data...)
			var records packet.RecordsData
			err = records.Decode(data)
			assert.Error(t, err)
			assert.Nil(t, records)
		})
	}
}

func TestSRAccelerationHeaderRepeatedDecode(t *testing.T) {
	result := subrecord.SRAccelerationHeader{}
	err := result.Decode(accelerationMeasurementsData)
	assert.NoError(t, err)
	if err != nil {
		return
	}
	previous := result
	err = result.Decode(accelerationMeasurementsData[:20])
	assert.Error(t, err)
	assert.Equal(t, previous, result)
	err = result.Decode(nil)
	assert.ErrorIs(t, err, io.EOF)
	assert.Equal(t, previous, result)

	err = result.Decode(append([]byte{1, 0, 0, 0, 0}, make([]byte, 8)...))
	assert.NoError(t, err)
	expected := subrecord.SRAccelerationHeader{
		StructuresAmount: 1,
		AbsoluteTime:     time.Date(2010, 1, 1, 0, 0, 0, 0, time.UTC),
		AccelerationData: subrecord.SRAccelerationsData{&subrecord.SRAccelerationData{}},
	}
	assert.Equal(t, expected, result)
}

func TestSRAccelerationHeaderInvalidEncode(t *testing.T) {
	sample := &subrecord.SRAccelerationData{RTM: 1, XAAV: -1}
	tooMany := make(subrecord.SRAccelerationsData, 256)
	for i := range tooMany {
		tooMany[i] = sample
	}
	cases := []accelerationEncodeErrorTestCase{
		{name: "zero"},
		{name: "missing sample", value: subrecord.SRAccelerationHeader{StructuresAmount: 1}},
		{name: "empty samples", value: subrecord.SRAccelerationHeader{StructuresAmount: 1, AccelerationData: subrecord.SRAccelerationsData{}}},
		{name: "too few samples", value: subrecord.SRAccelerationHeader{StructuresAmount: 2, AccelerationData: subrecord.SRAccelerationsData{sample}}},
		{name: "too many samples", value: subrecord.SRAccelerationHeader{StructuresAmount: 1, AccelerationData: subrecord.SRAccelerationsData{sample, sample}}},
		{name: "zero with sample", value: subrecord.SRAccelerationHeader{AccelerationData: subrecord.SRAccelerationsData{sample}}},
		{name: "nil sample", value: subrecord.SRAccelerationHeader{StructuresAmount: 1, AccelerationData: subrecord.SRAccelerationsData{nil}}},
		{name: "nil second sample", value: subrecord.SRAccelerationHeader{StructuresAmount: 2, AccelerationData: subrecord.SRAccelerationsData{sample, nil}}},
		{name: "more than 255", value: subrecord.SRAccelerationHeader{StructuresAmount: 255, AccelerationData: tooMany}},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			value := tc.value
			value.AbsoluteTime = time.Date(2010, 1, 1, 0, 0, 0, 0, time.UTC)
			encoded, err := value.Encode()
			assert.Error(t, err)
			assert.Nil(t, encoded)
			assert.Equal(t, uint16(0), value.Len())
		})
	}
}

func TestSRAccelerationHeaderTime(t *testing.T) {
	cases := []accelerationTimeTestCase{
		{name: "epoch", seconds: 0, expected: time.Date(2010, 1, 1, 0, 0, 0, 0, time.UTC)},
		{name: "signed limit", seconds: 0x7fffffff, expected: time.Date(2078, 1, 19, 3, 14, 7, 0, time.UTC)},
		{name: "above signed limit", seconds: 0x80000000, expected: time.Date(2078, 1, 19, 3, 14, 8, 0, time.UTC)},
		{name: "maximum", seconds: 0xffffffff, expected: time.Date(2146, 2, 7, 6, 28, 15, 0, time.UTC)},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			data := make([]byte, 13)
			data[0] = 1
			binary.LittleEndian.PutUint32(data[1:5], tc.seconds)
			result := subrecord.SRAccelerationHeader{}
			err := result.Decode(data)
			assert.NoError(t, err)
			assert.Equal(t, tc.seconds, result.AbsoluteTimeUint)
			assert.Equal(t, tc.expected, result.AbsoluteTime)
			encoded, err := result.Encode()
			assert.NoError(t, err)
			assert.Equal(t, data, encoded)
		})
	}
}
