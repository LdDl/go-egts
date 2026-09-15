package subrecord_test

import (
	"encoding/json"
	"strings"
	"testing"

	"github.com/LdDl/go-egts/egts/subrecord"
	"github.com/stretchr/testify/assert"
)

func TestAuthInfo(t *testing.T) {
	for _, raw := range []string{"\x00\x00", "user\x00password\x00", "user\x00password\x00\x00", "user\x00password\x00sequence\x00", strings.Repeat("u", 32) + "\x00" + strings.Repeat("p", 32) + "\x00" + strings.Repeat("s", 255) + "\x00"} {
		value := subrecord.SRAuthInfo{}
		err := value.Decode([]byte(raw))
		assert.NoError(t, err)
		data, err := value.Encode()
		assert.NoError(t, err)
		assert.Equal(t, []byte(raw), data)
		assert.Equal(t, uint16(len(raw)), value.Len())
	}
	for _, raw := range []string{"", "user\x00", "user\x00password", "user\x00password\x00sequence", "u\x00p\x00s\x00extra\x00", strings.Repeat("u", 33) + "\x00p\x00", "u\x00" + strings.Repeat("p", 33) + "\x00", "u\x00p\x00" + strings.Repeat("s", 256) + "\x00"} {
		value := subrecord.SRAuthInfo{UserName: "keep", Password: "keep"}
		before := value
		err := value.Decode([]byte(raw))
		assert.Error(t, err)
		assert.Equal(t, before, value)
	}
	sequence := "old"
	value := subrecord.SRAuthInfo{ServerSequence: &sequence}
	err := value.Decode([]byte("u\x00p\x00"))
	assert.NoError(t, err)
	assert.Nil(t, value.ServerSequence)
	data, err := json.Marshal(value)
	assert.NoError(t, err)
	assert.Contains(t, string(data), `"SS":null`)
	value.Password = "contains\x00delimiter"
	data, err = value.Encode()
	assert.Error(t, err)
	assert.Nil(t, data)
	assert.Zero(t, value.Len())
	var missing *subrecord.SRAuthInfo
	data, err = missing.Encode()
	assert.Error(t, err)
	assert.Nil(t, data)
}

func TestAuthParams(t *testing.T) {
	value := subrecord.SRAuthParams{}
	err := value.Decode([]byte{0})
	assert.NoError(t, err)
	data, err := value.Encode()
	assert.NoError(t, err)
	assert.Equal(t, []byte{0}, data)
	assert.Equal(t, uint16(1), value.Len())
	for _, raw := range [][]byte{nil, {0, 0}, {1}, {255}} {
		err = value.Decode(raw)
		assert.Error(t, err)
		assert.Zero(t, value.Flags)
	}
	value.Flags = 1
	data, err = value.Encode()
	assert.Error(t, err)
	assert.Nil(t, data)
	var missing *subrecord.SRAuthParams
	data, err = missing.Encode()
	assert.Error(t, err)
	assert.Nil(t, data)
}
