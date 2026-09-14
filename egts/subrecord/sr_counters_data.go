package subrecord

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"io"
	"strconv"
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
	decoded := SRCountersData{}

	// CFE1 ... CFE8 - (Counter Field Exists)
	flagByte := byte(0)
	flagByte, err = buffer.ReadByte()
	if err != nil {
		return fmt.Errorf("EGTS_SR_COUNTERS_DATA; Error reading flags: %w", err)
	}
	flagByteAsBits := fmt.Sprintf("%08b", flagByte)
	decoded.CountersExists = make([]string, 8)
	decoded.CountersExists[0] = flagByteAsBits[7:]
	decoded.CountersExists[1] = flagByteAsBits[6:7]
	decoded.CountersExists[2] = flagByteAsBits[5:6]
	decoded.CountersExists[3] = flagByteAsBits[4:5]
	decoded.CountersExists[4] = flagByteAsBits[3:4]
	decoded.CountersExists[5] = flagByteAsBits[2:3]
	decoded.CountersExists[6] = flagByteAsBits[1:2]
	decoded.CountersExists[7] = flagByteAsBits[:1]
	for i := range decoded.CountersExists {
		if decoded.CountersExists[i] == "1" {
			cn := make([]byte, 3)
			_, err = io.ReadFull(buffer, cn)
			if err != nil {
				return fmt.Errorf("EGTS_SR_COUNTERS_DATA; Error reading CN%d: %w", i+1, err)
			}
			cn = append(cn, 0x00)
			decoded.Counters[i] = binary.LittleEndian.Uint32(cn)
		}
	}
	if buffer.Len() != 0 {
		return fmt.Errorf("EGTS_SR_COUNTERS_DATA; Unexpected trailing data")
	}
	*subr = decoded
	return nil
}

// Encode Parse EGTS_SR_COUNTERS_DATA to array of bytes
func (subr *SRCountersData) Encode() (b []byte, err error) {

	buffer := new(bytes.Buffer)

	if len(subr.CountersExists) != len(subr.Counters) {
		return nil, fmt.Errorf("EGTS_SR_COUNTERS_DATA; Expected 8 counter flags")
	}
	flagsBits := ""
	for i := len(subr.CountersExists) - 1; i >= 0; i-- {
		if subr.CountersExists[i] != "0" && subr.CountersExists[i] != "1" {
			return nil, fmt.Errorf("EGTS_SR_COUNTERS_DATA; Invalid counter flag %d", i+1)
		}
		flagsBits += subr.CountersExists[i]
	}
	flags := uint64(0)
	flags, err = strconv.ParseUint(flagsBits, 2, 8)
	if err != nil {
		return nil, fmt.Errorf("EGTS_SR_COUNTERS_DATA; Error writing flags: %w", err)
	}
	err = buffer.WriteByte(uint8(flags))
	if err != nil {
		return nil, fmt.Errorf("EGTS_SR_COUNTERS_DATA; Error writing byte flags: %w", err)
	}

	for i := range subr.CountersExists {
		if subr.CountersExists[i] == "1" {
			if subr.Counters[i] > 0xffffff {
				return nil, fmt.Errorf("EGTS_SR_COUNTERS_DATA; CN%d exceeds 24 bits", i+1)
			}
			ans := make([]byte, 4)
			binary.LittleEndian.PutUint32(ans, subr.Counters[i])
			_, err = buffer.Write(ans[:3])
			if err != nil {
				return nil, fmt.Errorf("EGTS_SR_COUNTERS_DATA; Error writing CN%d: %w", i+1, err)
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
