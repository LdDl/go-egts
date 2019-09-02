package subrecord

// SRResultCode - EGTS_SR_RESULT_CODE
/*
	Код результата для подзаписей сервиса EGTS_AUTH_SERVICE
*/
type SRResultCode struct {
	RCD uint8 // RCD — код, определяющий результат выполнения операции авторизации.
}

// Decode Parse array of bytes to EGTS_SR_RESULT_CODE
func (subr *SRResultCode) Decode(b []byte) {
	subr.RCD = uint8(b[0])
}

// Encode Parse EGTS_SR_RESULT_CODE to array of bytes
func (subr *SRResultCode) Encode() (b []byte) {
	return b
}
