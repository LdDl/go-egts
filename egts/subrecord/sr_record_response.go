package subrecord

import "encoding/binary"

// SRRecordResponse EGTS_SR_RECORD_RESPONSE
/*
	Применяется для осуществления подтверждения приема и передачи
	результатов обработки записи уровня поддержки услуг
*/
type SRRecordResponse struct {
	ConfirmedRecordNumber uint16 `json:"CRN"`
	RecordStatus          uint8  `json:"RST"`
}

// BytesToSRRecordResponse Parse array of bytes to EGTS_SR_RECORD_RESPONSE
func BytesToSRRecordResponse(b []byte) (subr SRRecordResponse) {
	subr.ConfirmedRecordNumber = binary.LittleEndian.Uint16(b[0:2])
	subr.RecordStatus = uint8(b[2])
	return subr
}
