package subrecord

import (
	"bytes"
	"encoding/binary"
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
	buffer := new(bytes.Buffer)
	subr.ConfirmedRecordNumber = binary.LittleEndian.Uint16(b[0:2])
	subr.RecordStatus = uint8(b[2])
}

// Encode Parse EGTS_SR_RECORD_RESPONSE to array of bytes
func (subr *SRRecordResponse) Encode() (b []byte) {
	crn := make([]byte, 2)
	binary.LittleEndian.PutUint16(crn, subr.ConfirmedRecordNumber)
	b = append(b, crn...)
	b = append(b, subr.RecordStatus)
	return b
}

// Len Returns length of bytes slice
func (subr *SRRecordResponse) Len() (l uint16) {
	l = uint16(len(subr.Encode()))
	return l
}
