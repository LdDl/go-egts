package subrecord

// EgtsSrResultCode - EGTS_SR_RESULT_CODE подзаписей сервиса EGTS_AUTH_SERVICE
type EgtsSrResultCode struct {
	RCD uint8 //RCD — код, определяющий результат выполнения операции авторизации.
}

//ParseEgtsSrResultCode - read bytes in struct EgtsSrResultCode service EGTS_SR_RESULT_CODE
//Подзапись EGTS_SR_RESULT_CODE подзаписей сервиса EGTS_AUTH_SERVICE
//Code subrecord 7
//Подзапись применяется телематической платформой для информирования АС о результатах процедуры аутентификации АС
func ParseEgtsSrResultCode(b []byte) interface{} {
	var d EgtsSrResultCode
	d.RCD = uint8(b[0])
	return d
}

// Encode - generation EGTS_SR_RESULT_CODE with code rezult rcd (0 - EGTS_PC_OK)
func (d *EgtsSrResultCode) Encode(rcd uint8) (b []byte) {
	b = make([]byte, 1)
	b[0] = byte(rcd) // code rezult
	return b
}
