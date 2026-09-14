package subrecord

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"io"
)

// SRRecordResponse EGTS_SR_RECORD_RESPONSE
/*
	Применяется для осуществления подтверждения приема и передачи
	результатов обработки записи уровня поддержки услуг
*/
type SRRecordResponse struct {
	ConfirmedRecordNumber uint16 `json:"CRN"`
	RecordStatus          uint8  `json:"RST"`
}

// Decode Parse array of bytes to EGTS_SR_RECORD_RESPONSE
func (subr *SRRecordResponse) Decode(b []byte) (err error) {
	buffer := bytes.NewReader(b)
	decoded := SRRecordResponse{}
	crn := make([]byte, 2)
	_, err = io.ReadFull(buffer, crn)
	if err != nil {
		return fmt.Errorf("EGTS_SR_RECORD_RESPONSE; Error reading CRN: %w", err)
	}
	decoded.ConfirmedRecordNumber = binary.LittleEndian.Uint16(crn)
	decoded.RecordStatus, err = buffer.ReadByte()
	if err != nil {
		return fmt.Errorf("EGTS_SR_RECORD_RESPONSE; Error reading RST: %w", err)
	}

	if buffer.Len() != 0 {
		return fmt.Errorf("EGTS_SR_RECORD_RESPONSE; Unexpected trailing data")
	}
	*subr = decoded
	return nil
}

// Encode Parse EGTS_SR_RECORD_RESPONSE to array of bytes
func (subr *SRRecordResponse) Encode() (b []byte, err error) {
	if subr == nil {
		return nil, fmt.Errorf("SRRecordResponse; Subrecord is nil")
	}
	buffer := new(bytes.Buffer)
	err = binary.Write(buffer, binary.LittleEndian, subr.ConfirmedRecordNumber)
	if err != nil {
		return nil, fmt.Errorf("EGTS_SR_RECORD_RESPONSE; Error writing CRN: %w", err)
	}
	err = buffer.WriteByte(subr.RecordStatus)
	if err != nil {
		return nil, fmt.Errorf("EGTS_SR_RECORD_RESPONSE; Error writing RST: %w", err)
	}
	return buffer.Bytes(), nil
}

// Len Returns length of bytes slice
func (subr *SRRecordResponse) Len() (l uint16) {
	encoded, _ := subr.Encode()
	l = uint16(len(encoded))
	return l
}
