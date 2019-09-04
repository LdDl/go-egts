package subrecord

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"strconv"
	"strings"
)

// SRCountersData EGTS_SR_COUNTERS_DATA
/*
	Используется аппаратно-программным комплексом для передачи
	на абонентский терминал данных о значении счетных входов
*/
type SRCountersData struct {
	Counters       [8]uint32 `json:"CN"`
	CountersExists []string  `json:"CNE"`
}

// Decode Parse array of bytes to EGTS_SR_COUNTERS_DATA
func (subr *SRCountersData) Decode(b []byte) (err error) {
	buffer := bytes.NewReader(b)

	// CFE1 ... CFE8 - (Counter Field Exists)
	flagByte := byte(0)
	if flagByte, err = buffer.ReadByte(); err != nil {
		return fmt.Errorf("EGTS_SR_COUNTERS_DATA; Error reading flags")
	}
	flagByteAsBits := fmt.Sprintf("%08b", flagByte)
	subr.CountersExists = make([]string, 8) // not [8]string{}, because slice is needed in Encode() method
	subr.CountersExists[0] = flagByteAsBits[7:]
	subr.CountersExists[1] = flagByteAsBits[6:7]
	subr.CountersExists[2] = flagByteAsBits[5:6]
	subr.CountersExists[3] = flagByteAsBits[4:5]
	subr.CountersExists[4] = flagByteAsBits[3:4]
	subr.CountersExists[5] = flagByteAsBits[2:3]
	subr.CountersExists[6] = flagByteAsBits[1:2]
	subr.CountersExists[7] = flagByteAsBits[:1]
	for i := range subr.CountersExists {
		if subr.CountersExists[i] == "1" {
			cn := make([]byte, 3)
			if _, err = buffer.Read(cn); err != nil {
				return fmt.Errorf("EGTS_SR_COUNTERS_DATA; Error reading CN")
			}
			cn = append(cn, 0x00)
			subr.Counters[i] = binary.LittleEndian.Uint32(cn)
		}
	}
	return nil
}

// Encode Parse EGTS_SR_COUNTERS_DATA to array of bytes
func (subr *SRCountersData) Encode() (b []byte, err error) {

	buffer := new(bytes.Buffer)

	flags := uint64(0)
	for i := len(subr.CountersExists)/2 - 1; i >= 0; i-- { // reversed order of CNE
		opp := len(subr.CountersExists) - 1 - i
		subr.CountersExists[i], subr.CountersExists[opp] = subr.CountersExists[opp], subr.CountersExists[i]
	}
	flags, err = strconv.ParseUint(strings.Join(subr.CountersExists, ""), 2, 8)
	if err != nil {
		return nil, fmt.Errorf("EGTS_SR_COUNTERS_DATA; Error writing flags")
	}
	if err = buffer.WriteByte(uint8(flags)); err != nil {
		return nil, fmt.Errorf("EGTS_SR_COUNTERS_DATA; Error writing byte flags")
	}

	for i := range subr.CountersExists {
		if subr.CountersExists[i] == "1" {
			ans := make([]byte, 4)
			binary.LittleEndian.PutUint32(ans, subr.Counters[i])
			if _, err = buffer.Write(ans[:3]); err != nil {
				return nil, fmt.Errorf("EGTS_SR_COUNTERS_DATA; Error writing CN")
			}
		}
	}

	return buffer.Bytes(), nil
}

// Len Returns length of bytes slice
func (subr *SRCountersData) Len() (l uint16) {
	encoded, _ := subr.Encode()
	l = uint16(len(encoded))
	return l
}
