package subrecord

import (
	"encoding/binary"

	"github.com/LdDl/go-egts/egts/utils"
)

// SRLiquidLevelSensor EGTS_SR_LIQUID_LEVEL_SENSOR
/*
	Применяется абонентским терминалом
	для передачи на аппаратно-программный комплекс
	данных о показаниях ДУЖ
*/
type SRLiquidLevelSensor struct {
	LiquidLevelSensorNumber    int  // LLSN Liquid Level Sensor Number
	RawbFlag                   bool // RDF bit 3 флаг, определяющий формат поля LLSD данной подзаписи:
	LiquidLevelSensorValueUnit int  // LLSVU	bit 4-5 битовый флаг, определяющий единицы измерения показаний ДУЖ:
	LiquidLevelSensorErrorFlag bool // LLSEF	bit 7	битовый флаг, определяющий наличие ошибок при считывании значения датчика уровня жидкости
	Flags                      uint8
	MADDR                      uint16 // MAC Address  адрес модуля, данные о показаниях ДУЖ с которого поступили в абонентский терминал
	LiquidLevelSensorb         uint32 // LLSD показания ДУЖ в формате, определяемом флагом RDF
}

// Decode Parse array of bytes to EGTS_SR_LIQUID_LEVEL_SENSOR
func (subr *SRLiquidLevelSensor) Decode(b []byte) {
	/*
	   #pragma pack( push, 1 )
	   typedef struct {
	   	uint8_t		FLG;		// битовые флаги
	   	uint16_t	MADDR;	// адрес модуля, данные о показаниях ДУЖ с которого поступили в абонентский терминал
	   	uint32_t	LLSD;		// показания ДУЖ в формате, определяемом флагом RDF
	   } EGTS_SR_LIQUID_LEVEL_SENSOR_RECORD;
	   #pragma pack( pop )

	   /* FLG:
	   Name	Bit Value
	   LLSN	0-2	порядковый номер датчика, 3 бита
	   RDF		3		флаг, определяющий формат поля LLSD данной подзаписи:
	   					0 - поле LLSD имеет размер 4 байта (тип данных UINT) и содержит показания ДУЖ в формате,
	   					определяемом полем LLSVU;
	   					1 - поле LLSD содержит данные ДУЖ в неизменном виде, как они поступили из внешнего
	   					порта абонентского терминала (размер поля LLSD при этом определяется исходя из
	   					общей длины данной подзаписи и размеров расположенных перед LLSD полей).
	   LLSVU	4-5	битовый флаг, определяющий единицы измерения показаний ДУЖ:
	   					00 - нетарированное показание ДУЖ.
	   					01 - показания ДУЖ в процентах от общего объема емкости;
	   					10 - показания ДУЖ в литрах с дискретностью в 0,1 литра.
	   LLSEF	6		битовый флаг, определяющий наличие ошибок при считывании значения датчика уровня жидкости
	   			7		не используется
	*/
	if len(b) != 7 {
		//log.Panicln("INVALID LEN ", len(b))
		return
	}
	flagBytes := uint16(b[0])
	subr.LiquidLevelSensorNumber = utils.BitField(flagBytes, 0, 1, 2).(int)
	subr.RawbFlag = utils.BitField(flagBytes, 3).(bool)
	subr.LiquidLevelSensorValueUnit = utils.BitField(flagBytes, 4, 5).(int)
	subr.LiquidLevelSensorErrorFlag = utils.BitField(flagBytes, 6).(bool)
	subr.Flags = uint8(flagBytes)
	subr.MADDR = binary.LittleEndian.Uint16(b[1:3])
	subr.LiquidLevelSensorb = binary.LittleEndian.Uint32(b[3:])

	/* TODO LLVU Check ErrorFlag

	   	int Parse_EGTS_SR_LIQUID_LEVEL_SENSOR(int rlen, EGTS_SR_LIQUID_LEVEL_SENSOR_RECORD *posb, ST_RECORD *record){
	   	int b_size;

	   	if( !record )
	   		return 0;

	   	if( posb->FLG & B3 ){	// размер поля LLSD определяется исходя из общей длины данной подзаписи и размеров расположенных перед LLSD полей
	      	b_size = rlen;	// здесь нас интересует общая длинна записи
	   		// ибо как хранить такие данные хз
	   	}
	   	else {	// поле LLSD имеет размер 4 байта
	      	b_size = sizeof(EGTS_SR_LIQUID_LEVEL_SENSOR_RECORD);

	   		if( !(posb->FLG & B6) ){	// ошибок не обнаружено
	   	  	if( record->fuel[0] ){	// показания первого датчика уже записаны
	   				record->fuel[1] = posb->LLSD;
	   				if( posb->FLG & 32 )	// показания ДУЖ в литрах с дискретностью в 0,1 литра
	   					record->fuel[1] = 0.1 * posb->LLSD;
	   			}
	   			else {
	   				record->fuel[0] = posb->LLSD;
	   				if( posb->FLG & 32 )	// показания ДУЖ в литрах с дискретностью в 0,1 литра
	   					record->fuel[0] = 0.1 * posb->LLSD;
	   			}
	   		}	// if( !(posb->FLG & B6) )
	   	}

	   	return b_size;
	   }
	*/
}
