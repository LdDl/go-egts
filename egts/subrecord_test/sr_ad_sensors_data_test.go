package subrecord

import (
	"encoding/hex"
	"fmt"
	"io"
	"testing"

	"github.com/LdDl/go-egts/egts/subrecord"
	"github.com/stretchr/testify/assert"
)

var (
	SRAdSensorsDataCheckIncome = []string{"000007000000000000000000"}
)

func TestSRAdSensorsDataDecoding(t *testing.T) {
	for i := range SRAdSensorsDataCheckIncome {
		pkgHex := SRAdSensorsDataCheckIncome[i]
		pkgBytes, err := hex.DecodeString(pkgHex)
		if err != nil {
			t.Errorf("Error: %s", err.Error())
		}
		subr := subrecord.SRAdSensorsData{}
		err = subr.Decode(pkgBytes)
		if err != nil {
			t.Errorf("Error: %s", err.Error())
		}
		hexed, err := subr.Encode()
		if err != nil {
			t.Errorf("Error: %s", err.Error())
		}
		if hex.EncodeToString(hexed) != SRAdSensorsDataCheckIncome[i] {
			t.Errorf("Have to be %s, but got %s", SRAdSensorsDataCheckIncome[i], hex.EncodeToString(hexed))
		}
	}
}

type adSensorsMaskTestCase struct {
	name    string
	dioMask byte
	ansMask byte
}

type adSensorsFlagsTestCase struct {
	name  string
	flags []string
}

func TestSRAdSensorsDataMasks(t *testing.T) {
	digital := [8]byte{0x01, 0x80, 0, 0xff, 0x12, 0x34, 0x56, 0xa5}
	analog := [8]uint32{1, 0, 0x1234, 0x123456, 0xabcdef, 0xffffff, 0x010203, 0x654321}
	cases := []adSensorsMaskTestCase{
		{name: "all", dioMask: 0xff, ansMask: 0xff},
	}
	for mask := 0; mask < 256; mask++ {
		cases = append(cases,
			adSensorsMaskTestCase{name: fmt.Sprintf("digital/%02x", mask), dioMask: byte(mask)},
			adSensorsMaskTestCase{name: fmt.Sprintf("analog/%02x", mask), ansMask: byte(mask)},
			adSensorsMaskTestCase{name: fmt.Sprintf("mixed/%02x", mask), dioMask: byte(mask), ansMask: ^byte(mask)},
		)
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			data := []byte{tc.dioMask, 0xa5, tc.ansMask}
			expected := subrecord.SRAdSensorsData{
				DigitalOutputs: 0xa5,
				DIOExists:      []string{"0", "0", "0", "0", "0", "0", "0", "0"},
				ANSExists:      []string{"0", "0", "0", "0", "0", "0", "0", "0"},
			}
			for i := range digital {
				if tc.dioMask&(1<<i) != 0 {
					data = append(data, digital[i])
					expected.ADI[i] = digital[i]
					expected.DIOExists[i] = "1"
				}
			}
			for i := range analog {
				if tc.ansMask&(1<<i) != 0 {
					data = append(data, byte(analog[i]), byte(analog[i]>>8), byte(analog[i]>>16))
					expected.ANS[i] = analog[i]
					expected.ANSExists[i] = "1"
				}
			}
			result := subrecord.SRAdSensorsData{}
			err := result.Decode(data)
			assert.NoError(t, err)
			if err != nil {
				return
			}
			assert.Equal(t, expected, result)
			for attempt := 0; attempt < 3; attempt++ {
				assert.Equal(t, uint16(len(data)), result.Len())
				assert.Equal(t, expected, result)
				encoded, err := result.Encode()
				assert.NoError(t, err)
				assert.Equal(t, data, encoded)
				assert.Equal(t, expected, result)
			}
		})
	}
}

func TestSRAdSensorsDataTruncated(t *testing.T) {
	data := []byte{0x81, 0xa5, 0x05, 0x12, 0x34, 0x56, 0x34, 0x12, 0xef, 0xcd, 0xab}
	for length := 0; length < len(data); length++ {
		t.Run(fmt.Sprintf("length/%d", length), func(t *testing.T) {
			result := subrecord.SRAdSensorsData{}
			err := result.Decode(data[:length])
			assert.Error(t, err)
			assert.Equal(t, subrecord.SRAdSensorsData{}, result)
		})
	}
}

func TestSRAdSensorsDataRepeatedDecode(t *testing.T) {
	first := []byte{0x81, 0xa5, 0x05, 0x12, 0x34, 0x56, 0x34, 0x12, 0xef, 0xcd, 0xab}
	result := subrecord.SRAdSensorsData{}
	err := result.Decode(first)
	assert.NoError(t, err)
	if err != nil {
		return
	}
	previous := result
	err = result.Decode([]byte{0, 0, 1, 0x12, 0x34})
	assert.ErrorIs(t, err, io.ErrUnexpectedEOF)
	assert.Equal(t, previous, result)
	err = result.Decode(nil)
	assert.ErrorIs(t, err, io.EOF)
	assert.Equal(t, previous, result)
	err = result.Decode(append(append([]byte(nil), first...), 0))
	assert.Error(t, err)
	assert.Equal(t, previous, result)

	err = result.Decode([]byte{0, 0, 0})
	assert.NoError(t, err)
	expected := subrecord.SRAdSensorsData{
		DIOExists: []string{"0", "0", "0", "0", "0", "0", "0", "0"},
		ANSExists: []string{"0", "0", "0", "0", "0", "0", "0", "0"},
	}
	assert.Equal(t, expected, result)
}

func TestSRAdSensorsDataInvalidFlags(t *testing.T) {
	cases := []adSensorsFlagsTestCase{
		{name: "nil"},
		{name: "empty", flags: []string{}},
		{name: "short", flags: []string{"1"}},
		{name: "long", flags: []string{"1", "0", "0", "0", "0", "0", "0", "0", "0"}},
		{name: "empty flag", flags: []string{"", "0", "0", "0", "0", "0", "0", "0"}},
		{name: "multiple bits", flags: []string{"01", "0", "0", "0", "0", "0", "0", "0"}},
		{name: "invalid bit", flags: []string{"2", "0", "0", "0", "0", "0", "0", "0"}},
	}
	for _, field := range []string{"DIO", "ANS"} {
		t.Run(field, func(t *testing.T) {
			for _, tc := range cases {
				t.Run(tc.name, func(t *testing.T) {
					value := subrecord.SRAdSensorsData{
						DIOExists: []string{"0", "0", "0", "0", "0", "0", "0", "0"},
						ANSExists: []string{"0", "0", "0", "0", "0", "0", "0", "0"},
					}
					if field == "DIO" {
						value.DIOExists = tc.flags
					} else {
						value.ANSExists = tc.flags
					}
					expected := value
					if value.DIOExists != nil {
						expected.DIOExists = append([]string{}, value.DIOExists...)
					}
					if value.ANSExists != nil {
						expected.ANSExists = append([]string{}, value.ANSExists...)
					}
					encoded, err := value.Encode()
					assert.Error(t, err)
					assert.Nil(t, encoded)
					assert.Equal(t, expected, value)
					assert.Equal(t, uint16(0), value.Len())
					assert.Equal(t, expected, value)
				})
			}
		})
	}
}

func TestSRAdSensorsDataOverflow(t *testing.T) {
	for _, index := range []int{0, 7} {
		t.Run(fmt.Sprintf("sensor/%d", index+1), func(t *testing.T) {
			value := subrecord.SRAdSensorsData{
				DIOExists: []string{"0", "0", "0", "0", "0", "0", "0", "0"},
				ANSExists: []string{"0", "0", "0", "0", "0", "0", "0", "0"},
			}
			value.ANSExists[index] = "1"
			value.ANS[index] = 0x1000000
			encoded, err := value.Encode()
			assert.Error(t, err)
			assert.Nil(t, encoded)
			value.ANSExists[index] = "0"
			encoded, err = value.Encode()
			assert.NoError(t, err)
			assert.Equal(t, []byte{0, 0, 0}, encoded)
		})
	}
}

func TestSRAdSensorsDataConcurrentEncode(t *testing.T) {
	data := []byte{0x81, 0xa5, 0x05, 0x12, 0x34, 0x56, 0x34, 0x12, 0xef, 0xcd, 0xab}
	value := subrecord.SRAdSensorsData{}
	err := value.Decode(data)
	assert.NoError(t, err)
	if err != nil {
		return
	}
	for worker := 0; worker < 8; worker++ {
		t.Run(fmt.Sprintf("worker/%d", worker), func(t *testing.T) {
			t.Parallel()
			for attempt := 0; attempt < 10; attempt++ {
				encoded, err := value.Encode()
				assert.NoError(t, err)
				assert.Equal(t, data, encoded)
				assert.Equal(t, uint16(len(data)), value.Len())
			}
		})
	}
}
