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
	SRCountersDataCheckIncome = []string{"03000000000000"}
)

func TestSRCountersDataDecoding(t *testing.T) {
	for i := range SRCountersDataCheckIncome {
		pkgHex := SRCountersDataCheckIncome[i]
		pkgBytes, err := hex.DecodeString(pkgHex)
		if err != nil {
			t.Errorf("Error: %s", err.Error())
		}
		subr := subrecord.SRCountersData{}
		err = subr.Decode(pkgBytes)
		if err != nil {
			t.Errorf("Error: %s", err.Error())
		}
		hexed, err := subr.Encode()
		if err != nil {
			t.Errorf("Error: %s", err.Error())
		}
		if hex.EncodeToString(hexed) != SRCountersDataCheckIncome[i] {
			t.Errorf("Have to be %s, but got %s", SRCountersDataCheckIncome[i], hex.EncodeToString(hexed))
		}
	}
}

type countersFlagsTestCase struct {
	name  string
	flags []string
}

func TestSRCountersDataMasks(t *testing.T) {
	values := [8]uint32{1, 0, 0x1234, 0x123456, 0xabcdef, 0xffffff, 0x010203, 0x654321}
	for mask := 0; mask < 256; mask++ {
		t.Run(fmt.Sprintf("mask/%02x", mask), func(t *testing.T) {
			data := []byte{byte(mask)}
			expected := subrecord.SRCountersData{
				CountersExists: []string{"0", "0", "0", "0", "0", "0", "0", "0"},
			}
			for i := range values {
				if mask&(1<<i) != 0 {
					data = append(data, byte(values[i]), byte(values[i]>>8), byte(values[i]>>16))
					expected.Counters[i] = values[i]
					expected.CountersExists[i] = "1"
				}
			}
			result := subrecord.SRCountersData{}
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

func TestSRCountersDataTruncated(t *testing.T) {
	data := []byte{0x85, 0x56, 0x34, 0x12, 0xef, 0xcd, 0xab, 0xff, 0xff, 0xff}
	for length := 0; length < len(data); length++ {
		t.Run(fmt.Sprintf("length/%d", length), func(t *testing.T) {
			result := subrecord.SRCountersData{}
			err := result.Decode(data[:length])
			assert.Error(t, err)
			assert.Equal(t, subrecord.SRCountersData{}, result)
		})
	}
}

func TestSRCountersDataRepeatedDecode(t *testing.T) {
	first := []byte{0x85, 0x56, 0x34, 0x12, 0xef, 0xcd, 0xab, 0xff, 0xff, 0xff}
	result := subrecord.SRCountersData{}
	err := result.Decode(first)
	assert.NoError(t, err)
	if err != nil {
		return
	}
	previous := result
	err = result.Decode([]byte{1, 0x12, 0x34})
	assert.ErrorIs(t, err, io.ErrUnexpectedEOF)
	assert.Equal(t, previous, result)
	err = result.Decode(nil)
	assert.ErrorIs(t, err, io.EOF)
	assert.Equal(t, previous, result)
	err = result.Decode(append(append([]byte(nil), first...), 0))
	assert.Error(t, err)
	assert.Equal(t, previous, result)

	err = result.Decode([]byte{0})
	assert.NoError(t, err)
	expected := subrecord.SRCountersData{
		CountersExists: []string{"0", "0", "0", "0", "0", "0", "0", "0"},
	}
	assert.Equal(t, expected, result)
}

func TestSRCountersDataInvalidFlags(t *testing.T) {
	cases := []countersFlagsTestCase{
		{name: "nil"},
		{name: "empty", flags: []string{}},
		{name: "short", flags: []string{"1"}},
		{name: "long", flags: []string{"1", "0", "0", "0", "0", "0", "0", "0", "0"}},
		{name: "empty flag", flags: []string{"", "0", "0", "0", "0", "0", "0", "0"}},
		{name: "multiple bits", flags: []string{"01", "0", "0", "0", "0", "0", "0", "0"}},
		{name: "invalid bit", flags: []string{"2", "0", "0", "0", "0", "0", "0", "0"}},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			value := subrecord.SRCountersData{CountersExists: tc.flags}
			expected := subrecord.SRCountersData{CountersExists: tc.flags}
			if tc.flags != nil {
				expected.CountersExists = append([]string{}, tc.flags...)
			}
			encoded, err := value.Encode()
			assert.Error(t, err)
			assert.Nil(t, encoded)
			assert.Equal(t, expected, value)
			assert.Equal(t, uint16(0), value.Len())
			assert.Equal(t, expected, value)
		})
	}
}

func TestSRCountersDataOverflow(t *testing.T) {
	for _, index := range []int{0, 7} {
		t.Run(fmt.Sprintf("counter/%d", index+1), func(t *testing.T) {
			value := subrecord.SRCountersData{
				CountersExists: []string{"0", "0", "0", "0", "0", "0", "0", "0"},
			}
			value.CountersExists[index] = "1"
			value.Counters[index] = 0x1000000
			encoded, err := value.Encode()
			assert.Error(t, err)
			assert.Nil(t, encoded)
			value.CountersExists[index] = "0"
			encoded, err = value.Encode()
			assert.NoError(t, err)
			assert.Equal(t, []byte{0}, encoded)
		})
	}
}

func TestSRCountersDataConcurrentEncode(t *testing.T) {
	data := []byte{5, 0x56, 0x34, 0x12, 0xef, 0xcd, 0xab}
	value := subrecord.SRCountersData{}
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
