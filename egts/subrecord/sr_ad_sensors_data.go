package subrecord

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"strconv"
	"strings"
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

	flagByteADI := byte(0)
	if flagByteADI, err = buffer.ReadByte(); err != nil {
		return fmt.Errorf("EGTS_SR_AD_SENSORS_DATA; Error reading flags ADI")
	}
	flagByteAsBitsADI := fmt.Sprintf("%08b", flagByteADI)

	// DIOE1 ... DIOE8 - Digital Inputs Octet Exists
	subr.DIOExists = make([]string, 8) // not [8]string{}, because slice is needed in Encode() method
	subr.DIOExists[0] = flagByteAsBitsADI[7:]
	subr.DIOExists[1] = flagByteAsBitsADI[6:7]
	subr.DIOExists[2] = flagByteAsBitsADI[5:6]
	subr.DIOExists[3] = flagByteAsBitsADI[4:5]
	subr.DIOExists[4] = flagByteAsBitsADI[3:4]
	subr.DIOExists[5] = flagByteAsBitsADI[2:3]
	subr.DIOExists[6] = flagByteAsBitsADI[1:2]
	subr.DIOExists[7] = flagByteAsBitsADI[:1]

	// Digital Outputs
	if subr.DigitalOutputs, err = buffer.ReadByte(); err != nil {
		return fmt.Errorf("EGTS_SR_AD_SENSORS_DATA; Error reading DOUT")
	}

	flagByteANS := byte(0)
	if flagByteANS, err = buffer.ReadByte(); err != nil {
		return fmt.Errorf("EGTS_SR_AD_SENSORS_DATA; Error reading flags ANS")
	}
	flagByteAsBitsANS := fmt.Sprintf("%08b", flagByteANS)
	// ASFE1 ... ASFE8 - (Analog Sensor Field Exists)
	subr.ANSExists = make([]string, 8) // not [8]string{}, because slice is needed in Encode() method
	subr.ANSExists[0] = flagByteAsBitsANS[7:]
	subr.ANSExists[1] = flagByteAsBitsANS[6:7]
	subr.ANSExists[2] = flagByteAsBitsANS[5:6]
	subr.ANSExists[3] = flagByteAsBitsANS[4:5]
	subr.ANSExists[4] = flagByteAsBitsANS[3:4]
	subr.ANSExists[5] = flagByteAsBitsANS[2:3]
	subr.ANSExists[6] = flagByteAsBitsANS[1:2]
	subr.ANSExists[7] = flagByteAsBitsANS[:1]

	for i := range subr.DIOExists {
		if subr.DIOExists[i] == "1" {
			if subr.ADI[i], err = buffer.ReadByte(); err != nil {
				return fmt.Errorf("EGTS_SR_AD_SENSORS_DATA; Error reading flags ADI")
			}
		}
	}

	for i := range subr.ANSExists {
		if subr.ANSExists[i] == "1" {
			ans := make([]byte, 3)
			if _, err = buffer.Read(ans); err != nil {
				return fmt.Errorf("EGTS_SR_AD_SENSORS_DATA; Error reading flags ANS")
			}
			ans = append(ans, 0x00)
			subr.ANS[i] = binary.LittleEndian.Uint32(ans)
		}
	}

	return nil
}

// Encode Parse EGTS_SR_AD_SENSORS_DATA to array of bytes
func (subr *SRAdSensorsData) Encode() (b []byte, err error) {
	buffer := new(bytes.Buffer)

	flagsDIO := uint64(0)

	for i := len(subr.DIOExists)/2 - 1; i >= 0; i-- { // reversed order of DIO
		opp := len(subr.DIOExists) - 1 - i
		subr.DIOExists[i], subr.DIOExists[opp] = subr.DIOExists[opp], subr.DIOExists[i]
	}

	flagsDIO, err = strconv.ParseUint(strings.Join(subr.DIOExists, ""), 2, 8)
	if err != nil {
		return nil, fmt.Errorf("EGTS_SR_AD_SENSORS_DATA; Error writing flags ADI")
	}
	if err = buffer.WriteByte(uint8(flagsDIO)); err != nil {
		return nil, fmt.Errorf("EGTS_SR_AD_SENSORS_DATA; Error writing byte flags ADI")
	}

	if err = buffer.WriteByte(subr.DigitalOutputs); err != nil {
		return nil, fmt.Errorf("EGTS_SR_AD_SENSORS_DATA; Error writing DOUT")
	}

	flagsANS := uint64(0)
	for i := len(subr.ANSExists)/2 - 1; i >= 0; i-- { // reversed order of ANS
		opp := len(subr.ANSExists) - 1 - i
		subr.ANSExists[i], subr.ANSExists[opp] = subr.ANSExists[opp], subr.ANSExists[i]
	}
	flagsANS, err = strconv.ParseUint(strings.Join(subr.ANSExists, ""), 2, 8)
	if err != nil {
		return nil, fmt.Errorf("EGTS_SR_AD_SENSORS_DATA; Error writing flags ANS")
	}
	if err = buffer.WriteByte(uint8(flagsANS)); err != nil {
		return nil, fmt.Errorf("EGTS_SR_AD_SENSORS_DATA; Error writing byte flags ANS")
	}

	for i := range subr.DIOExists {
		if subr.DIOExists[i] == "1" {
			if err = buffer.WriteByte(subr.ADI[i]); err != nil {
				return nil, fmt.Errorf("EGTS_SR_AD_SENSORS_DATA; Error writing ADI")
			}
		}
	}

	for i := range subr.ANSExists {
		if subr.ANSExists[i] == "1" {
			ans := make([]byte, 4)
			binary.LittleEndian.PutUint32(ans, subr.ANS[i])
			if _, err = buffer.Write(ans[:3]); err != nil {
				return nil, fmt.Errorf("EGTS_SR_AD_SENSORS_DATA; Error writing ANS")
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
