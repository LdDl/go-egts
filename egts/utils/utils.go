package utils

import (
	"log"
	"strconv"
)

// ReverseByte -
func ReverseByte(val byte) byte {
	var rval byte = 0
	for i := uint(0); i < 8; i++ {
		if val&(1<<i) != 0 {
			rval |= 0x80 >> i
		}
	}
	return rval
}

// ReverseUint8 ..
func ReverseUint8(val uint8) uint8 {
	return ReverseByte(val)
}

// ReverseUint16 ..
func ReverseUint16(val uint16) uint16 {
	var rval uint16 = 0
	for i := uint(0); i < 16; i++ {
		if val&(uint16(1)<<i) != 0 {
			rval |= uint16(0x8000) >> i
		}
	}
	return rval
}

//BitField --
func BitField(b uint16, i ...int) interface{} {
	bitField := []int{1, 2, 4, 8, 16, 32, 64, 128, 256, 512, 1024, 2048, 4096, 8192, 16384, 32768}
	var sum int
	for _, l := range i {
		sum += bitField[l]
	}
	if len(i) == 1 {
		ui := uint(i[0])
		bb := b & uint16(sum)
		f, err := strconv.ParseBool(strconv.FormatUint(uint64(bb&(1<<ui)>>ui), 10))
		if err != nil {
			log.Fatalln("BitField: strconv 1 error: ", err)
		}
		return f
	} else {
		return int(b & uint16(sum))
	}
}

// SetBit Sets the bit at position 'pos' in the integer 'n' with value 'val'
func SetBit(n int, pos uint, val int) int {
	n |= (val << pos)
	return n
}
