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

// BytesToSRAdSensorsData Parse array of bytes to EGTS_SR_AD_SENSORS_DATA
func BytesToSRAdSensorsData(b []byte) interface{} {
	var d SRAdSensorsData
	d.ADI = make([]int, 8)
	d.ANS = make([]int, 8)
	// Digital Outputs
	d.DigitalOutputs = uint8(b[1])
	// DIOE1 ... DIOE8 - Digital Inputs Octet Exists
	dioeFlag := uint16(b[0])
	n := 3
	for i := 0; i < 8; i++ {
		if utils.BitField(dioeFlag, i).(bool) {
			d.ADI[i] = int(b[n])
			n++
		} else {
			d.ADI[i] = int(-1)
		}
	}
	// ASFE1 ... ASFE8 - (Analog Sensor Field Exists)
	asfeFlag := uint16(b[2])
	for i := 0; i < 8; i++ {
		if utils.BitField(asfeFlag, i).(bool) {
			b := append([]byte{0}, b[n:n+3]...)
			d.ANS[i] = int(binary.LittleEndian.Uint32(b))
			n += 3
		} else {
			d.ANS[i] = int(-1)
		}
	}
	return d
}
