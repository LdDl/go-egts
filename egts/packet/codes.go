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
	AccelerationData  = uint8(21) // EGTS_SR_ACCEL_DATA
	LiquidLevelSensor = uint8(27) // EGTS_SR_LIQUID_LEVEL_SENSOR
)

// Типы кодов результатов
var (
	EGTS_PC_OK              = uint8(0)   // Успешно
	EGTS_PC_IN_PROGRESS     = uint8(1)   // В процессе обработки (результат обработки ещё не известен)
	EGTS_PC_UNS_PROTOCOL    = uint8(128) // Неподдерживаемый протокол
	EGTS_PC_DECRYPT_ERROR   = uint8(129) // Ошибка декодирования
	EGTS_PC_PROC_DENIED     = uint8(130) // Обработка запрещена
	EGTS_PC_INC_HEADERFORM  = uint8(131) // Неверный формат заголовка
	EGTS_PC_INC_DATAFORM    = uint8(132) // Неверный формат данных
	EGTS_PC_UNS_TYPE        = uint8(133) // Неподдерживаемый тип
	EGTS_PC_NOTEN_PARAMS    = uint8(134) // Неверное количество параметров
	EGTS_PC_DBL_PROC        = uint8(135) // Попытка повторной обработки
	EGTS_PC_PROC_SRC_DENIED = uint8(136) // Обработка данных от источника запрещена
	EGTS_PC_HEADERCRC_ERROR = uint8(137) // Ошибка контрольной суммы заголовка
	EGTS_PC_DATACRC_ERROR   = uint8(138) // Ошибка контрольной суммы данных
	EGTS_PC_INVDATALEN      = uint8(139) // Некорректная длина данных
	EGTS_PC_ROUTE_NFOUND    = uint8(140) // Маршрут не найден
	EGTS_PC_ROUTE_CLOSED    = uint8(141) // Маршрут закрыт
	EGTS_PC_ROUTE_DENIED    = uint8(142) // Маршрутизация запрещена
	EGTS_PC_INVADDR         = uint8(143) // Неверный адрес
	EGTS_PC_TTLEXPIRED      = uint8(144) // Превышено количество ретрансляции данных
	EGTS_PC_NO_ACK          = uint8(145) // Нет подтверждения
	EGTS_PC_OBJ_NFOUND      = uint8(146) // Объект не найден
	EGTS_PC_EVNT_NFOUND     = uint8(147) // Событие не найдено
	EGTS_PC_SRVC_NFOUND     = uint8(148) // Сервис не найден
	EGTS_PC_SRVC_DENIED     = uint8(149) // Сервис запрещён
	EGTS_PC_SRVC_UNKN       = uint8(150) // Неизвестный тип сервиса
	EGTS_PC_AUTH_DENIED     = uint8(151) // Авторизация запрещена
	EGTS_PC_ALREADY_EXISTS  = uint8(152) // Объект уже существует
	EGTS_PC_ID_NFOUND       = uint8(153) // Идентификатор не найден
	EGTS_PC_INC_DATETIME    = uint8(154) // Неправильная дата и время
	EGTS_PC_IO_ERROR        = uint8(155) // Ошибка ввода/вывода
	EGTS_PC_NO_RES_AVAIL    = uint8(156) // Недостаточно ресурсов
	EGTS_PC_MODULE_FAULT    = uint8(157) // Внутренний сбой модуля
	EGTS_PC_MODULE_PWR_FLT  = uint8(158) // Сбой в работе цепи питания модуля
	EGTS_PC_MODULE_PROC_FLT = uint8(159) // Сбой в работе микроконтроллера модуля
	EGTS_PC_MODULE_SW_FLT   = uint8(160) // Сбой в работе программы модуля
	EGTS_PC_MODULE_FW_FLT   = uint8(161) // Сбой в работе внутреннего ПО модуля
	EGTS_PC_MODULE_IO_FLT   = uint8(162) // Сбой в работе блока ввода/вывода модуля
	EGTS_PC_MODULE_MEM_FLT  = uint8(163) // Сбой в работе внутренней памяти модуля
	EGTS_PC_TEST_FAILED     = uint8(164) // Тест не пройден

)

// Типы сервисов
var (
	SERVICE_AUTH = uint8(1) // Сервис AUTH_SERVICE
	SERVICE_DATA = uint8(2) // Сервис TELEDATA_SERVICE
)
