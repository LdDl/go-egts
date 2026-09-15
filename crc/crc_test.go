package crc

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

type crcTestCase struct {
	name string
	bit  int
	data []byte
	want int
}

func TestEGTSChecks(t *testing.T) {
	cases := []crcTestCase{
		{name: "CRC8 digits", bit: 8, data: []byte("123456789"), want: 0xF7},
		{name: "CRC16 digits", bit: 16, data: []byte("123456789"), want: 0x29B1},
		{name: "CRC8 nil", bit: 8, data: nil, want: 0xFF},
		{name: "CRC16 nil", bit: 16, data: nil, want: 0xFFFF},
		{name: "CRC8 empty", bit: 8, data: []byte{}, want: 0xFF},
		{name: "CRC16 empty", bit: 16, data: []byte{}, want: 0xFFFF},
		{name: "unsupported width", bit: 32, data: []byte("123456789"), want: 0},
	}
	for _, tc := range cases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			got := Crc(tc.bit, tc.data)
			assert.Equal(t, tc.want, got)
		})
	}
}
