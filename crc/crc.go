package crc

import (
	"github.com/LdDl/go-egts/crc/crc16"
	"github.com/LdDl/go-egts/crc/crc8"
)

var (
	table8 = crc8.MakeTable(crc8.Params{
		Poly:   0x31,
		Init:   0xFF,
		RefIn:  false,
		RefOut: false,
		XorOut: 0x00,
		Check:  0xF7,
		Name:   "CRC-8/EGTS",
	})
	table16 = crc16.MakeTable(crc16.Params{
		Poly:   0x1021,
		Init:   0xFFFF,
		RefIn:  false,
		RefOut: false,
		XorOut: 0x0,
		Check:  0x29B1,
		Name:   "CRC-16/EGTS",
	})
)

// Crc calculates the EGTS checksum for 8-bit or 16-bit data validation.
func Crc(bit int, data []byte) (crc int) {
	if bit == 8 {
		crc = int(crc8.Checksum(data, table8))
	} else if bit == 16 {
		crc = int(crc16.Checksum(data, table16))
	}
	return
}
