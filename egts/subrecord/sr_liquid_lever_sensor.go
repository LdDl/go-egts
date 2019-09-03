package subrecord

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"strconv"
)

// SRLiquidLevelSensor EGTS_SR_LIQUID_LEVEL_SENSOR
/*
	Применяется абонентским терминалом
	для передачи на аппаратно-программный комплекс
	данных о показаниях ДУЖ
*/
type SRLiquidLevelSensor struct {
	LiquidLevelSensorNumber    uint8  `json:"LLSN"`  // LLSN Liquid Level Sensor Number
	RawbFlag                   string `json:"RDF"`   // RDF bit 3 флаг, определяющий формат поля LLSD данной подзаписи:
	LiquidLevelSensorValueUnit string `json:"LLSVU"` // LLSVU	bit 4-5 битовый флаг, определяющий единицы измерения показаний ДУЖ:
	LiquidLevelSensorErrorFlag string `json:"LLSEF"` // LLSEF	bit 7	битовый флаг, определяющий наличие ошибок при считывании значения датчика уровня жидкости
	Flags                      uint8
	MADDR                      uint16 `json:"MADDR"` // MAC Address  адрес модуля, данные о показаниях ДУЖ с которого поступили в абонентский терминал
	LiquidLevelSensorb         uint32 `json:"LLSD"`  // LLSD показания ДУЖ в формате, определяемом флагом RDF
}

// Decode Parse array of bytes to EGTS_SR_LIQUID_LEVEL_SENSOR
func (subr *SRLiquidLevelSensor) Decode(b []byte) (err error) {
	buffer := bytes.NewReader(b)

	flagByte := byte(0)
	if flagByte, err = buffer.ReadByte(); err != nil {
		return fmt.Errorf("Error reading flags")
	}
	flagByteAsBits := fmt.Sprintf("%08b", flagByte)
	subr.LiquidLevelSensorErrorFlag = flagByteAsBits[1:2]
	subr.LiquidLevelSensorValueUnit = flagByteAsBits[2:4]
	subr.RawbFlag = flagByteAsBits[4:5]

	llsn, _ := strconv.ParseUint(flagByteAsBits[5:], 2, 8)
	subr.LiquidLevelSensorNumber = uint8(llsn)

	maddr := make([]byte, 2)
	if _, err = buffer.Read(maddr); err != nil {
		return fmt.Errorf("Error reading maddr")
	}
	subr.MADDR = binary.LittleEndian.Uint16(maddr)

	sb := make([]byte, 4)
	if _, err = buffer.Read(sb); err != nil {
		return fmt.Errorf("Error reading liquid data")
	}
	subr.LiquidLevelSensorb = binary.LittleEndian.Uint32(sb)

	return nil
}

// Encode Parse EGTS_SR_LIQUID_LEVEL_SENSOR to array of bytes
func (subr *SRLiquidLevelSensor) Encode() (b []byte) {
	return b
}

// Len Returns length of bytes slice
func (subr *SRLiquidLevelSensor) Len() (l uint16) {
	l = uint16(len(subr.Encode()))
	return l
}
