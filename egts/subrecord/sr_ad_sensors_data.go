package subrecord

import (
	"encoding/binary"

	"github.com/LdDl/go-egts/egts/utils"
)

// SRAdSensorsData EGTS_SR_AD_SENSORS_DATA
/*
	Применяется абонентским терминалом для передачи
	на аппаратно-программный комплекс информации о состоянии
	дополнительных дискретных и аналоговых входов
*/
type SRAdSensorsData struct {
	// Digital Outputs
	DigitalOutputs uint8
	// Additional Digital Inputs Octets 1-8
	ADI []int
	// Analog Sensors 1-8
	ANS []int
}

// Decode Parse array of bytes to EGTS_SR_AD_SENSORS_DATA
func (subr *SRAdSensorsData) Decode(b []byte) (err error) {
	// buffer := new(bytes.Buffer)
	subr.ADI = make([]int, 8)
	subr.ANS = make([]int, 8)
	// Digital Outputs
	subr.DigitalOutputs = uint8(b[1])
	// DIOE1 ... DIOE8 - Digital Inputs Octet Exists
	dioeFlag := uint16(b[0])
	n := 3
	for i := 0; i < 8; i++ {
		if utils.BitField(dioeFlag, i).(bool) {
			subr.ADI[i] = int(b[n])
			n++
		} else {
			subr.ADI[i] = int(-1)
		}
	}
	// ASFE1 ... ASFE8 - (Analog Sensor Field Exists)
	asfeFlag := uint16(b[2])
	for i := 0; i < 8; i++ {
		if utils.BitField(asfeFlag, i).(bool) {
			b := append([]byte{0}, b[n:n+3]...)
			subr.ANS[i] = int(binary.LittleEndian.Uint32(b))
			n += 3
		} else {
			subr.ANS[i] = int(-1)
		}
	}
	return nil
}

// Encode Parse EGTS_SR_AD_SENSORS_DATA to array of bytes
func (subr *SRAdSensorsData) Encode() (b []byte) {
	return b
}

// Len Returns length of bytes slice
func (subr *SRAdSensorsData) Len() (l uint16) {
	l = uint16(len(subr.Encode()))
	return l
}
