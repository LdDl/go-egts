package subrecord

// SRResultCode - EGTS_SR_RESULT_CODE
/*
	Код результата для подзаписей сервиса EGTS_AUTH_SERVICE
*/
type SRResultCode struct {
	RCD uint8 // RCD — код, определяющий результат выполнения операции авторизации.
}

// BytesToSRResultCode Parse array of bytes to EGTS_SR_RESULT_CODE
func BytesToSRResultCode(b []byte) (subr SRResultCode) {
	subr.RCD = uint8(b[0])
	return subr
}

// Encode Get array of bytes from EGTS_SR_RESULT_CODE
func (subr *SRResultCode) Encode(rcd uint8) (b []byte) {
	b = make([]byte, 1)
	b[0] = byte(rcd)
	return b
}
