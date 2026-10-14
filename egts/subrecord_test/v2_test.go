package subrecord_test

import (
	"encoding/hex"
	"strings"
	"testing"

	"github.com/LdDl/go-egts/egts/subrecord"
	"github.com/stretchr/testify/assert"
)

func TestV2IdentityOptionalFields(t *testing.T) {
	// Table 20: eight-byte TID, flags, HDID, IMEI, IMSI, LNGC, NID, BS, MSISDN, SSLPV.
	b := append([]byte{8, 7, 6, 5, 4, 3, 2, 1, 0xff, 2, 1}, []byte("1234567890123451234567890123456rus")...)
	b = append(b, 1, 2, 3, 4, 5)
	b = append(b, []byte("12345678901234502")...)
	var term subrecord.SRTermIdentityV2
	if !assert.NoError(t, term.Decode(b)) {
		return
	}
	assert.Equal(t, uint64(0x0102030405060708), term.TerminalIdentifier)
	assert.Equal(t, "1234567890123456", term.InternationalMobileSubscriberIdentity)
	assert.Equal(t, uint16(0x0504), term.BufferSize)
	encoded, err := term.Encode()
	if !assert.NoError(t, err) {
		return
	}
	assert.Equal(t, b, encoded)
	for n := 0; n < len(b); n++ {
		if n == len(b)-2 {
			continue
		}
		before := term
		assert.Error(t, term.Decode(b[:n]), "length %d", n)
		assert.Equal(t, before, term)
	}
	if !assert.NoError(t, term.Decode(b[:len(b)-2])) {
		return
	}
	assert.Empty(t, term.ServiceSupportLevelProtocolVersion)
	if !assert.NoError(t, term.Decode([]byte{0x88, 0x77, 0x66, 0x55, 0x44, 0x33, 0x22, 0x11, 0, '0', '2'})) {
		return
	}
	assert.Empty(t, term.InternationalMobileSubscriberIdentity)
	invalid := append([]byte(nil), b...)
	invalid[len(invalid)-1] = '3'
	assert.Error(t, term.Decode(invalid))
}

func TestV2PositionOptionalFields(t *testing.T) {
	b, err := hex.DecodeString("2a000000ffffffffffffffff837bc02c563412a5042143057856341285ff57e80300feff")
	if !assert.NoError(t, err) {
		return
	}
	var pos subrecord.SRPosDataV2
	for _, size := range []int{34, 36} {
		if !assert.NoError(t, pos.Decode(b[:size])) {
			return
		}
		encoded, err := pos.Encode()
		if !assert.NoError(t, err) {
			return
		}
		assert.Equal(t, b[:size], encoded)
	}
	for n := 0; n < len(b); n++ {
		if n == 34 {
			continue
		}
		before := pos
		assert.Error(t, pos.Decode(b[:n]), "length %d", n)
		assert.Equal(t, before, pos)
	}
	withoutAltitude := append([]byte(nil), b[:31]...)
	withoutAltitude[12] &= 0x7f
	if !assert.NoError(t, pos.Decode(withoutAltitude)) {
		return
	}
	assert.Zero(t, pos.Altitude)
	assert.Zero(t, pos.SourceDataExists)
	encoded, err := pos.Encode()
	if !assert.NoError(t, err) {
		return
	}
	assert.Equal(t, withoutAltitude, encoded)
	pos.SpeedTenths = 0x4000
	_, err = pos.Encode()
	assert.Error(t, err)
}

func TestV2VehicleDataAddFixedFields(t *testing.T) {
	b := append([]byte{0x1f}, []byte(strings.Repeat("R", 32))...)
	for _, size := range []int{64, 32, 12, 15, 64} {
		b = append(b, []byte(strings.Repeat("A", size))...)
	}
	var data subrecord.SRVehicleDataAdd
	if !assert.NoError(t, data.Decode(b)) {
		return
	}
	if !assert.Len(t, *data.Model, 64) {
		return
	}
	if !assert.Len(t, *data.OwnerRegistrationNumber, 15) {
		return
	}
	encoded, err := data.Encode()
	if !assert.NoError(t, err) {
		return
	}
	assert.Equal(t, b, encoded)
	for n := 0; n < len(b); n++ {
		assert.Error(t, data.Decode(b[:n]))
	}
	assert.Error(t, data.Decode(append(b, 0)))
	b[0] |= 0x80
	assert.Error(t, data.Decode(b))
}
