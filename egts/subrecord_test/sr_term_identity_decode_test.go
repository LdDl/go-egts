package subrecord_test

import (
	"encoding/binary"
	"encoding/json"
	"fmt"
	"io"
	"testing"

	"github.com/LdDl/go-egts/egts/packet"
	"github.com/LdDl/go-egts/egts/subrecord"
	"github.com/stretchr/testify/assert"
)

type termIdentityDecodeTestCase struct {
	name     string
	flags    byte
	flagBits string
	data     []byte
	expected subrecord.SRTermIdentity
}

type termIdentityDecodeErrorTestCase struct {
	name string
	data []byte
}

var termIdentityAllFieldsData = []byte("\x78\x56\x34\x12\xff" +
	"\x34\x12" +
	"123456789012345" +
	"250011234567890\x00" +
	"rus" +
	"\x2a\x7c\x03" +
	"\x00\x04" +
	"79991234567\x00\x00\x00\x00")

func TestTermIdentityDecodeFields(t *testing.T) {
	cases := []termIdentityDecodeTestCase{
		{name: "no optional fields", flags: 0, flagBits: "00000000"},
		{name: "simple service algorithm", flags: 0x10, flagBits: "00010000"},
		{
			name:     "home dispatcher",
			flags:    0x01,
			flagBits: "00000001",
			data:     []byte{0x34, 0x12},
			expected: subrecord.SRTermIdentity{HomeDispatcherIdentifier: 0x1234},
		},
		{
			name:     "IMEI",
			flags:    0x02,
			flagBits: "00000010",
			data:     []byte("123456789012345"),
			expected: subrecord.SRTermIdentity{InternationalMobileEquipmentIdentity: "123456789012345"},
		},
		{
			name:     "IMSI",
			flags:    0x04,
			flagBits: "00000100",
			data:     []byte("250011234567890\x00"),
			expected: subrecord.SRTermIdentity{InternationalMobileSubscriberIdentity: "250011234567890\x00"},
		},
		{
			name:     "language",
			flags:    0x08,
			flagBits: "00001000",
			data:     []byte("rus"),
			expected: subrecord.SRTermIdentity{LanguageCode: "rus"},
		},
		{
			name:     "network",
			flags:    0x20,
			flagBits: "00100000",
			data:     []byte{0x2a, 0x7c, 0x03},
			expected: subrecord.SRTermIdentity{NetworkIdentifier: []byte{0x2a, 0x7c, 0x03}},
		},
		{
			name:     "buffer size",
			flags:    0x40,
			flagBits: "01000000",
			data:     []byte{0x00, 0x04},
			expected: subrecord.SRTermIdentity{BufferSize: 1024},
		},
		{
			name:     "MSISDN",
			flags:    0x80,
			flagBits: "10000000",
			data:     []byte("79991234567\x00\x00\x00\x00"),
			expected: subrecord.SRTermIdentity{MobileStationIntegratedServicesDigitalNetworkNumber: "79991234567\x00\x00\x00\x00"},
		},
		{
			name:     "all fields",
			flags:    0xff,
			flagBits: "11111111",
			data:     termIdentityAllFieldsData[5:],
			expected: subrecord.SRTermIdentity{
				HomeDispatcherIdentifier:              0x1234,
				InternationalMobileEquipmentIdentity:  "123456789012345",
				InternationalMobileSubscriberIdentity: "250011234567890\x00",
				LanguageCode:                          "rus",
				NetworkIdentifier:                     []byte{0x2a, 0x7c, 0x03},
				BufferSize:                            1024,
				MobileStationIntegratedServicesDigitalNetworkNumber: "79991234567\x00\x00\x00\x00",
			},
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			data := []byte{0x78, 0x56, 0x34, 0x12, tc.flags}
			data = append(data, tc.data...)
			result := subrecord.SRTermIdentity{}
			err := result.Decode(data)
			assert.NoError(t, err)
			if err != nil {
				return
			}
			assert.Equal(t, uint32(0x12345678), result.TerminalIdentifier)
			assert.Equal(t, tc.flagBits, result.MNE+result.BSE+result.NIDE+result.SSRA+result.LNGCE+result.IMSIE+result.IMEIE+result.HDIDE)
			assert.Equal(t, tc.expected.HomeDispatcherIdentifier, result.HomeDispatcherIdentifier)
			assert.Equal(t, tc.expected.InternationalMobileEquipmentIdentity, result.InternationalMobileEquipmentIdentity)
			assert.Equal(t, tc.expected.InternationalMobileSubscriberIdentity, result.InternationalMobileSubscriberIdentity)
			assert.Equal(t, tc.expected.LanguageCode, result.LanguageCode)
			assert.Equal(t, tc.expected.NetworkIdentifier, result.NetworkIdentifier)
			assert.Equal(t, tc.expected.BufferSize, result.BufferSize)
			assert.Equal(t, tc.expected.MobileStationIntegratedServicesDigitalNetworkNumber, result.MobileStationIntegratedServicesDigitalNetworkNumber)
			encoded, err := result.Encode()
			assert.NoError(t, err)
			assert.Equal(t, data, encoded)

			for length := 0; length < len(data); length++ {
				t.Run(fmt.Sprintf("truncated/%d", length), func(t *testing.T) {
					truncated := subrecord.SRTermIdentity{}
					err := truncated.Decode(data[:length])
					assert.Error(t, err)
					assert.Equal(t, subrecord.SRTermIdentity{}, truncated)
					if len(tc.data) > 0 && length == len(data)-1 {
						assert.ErrorIs(t, err, io.ErrUnexpectedEOF)
					}
				})
			}
		})
	}
}

func TestTermIdentityDecodeTrailingData(t *testing.T) {
	cases := []termIdentityDecodeErrorTestCase{
		{name: "no optional fields", data: []byte{1, 0, 0, 0, 0}},
		{name: "all fields", data: termIdentityAllFieldsData},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			data := append(append([]byte(nil), tc.data...), 0)
			result := subrecord.SRTermIdentity{}
			err := result.Decode(data)
			assert.Error(t, err)
			assert.Equal(t, subrecord.SRTermIdentity{}, result)
		})
	}
}

func TestTermIdentityDecodeRepeated(t *testing.T) {
	result := subrecord.SRTermIdentity{}
	err := result.Decode(termIdentityAllFieldsData)
	assert.NoError(t, err)
	if err != nil {
		return
	}
	previous := result
	err = result.Decode([]byte{1, 0, 0, 0, 0x20, 0x12, 0x34})
	assert.ErrorIs(t, err, io.ErrUnexpectedEOF)
	assert.Equal(t, previous, result)
	err = result.Decode(nil)
	assert.ErrorIs(t, err, io.EOF)
	assert.Equal(t, previous, result)

	err = result.Decode([]byte{0, 0, 0, 0, 0})
	assert.NoError(t, err)
	expected := subrecord.SRTermIdentity{
		MNE:   "0",
		BSE:   "0",
		NIDE:  "0",
		SSRA:  "0",
		LNGCE: "0",
		IMSIE: "0",
		IMEIE: "0",
		HDIDE: "0",
	}
	assert.Equal(t, expected, result)
	encoded, err := json.Marshal(result)
	assert.NoError(t, err)
	var fields map[string]json.RawMessage
	err = json.Unmarshal(encoded, &fields)
	assert.NoError(t, err)
	assert.Equal(t, json.RawMessage("null"), fields["NID"])
}

func TestTermIdentityDecodeInvalidRecord(t *testing.T) {
	cases := []termIdentityDecodeErrorTestCase{
		{name: "short home dispatcher", data: []byte{1, 0, 0, 0, 0x01, 0x34}},
		{name: "short network", data: []byte{1, 0, 0, 0, 0x20, 0x12, 0x34}},
		{name: "trailing data", data: []byte{1, 0, 0, 0, 0, 0}},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			data := []byte{packet.TermIdentity, 0, 0}
			binary.LittleEndian.PutUint16(data[1:3], uint16(len(tc.data)))
			data = append(data, tc.data...)
			var result packet.RecordsData
			err := result.Decode(data)
			assert.Error(t, err)
			assert.Nil(t, result)
		})
	}
}
