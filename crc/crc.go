package crc

import (
	"g_service_bus/pkgs/egts/crc16"
	"g_service_bus/pkgs/egts/crc8"
)

//Crc --
func Crc(bit int, data []byte) (crc int) {
	if bit == 8 {
		table := crc8.MakeTable(crc8.Params{
			Poly:   0x31,
			Init:   0xFF,
			RefIn:  false,
			RefOut: false,
			XorOut: 0x00,
			Check:  0xF7,
			Name:   "CRC-8/EGTS",
		})
		crc = int(crc8.Checksum(data, table))
	} else if bit == 16 {
		table := crc16.MakeTable(crc16.Params{
			Poly:   0x1021,
			Init:   0xFFFF,
			RefIn:  false,
			RefOut: false,
			XorOut: 0x0,
			Check:  0x29B1,
			Name:   "CRC-16/EGTS",
		})
		crc = int(crc16.Checksum(data, table))
	}
	return
}
