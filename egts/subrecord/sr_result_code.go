package subrecord

import (
	"bytes"
	"fmt"
)

// SRResultCode - EGTS_SR_RESULT_CODE
/*
	Код результата для подзаписей сервиса EGTS_AUTH_SERVICE
*/
type SRResultCode struct {
	// RCD определяет результат выполнения операции авторизации.
	RCD uint8 `json:"RCD"`
}

// Decode Parse array of bytes to EGTS_SR_RESULT_CODE
func (subr *SRResultCode) Decode(b []byte) (err error) {
	buffer := bytes.NewReader(b)
	decoded := SRResultCode{}
	decoded.RCD, err = buffer.ReadByte()
	if err != nil {
		return fmt.Errorf("EGTS_SR_RESULT_CODE; Error reading RCD: %w", err)
	}
	if buffer.Len() != 0 {
		return fmt.Errorf("EGTS_SR_RESULT_CODE; Unexpected trailing data")
	}
	*subr = decoded
	return nil
}

// Encode Parse EGTS_SR_RESULT_CODE to array of bytes
func (subr *SRResultCode) Encode() (b []byte, err error) {
	buffer := new(bytes.Buffer)
	err = buffer.WriteByte(subr.RCD)
	if err != nil {
		return nil, fmt.Errorf("EGTS_SR_RESULT_CODE; Error writing RCD: %w", err)
	}
	return buffer.Bytes(), nil
}

// Len Returns length of bytes slice
func (subr *SRResultCode) Len() (l uint16) {
	encoded, _ := subr.Encode()
	l = uint16(len(encoded))
	return l
}
