package subrecord

import (
	"encoding/binary"

	"github.com/LdDl/go-egts/egts/utils"
)

// SRCountersData EGTS_SR_COUNTERS_DATA
/*
	Используется аппаратно-программным комплексом для передачи
	на абонентский терминал данных о значении счетных входов
*/
type SRCountersData struct {
	Counters []int
}

// BytesToSRCountersData Parse array of bytes to EGTS_SR_COUNTERS_DATA
func BytesToSRCountersData(b []byte) interface{} {
	var d SRCountersData
	d.Counters = make([]int, 8)
	// CFE1 ... CFE8 - (Counter Field Exists)
	cfeFlag := uint16(b[0])
	n := 1
	for i := 0; i < 8; i++ {
		if utils.BitField(cfeFlag, i).(bool) {
			b := append([]byte{0}, b[n:n+3]...)
			d.Counters[i] = int(binary.LittleEndian.Uint32(b))
			n += 3
		} else {
			d.Counters[i] = int(-1)
		}
	}
	return d
}
