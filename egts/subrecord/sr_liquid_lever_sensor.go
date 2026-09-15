package subrecord

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"io"
	"strconv"
	"strings"
)

// SRLiquidLevelSensor EGTS_SR_LIQUID_LEVEL_SENSOR
/*
	Применяется абонентским терминалом
	для передачи на аппаратно-программный комплекс
	данных о показаниях ДУЖ
*/
type SRLiquidLevelSensor struct {
	// LLSN Liquid Level Sensor Number
	LiquidLevelSensorNumber uint8 `json:"LLSN"`
	// RDF bit 3 определяет формат поля LLSD.
	RawbFlag string `json:"RDF"`
	// LLSVU bits 4-5 определяют единицы измерения показаний ДУЖ.
	LiquidLevelSensorValueUnit string `json:"LLSVU"`
	// LLSEF bit 6 указывает на ошибку считывания показаний ДУЖ.
	LiquidLevelSensorErrorFlag string `json:"LLSEF"`
	Flags                      uint8
	// MADDR адрес модуля, данные о показаниях ДУЖ с которого поступили в терминал.
	MADDR uint16 `json:"MADDR"`
	// LLSD числовые показания ДУЖ при RDF = 0.
	LiquidLevelSensorb uint32 `json:"LLSD"`
	// LLSD исходные данные ДУЖ при RDF = 1, от 4 до 512 байт.
	RawData []byte `json:"LLSD_RAW"`
}

// Decode Parse array of bytes to EGTS_SR_LIQUID_LEVEL_SENSOR
func (subr *SRLiquidLevelSensor) Decode(b []byte) (err error) {
	buffer := bytes.NewReader(b)
	decoded := SRLiquidLevelSensor{}

	flagByte := byte(0)
	flagByte, err = buffer.ReadByte()
	if err != nil {
		return fmt.Errorf("EGTS_SR_LIQUID_LEVEL_SENSOR; Error reading flags: %w", err)
	}
	flagByteAsBits := fmt.Sprintf("%08b", flagByte)
	decoded.LiquidLevelSensorErrorFlag = flagByteAsBits[1:2]
	decoded.LiquidLevelSensorValueUnit = flagByteAsBits[2:4]
	decoded.RawbFlag = flagByteAsBits[4:5]

	llsn, err := strconv.ParseUint(flagByteAsBits[5:], 2, 8)
	if err != nil {
		return fmt.Errorf("EGTS_SR_LIQUID_LEVEL_SENSOR; Error parsing LLSN: %w", err)
	}
	decoded.LiquidLevelSensorNumber = uint8(llsn)

	maddr := make([]byte, 2)
	_, err = io.ReadFull(buffer, maddr)
	if err != nil {
		return fmt.Errorf("EGTS_SR_LIQUID_LEVEL_SENSOR; Error reading MADDR: %w", err)
	}
	decoded.MADDR = binary.LittleEndian.Uint16(maddr)

	if decoded.RawbFlag == "1" {
		if buffer.Len() < 4 || buffer.Len() > 512 {
			return fmt.Errorf("EGTS_SR_LIQUID_LEVEL_SENSOR; Raw LLSD length must be between 4 and 512 bytes")
		}
		decoded.RawData = make([]byte, buffer.Len())
		_, err = io.ReadFull(buffer, decoded.RawData)
		if err != nil {
			return fmt.Errorf("EGTS_SR_LIQUID_LEVEL_SENSOR; Error reading raw LLSD: %w", err)
		}
	} else {
		sb := make([]byte, 4)
		_, err = io.ReadFull(buffer, sb)
		if err != nil {
			return fmt.Errorf("EGTS_SR_LIQUID_LEVEL_SENSOR; Error reading LLSD: %w", err)
		}
		decoded.LiquidLevelSensorb = binary.LittleEndian.Uint32(sb)
	}

	if buffer.Len() != 0 {
		return fmt.Errorf("EGTS_SR_LIQUID_LEVEL_SENSOR; Unexpected trailing data")
	}
	*subr = decoded
	return nil
}

// Encode Parse EGTS_SR_LIQUID_LEVEL_SENSOR to array of bytes
func (subr *SRLiquidLevelSensor) Encode() (b []byte, err error) {
	if subr == nil {
		return nil, fmt.Errorf("SRLiquidLevelSensor; Subrecord is nil")
	}
	if subr.LiquidLevelSensorNumber > 7 {
		return nil, fmt.Errorf("EGTS_SR_LIQUID_LEVEL_SENSOR; LLSN must be between 0 and 7")
	}
	if subr.RawbFlag != "0" && subr.RawbFlag != "1" {
		return nil, fmt.Errorf("EGTS_SR_LIQUID_LEVEL_SENSOR; Invalid RDF flag")
	}
	if subr.LiquidLevelSensorErrorFlag != "0" && subr.LiquidLevelSensorErrorFlag != "1" {
		return nil, fmt.Errorf("EGTS_SR_LIQUID_LEVEL_SENSOR; Invalid LLSEF flag")
	}
	if len(subr.LiquidLevelSensorValueUnit) != 2 {
		return nil, fmt.Errorf("EGTS_SR_LIQUID_LEVEL_SENSOR; LLSVU must contain 2 bits")
	}
	if subr.RawbFlag == "1" && (len(subr.RawData) < 4 || len(subr.RawData) > 512) {
		return nil, fmt.Errorf("EGTS_SR_LIQUID_LEVEL_SENSOR; Raw LLSD length must be between 4 and 512 bytes")
	}

	buffer := new(bytes.Buffer)

	flags := uint64(0)
	flags, err = strconv.ParseUint(strings.Repeat("0", 1)+subr.LiquidLevelSensorErrorFlag+subr.LiquidLevelSensorValueUnit+subr.RawbFlag+fmt.Sprintf("%03b", subr.LiquidLevelSensorNumber), 2, 8)
	if err != nil {
		return nil, fmt.Errorf("EGTS_SR_LIQUID_LEVEL_SENSOR; Error parsing flags: %w", err)
	}
	err = buffer.WriteByte(uint8(flags))
	if err != nil {
		return nil, fmt.Errorf("EGTS_SR_LIQUID_LEVEL_SENSOR; Error writing byte flags: %w", err)
	}

	err = binary.Write(buffer, binary.LittleEndian, subr.MADDR)
	if err != nil {
		return nil, fmt.Errorf("EGTS_SR_LIQUID_LEVEL_SENSOR; Error writing MADDR: %w", err)
	}

	if subr.RawbFlag == "1" {
		_, err = buffer.Write(subr.RawData)
		if err != nil {
			return nil, fmt.Errorf("EGTS_SR_LIQUID_LEVEL_SENSOR; Error writing raw LLSD: %w", err)
		}
	} else {
		err = binary.Write(buffer, binary.LittleEndian, subr.LiquidLevelSensorb)
		if err != nil {
			return nil, fmt.Errorf("EGTS_SR_LIQUID_LEVEL_SENSOR; Error writing LLSD: %w", err)
		}
	}

	return buffer.Bytes(), nil
}

// Len Returns length of bytes slice
func (subr *SRLiquidLevelSensor) Len() (l uint16) {
	encoded, _ := subr.Encode()
	l = uint16(len(encoded))
	return l
}
