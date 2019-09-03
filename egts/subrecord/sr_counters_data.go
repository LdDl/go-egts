package subrecord

import (
	"bytes"
	"encoding/binary"

	"github.com/LdDl/go-egts/egts/utils"
)

// SRCountersData EGTS_SR_COUNTERS_DATA
/*
	Используется аппаратно-программным комплексом для передачи
	на абонентский терминал данных о значении счетных входов
*/
type SRCountersData struct {
	Counters []int `json:"CN"`
}

// Decode Parse array of bytes to EGTS_SR_COUNTERS_DATA
func (subr *SRCountersData) Decode(b []byte) (err error) {
	buffer := bytes.NewReader(b)
	subr.Counters = make([]int, 8)
	// CFE1 ... CFE8 - (Counter Field Exists)
	cfeFlag := uint16(b[0])
	n := 1
	for i := 0; i < 8; i++ {
		if utils.BitField(cfeFlag, i).(bool) {
			b := append([]byte{0}, b[n:n+3]...)
			subr.Counters[i] = int(binary.LittleEndian.Uint32(b))
			n += 3
		} else {
			subr.Counters[i] = int(-1)
		}
	}
	return nil
}

// Encode Parse EGTS_SR_COUNTERS_DATA to array of bytes
func (subr *SRCountersData) Encode() (b []byte) {
	return b
}

// Len Returns length of bytes slice
func (subr *SRCountersData) Len() (l uint16) {
	l = uint16(len(subr.Encode()))
	return l
}
