package packet

// Типы SFRD
var (
	EGTS_PT_RESPONSE       = uint8(0) // EGTS_PT_RESPONSE (подтверждение на пакет Транспортного Уровня)
	EGTS_PT_APPDATA        = uint8(1) // EGTS_PT_APPDATA (пакет содержащий данные  ППУ)
	EGTS_PT_SIGNED_APPDATA = uint8(2) // EGTS_PT_SIGNED_APPDATA (пакет содержащий данные  ППУс цифровой подписью)
)

// Типы подзаписей
var (
	RecordResponse    = uint8(0)  // EGTS_SR_RECORD_RESPONSE
	TermIdentity      = uint8(1)  // EGTS_SR_TERM_IDENTITY
	ResultCode        = uint8(9)  // EGTS_SR_RESULT_CODE
	AuthInfo          = uint8(7)  // EGTS_SR_AUTH_INFO
	PosData           = uint8(16) // EGTS_SR_POS_DATA
	ExtPosData        = uint8(17) // EGTS_SR_EXT_POS_DATA
	AdSensorsData     = uint8(18) // EGTS_SR_AD_SENSORS_DATA
	CountersData      = uint8(19) // EGTS_SR_COUNTERS_DATA
	StateData         = uint8(20) // EGTS_SR_STATE_DATA
	LiquidLevelSensor = uint8(27) // EGTS_SR_LIQUID_LEVEL_SENSOR
)

// Results codes
var (
	EGTS_PC_OK              = uint8(0)   // Успешно
	EGTS_PC_UNS_PROTOCOL    = uint8(128) // Неподдерживаемый протокол
	EGTS_PC_INC_HEADERFORM  = uint8(131) // Неверный формат заголовка
	EGTS_PC_INC_DATAFORM    = uint8(132) // Неверный формат данных
	EGTS_PC_HEADERCRC_ERROR = uint8(137) // Ошибка контрольной суммы заголовка
	EGTS_PC_DATACRC_ERROR   = uint8(138) // Ошибка контрольной суммы данных
	EGTS_PC_SRVC_NFOUND     = uint8(148) // Сервис не найден
)
