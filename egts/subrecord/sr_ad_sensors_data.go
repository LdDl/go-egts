package subrecord

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"io"
	"strconv"
)

// SRAdSensorsData EGTS_SR_AD_SENSORS_DATA
/*
	Применяется абонентским терминалом для передачи
	на аппаратно-программный комплекс информации о состоянии
	дополнительных дискретных и аналоговых входов
*/
type SRAdSensorsData struct {
	// Digital Outputs
	DigitalOutputs uint8 `json:"DOUT"`

	// Additional Digital Inputs Octets 1-8
	ADI [8]uint8 `json:"ADI"`
	// Analog Sensors 1-8
	ANS [8]uint32 `json:"ANS"`

	DIOExists []string `json:"DIOE"`
	ANSExists []string `json:"ANSE"`
}

// Decode Parse array of bytes to EGTS_SR_AD_SENSORS_DATA
func (subr *SRAdSensorsData) Decode(b []byte) (err error) {

	buffer := bytes.NewReader(b)
	decoded := SRAdSensorsData{}

	flagByteADI := byte(0)
	flagByteADI, err = buffer.ReadByte()
	if err != nil {
		return fmt.Errorf("EGTS_SR_AD_SENSORS_DATA; Error reading flags ADI: %w", err)
	}
	flagByteAsBitsADI := fmt.Sprintf("%08b", flagByteADI)

	// DIOE1 ... DIOE8 - Digital Inputs Octet Exists
	decoded.DIOExists = make([]string, 8)
	decoded.DIOExists[0] = flagByteAsBitsADI[7:]
	decoded.DIOExists[1] = flagByteAsBitsADI[6:7]
	decoded.DIOExists[2] = flagByteAsBitsADI[5:6]
	decoded.DIOExists[3] = flagByteAsBitsADI[4:5]
	decoded.DIOExists[4] = flagByteAsBitsADI[3:4]
	decoded.DIOExists[5] = flagByteAsBitsADI[2:3]
	decoded.DIOExists[6] = flagByteAsBitsADI[1:2]
	decoded.DIOExists[7] = flagByteAsBitsADI[:1]

	// Digital Outputs
	decoded.DigitalOutputs, err = buffer.ReadByte()
	if err != nil {
		return fmt.Errorf("EGTS_SR_AD_SENSORS_DATA; Error reading DOUT: %w", err)
	}

	flagByteANS := byte(0)
	flagByteANS, err = buffer.ReadByte()
	if err != nil {
		return fmt.Errorf("EGTS_SR_AD_SENSORS_DATA; Error reading flags ANS: %w", err)
	}
	flagByteAsBitsANS := fmt.Sprintf("%08b", flagByteANS)
	// ASFE1 ... ASFE8 - (Analog Sensor Field Exists)
	decoded.ANSExists = make([]string, 8)
	decoded.ANSExists[0] = flagByteAsBitsANS[7:]
	decoded.ANSExists[1] = flagByteAsBitsANS[6:7]
	decoded.ANSExists[2] = flagByteAsBitsANS[5:6]
	decoded.ANSExists[3] = flagByteAsBitsANS[4:5]
	decoded.ANSExists[4] = flagByteAsBitsANS[3:4]
	decoded.ANSExists[5] = flagByteAsBitsANS[2:3]
	decoded.ANSExists[6] = flagByteAsBitsANS[1:2]
	decoded.ANSExists[7] = flagByteAsBitsANS[:1]

	for i := range decoded.DIOExists {
		if decoded.DIOExists[i] == "1" {
			decoded.ADI[i], err = buffer.ReadByte()
			if err != nil {
				return fmt.Errorf("EGTS_SR_AD_SENSORS_DATA; Error reading ADI%d: %w", i+1, err)
			}
		}
	}

	for i := range decoded.ANSExists {
		if decoded.ANSExists[i] == "1" {
			ans := make([]byte, 3)
			_, err = io.ReadFull(buffer, ans)
			if err != nil {
				return fmt.Errorf("EGTS_SR_AD_SENSORS_DATA; Error reading ANS%d: %w", i+1, err)
			}
			ans = append(ans, 0x00)
			decoded.ANS[i] = binary.LittleEndian.Uint32(ans)
		}
	}

	if buffer.Len() != 0 {
		return fmt.Errorf("EGTS_SR_AD_SENSORS_DATA; Unexpected trailing data")
	}
	*subr = decoded
	return nil
}

// Encode Parse EGTS_SR_AD_SENSORS_DATA to array of bytes
func (subr *SRAdSensorsData) Encode() (b []byte, err error) {
	if subr == nil {
		return nil, fmt.Errorf("SRAdSensorsData; Subrecord is nil")
	}
	buffer := new(bytes.Buffer)

	if len(subr.DIOExists) != len(subr.ADI) {
		return nil, fmt.Errorf("EGTS_SR_AD_SENSORS_DATA; Expected 8 ADI flags")
	}
	if len(subr.ANSExists) != len(subr.ANS) {
		return nil, fmt.Errorf("EGTS_SR_AD_SENSORS_DATA; Expected 8 ANS flags")
	}
	flagsBitsDIO := ""
	for i := len(subr.DIOExists) - 1; i >= 0; i-- {
		if subr.DIOExists[i] != "0" && subr.DIOExists[i] != "1" {
			return nil, fmt.Errorf("EGTS_SR_AD_SENSORS_DATA; Invalid ADI flag %d", i+1)
		}
		flagsBitsDIO += subr.DIOExists[i]
	}

	flagsDIO := uint64(0)
	flagsDIO, err = strconv.ParseUint(flagsBitsDIO, 2, 8)
	if err != nil {
		return nil, fmt.Errorf("EGTS_SR_AD_SENSORS_DATA; Error writing flags ADI: %w", err)
	}
	err = buffer.WriteByte(uint8(flagsDIO))
	if err != nil {
		return nil, fmt.Errorf("EGTS_SR_AD_SENSORS_DATA; Error writing byte flags ADI: %w", err)
	}

	err = buffer.WriteByte(subr.DigitalOutputs)
	if err != nil {
		return nil, fmt.Errorf("EGTS_SR_AD_SENSORS_DATA; Error writing DOUT: %w", err)
	}

	flagsBitsANS := ""
	for i := len(subr.ANSExists) - 1; i >= 0; i-- {
		if subr.ANSExists[i] != "0" && subr.ANSExists[i] != "1" {
			return nil, fmt.Errorf("EGTS_SR_AD_SENSORS_DATA; Invalid ANS flag %d", i+1)
		}
		flagsBitsANS += subr.ANSExists[i]
	}
	flagsANS := uint64(0)
	flagsANS, err = strconv.ParseUint(flagsBitsANS, 2, 8)
	if err != nil {
		return nil, fmt.Errorf("EGTS_SR_AD_SENSORS_DATA; Error writing flags ANS: %w", err)
	}
	err = buffer.WriteByte(uint8(flagsANS))
	if err != nil {
		return nil, fmt.Errorf("EGTS_SR_AD_SENSORS_DATA; Error writing byte flags ANS: %w", err)
	}

	for i := range subr.DIOExists {
		if subr.DIOExists[i] == "1" {
			err = buffer.WriteByte(subr.ADI[i])
			if err != nil {
				return nil, fmt.Errorf("EGTS_SR_AD_SENSORS_DATA; Error writing ADI%d: %w", i+1, err)
			}
		}
	}

	for i := range subr.ANSExists {
		if subr.ANSExists[i] == "1" {
			if subr.ANS[i] > 0xffffff {
				return nil, fmt.Errorf("EGTS_SR_AD_SENSORS_DATA; ANS%d exceeds 24 bits", i+1)
			}
			ans := make([]byte, 4)
			binary.LittleEndian.PutUint32(ans, subr.ANS[i])
			_, err = buffer.Write(ans[:3])
			if err != nil {
				return nil, fmt.Errorf("EGTS_SR_AD_SENSORS_DATA; Error writing ANS%d: %w", i+1, err)
			}
		}
	}

	return buffer.Bytes(), nil
}

// Len Returns length of bytes slice
func (subr *SRAdSensorsData) Len() (l uint16) {
	encoded, _ := subr.Encode()
	l = uint16(len(encoded))
	return l
}
