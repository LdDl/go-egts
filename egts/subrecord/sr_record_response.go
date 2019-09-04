package subrecord

import (
	"bytes"
	"encoding/binary"
	"fmt"
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
	crn := make([]byte, 4)
	if _, err = buffer.Read(crn); err != nil {
		return fmt.Errorf("Error reading CRN")
	}
	subr.ConfirmedRecordNumber = binary.LittleEndian.Uint16(crn)
	if subr.RecordStatus, err = buffer.ReadByte(); err != nil {
		return fmt.Errorf("Error reading record status")
	}

	return nil
}

// Encode Parse EGTS_SR_RECORD_RESPONSE to array of bytes
func (subr *SRRecordResponse) Encode() (b []byte, err error) {
	crn := make([]byte, 2)
	binary.LittleEndian.PutUint16(crn, subr.ConfirmedRecordNumber)
	b = append(b, crn...)
	b = append(b, subr.RecordStatus)
	return b, nil
}

// Len Returns length of bytes slice
func (subr *SRRecordResponse) Len() (l uint16) {
	encoded, _ := subr.Encode()
	l = uint16(len(encoded))
	return l
}
