package subrecord

import (
	"fmt"
	"io"
	"testing"

	"github.com/LdDl/go-egts/egts/subrecord"
	"github.com/stretchr/testify/assert"
)

func TestSRResultCodeValues(t *testing.T) {
	for code := 0; code <= 255; code++ {
		t.Run(fmt.Sprint(code), func(t *testing.T) {
			result := subrecord.SRResultCode{}
			err := result.Decode([]byte{byte(code)})
			assert.NoError(t, err)
			assert.Equal(t, uint8(code), result.RCD)
			value := subrecord.SRResultCode{RCD: uint8(code)}
			encoded, err := value.Encode()
			assert.NoError(t, err)
			assert.Equal(t, []byte{byte(code)}, encoded)
			assert.Equal(t, uint16(1), result.Len())
		})
	}
}

func TestSRResultCodeInvalidLength(t *testing.T) {
	for _, data := range [][]byte{nil, {}, {151, 0}, {151, 0, 0}} {
		result := subrecord.SRResultCode{}
		err := result.Decode(data)
		assert.Error(t, err)
		assert.Equal(t, subrecord.SRResultCode{}, result)
		if len(data) == 0 {
			assert.ErrorIs(t, err, io.EOF)
		}
	}
}

func TestSRResultCodeRepeatedDecode(t *testing.T) {
	result := subrecord.SRResultCode{}
	err := result.Decode([]byte{151})
	assert.NoError(t, err)
	previous := result
	err = result.Decode(nil)
	assert.ErrorIs(t, err, io.EOF)
	assert.Equal(t, previous, result)
	err = result.Decode([]byte{1, 0})
	assert.Error(t, err)
	assert.Equal(t, previous, result)
	err = result.Decode([]byte{0})
	assert.NoError(t, err)
	assert.Equal(t, subrecord.SRResultCode{}, result)
}
