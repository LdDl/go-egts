package subrecord

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"log"
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
	crn := make([]byte, 2)
	if _, err = buffer.Read(crn); err != nil {
		return fmt.Errorf("EGTS_SR_RECORD_RESPONSE; Error reading CRN")
	}
	subr.ConfirmedRecordNumber = binary.LittleEndian.Uint16(crn)
	if subr.RecordStatus, err = buffer.ReadByte(); err != nil {
		log.Println(err, b)
		return fmt.Errorf("EGTS_SR_RECORD_RESPONSE; Error reading RST")
	}

	return nil
}

// Encode Parse EGTS_SR_RECORD_RESPONSE to array of bytes
func (subr *SRRecordResponse) Encode() (b []byte, err error) {
	buffer := new(bytes.Buffer)
	if err = binary.Write(buffer, binary.LittleEndian, subr.ConfirmedRecordNumber); err != nil {
		return nil, fmt.Errorf("EGTS_SR_RECORD_RESPONSE; Error writing CRN")
	}
	if err = buffer.WriteByte(subr.RecordStatus); err != nil {
		return nil, fmt.Errorf("EGTS_SR_RECORD_RESPONSE; Error writing RST")
	}
	return buffer.Bytes(), nil
}

// Len Returns length of bytes slice
func (subr *SRRecordResponse) Len() (l uint16) {
	encoded, _ := subr.Encode()
	l = uint16(len(encoded))
	return l
}
