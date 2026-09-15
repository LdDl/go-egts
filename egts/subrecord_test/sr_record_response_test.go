package subrecord

import (
	"fmt"
	"io"
	"testing"

	"github.com/LdDl/go-egts/egts/subrecord"
	"github.com/stretchr/testify/assert"
)

type recordResponseValueTestCase struct {
	name     string
	data     []byte
	expected subrecord.SRRecordResponse
}

func TestSRRecordResponseValues(t *testing.T) {
	cases := []recordResponseValueTestCase{
		{name: "zero", data: []byte{0, 0, 0}},
		{
			name:     "byte order",
			data:     []byte{0x34, 0x12, 1},
			expected: subrecord.SRRecordResponse{ConfirmedRecordNumber: 0x1234, RecordStatus: 1},
		},
		{
			name:     "error status",
			data:     []byte{0, 1, 151},
			expected: subrecord.SRRecordResponse{ConfirmedRecordNumber: 256, RecordStatus: 151},
		},
		{
			name:     "maximum",
			data:     []byte{0xff, 0xff, 0xff},
			expected: subrecord.SRRecordResponse{ConfirmedRecordNumber: 65535, RecordStatus: 255},
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			result := subrecord.SRRecordResponse{}
			err := result.Decode(tc.data)
			assert.NoError(t, err)
			assert.Equal(t, tc.expected, result)
			encoded, err := tc.expected.Encode()
			assert.NoError(t, err)
			assert.Equal(t, tc.data, encoded)
			assert.Equal(t, uint16(3), result.Len())
		})
	}
}

func TestSRRecordResponseInvalidLength(t *testing.T) {
	data := []byte{0x34, 0x12, 151}
	for length := 0; length < len(data); length++ {
		t.Run(fmt.Sprintf("length/%d", length), func(t *testing.T) {
			result := subrecord.SRRecordResponse{}
			err := result.Decode(data[:length])
			if length == 1 {
				assert.ErrorIs(t, err, io.ErrUnexpectedEOF)
			} else {
				assert.ErrorIs(t, err, io.EOF)
			}
			assert.Equal(t, subrecord.SRRecordResponse{}, result)
		})
	}
	result := subrecord.SRRecordResponse{}
	err := result.Decode(append(data, 0))
	assert.Error(t, err)
	assert.Equal(t, subrecord.SRRecordResponse{}, result)
}

func TestSRRecordResponseRepeatedDecode(t *testing.T) {
	result := subrecord.SRRecordResponse{}
	err := result.Decode([]byte{0xff, 0xff, 151})
	assert.NoError(t, err)
	previous := result
	for _, data := range [][]byte{nil, {1}, {1, 0}, {1, 0, 0, 0}} {
		err = result.Decode(data)
		assert.Error(t, err)
		assert.Equal(t, previous, result)
	}
	err = result.Decode([]byte{0, 0, 0})
	assert.NoError(t, err)
	assert.Equal(t, subrecord.SRRecordResponse{}, result)
}
