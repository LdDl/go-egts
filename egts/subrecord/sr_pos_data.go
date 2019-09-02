package subrecord

import (
	"encoding/binary"
	"time"

	"github.com/LdDl/go-egts/egts/utils"
)

// SRPosData EGTS_SR_POS_DATA
/*
	Используется абонентским терминалом при передаче
	основных данных определения местоположения
*/
type SRPosData struct {
	NavigationTimeUint uint32    `json:"NTM"`         // NTM , seconds since 00:00:00 01.01.2010 UTC
	NavigationTime     time.Time `json:"NTM_RFC3339"` // Navigation Time
	Latitude           float32   `json:"LAT"`         // LAT , degree,  (WGS - 84) / 90 * 0xFFFFFFFF
	Longitude          float32   `json:"LONG"`        // LONG , degree,  (WGS - 84) / 180 * 0xFFFFFFFF
	Speed              int       `json:"SPD"`         // SPD , 0.1 miles , 14 bit only
	Direction          uint8     `json:"DIR"`         // DIR Direction
	Odometer           int       `json:"ODM"`         // ODM Odometer, 3b
	DigitalInputs      uint8     `json:"DIN"`         // DIN Digital Inputs
	Source             uint8     `json:"SRC"`         // SRC Source
	Altitude           uint32    `json:"ALT"`         // ALT Altitude, 3b

	SourceDataExists uint8 `json:"SRCDE"` // SRCD exists
	SourceData       int16 `json:"SRCD"`  // SRDC Source Data

	// Flags
	Valid            bool `json:"VLD"`
	CoordinateSystem bool `json:"CS"`
	Fix              bool `json:"FIX"`
	BlackBox         bool `json:"BB"`
	Move             bool `json:"MV"`
	LAHS             bool `json:"LAHS"`
	LOHS             bool `json:"LOHS"`
	AltitudeFlag     bool `json:"ALTE"`
	AltsFlag         bool `json:"ALTS"` //		определяет высоту относительно уровня моря и имеет смысл только при установленном флаге ALTE: 0 - точка выше уровня моря; 1 - ниже уровня моря;
	DirhFlag         bool `json:"DIRH"` //	(Direction the Highest bit) старший бит (8) параметра DIR;

}

// Decode Parse array of bytes to EGTS_SR_POS_DATA
func (subr *SRPosData) Decode(b []byte) {
	// Navigation Time , seconds since 00:00:00 01.01.2010 UTC
	t1, _ := time.Parse(time.RFC3339, "2010-01-01T00:00:00+00:00")
	subr.NavigationTimeUint = binary.LittleEndian.Uint32(b[0:4])
	subr.NavigationTime = t1.Add(time.Duration(int(subr.NavigationTimeUint)) * time.Second)
	// Flags
	flagBytes := uint16(b[12])
	subr.Valid = utils.BitField(flagBytes, 0).(bool)
	subr.CoordinateSystem = utils.BitField(flagBytes, 1).(bool)
	subr.Fix = utils.BitField(flagBytes, 2).(bool)
	subr.BlackBox = utils.BitField(flagBytes, 3).(bool)
	subr.Move = utils.BitField(flagBytes, 4).(bool)
	subr.LAHS = utils.BitField(flagBytes, 5).(bool)
	subr.LOHS = utils.BitField(flagBytes, 6).(bool)
	subr.AltitudeFlag = utils.BitField(flagBytes, 7).(bool)
	if subr.Valid {
		// Latitude , degree,  (WGS - 84) / 90 * 0xFFFFFFFF
		subr.Latitude = 90 * float32(binary.LittleEndian.Uint32(b[4:8])) / float32(0xFFFFFFFF)
		if subr.LAHS == true {
			subr.Latitude = subr.Latitude * -1
		}
		// Longitude , degree,  (WGS - 84) / 180 * 0xFFFFFFFF
		subr.Longitude = 180 * float32(binary.LittleEndian.Uint32(b[8:12])) / float32(0xFFFFFFFF)
		if subr.LOHS == true {
			subr.Longitude = subr.Longitude * -1
		}
	}
	speedBytes := binary.LittleEndian.Uint16(b[13:15])
	subr.Speed = utils.BitField(speedBytes, 0, 1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 11, 12, 13).(int) / 10
	subr.AltsFlag = utils.BitField(speedBytes, 14).(bool)
	subr.DirhFlag = utils.BitField(speedBytes, 15).(bool)
	// DIR Direction
	subr.Direction = uint8(b[15])
	// ODM Odometer, 3b
	odm := append([]byte{0}, b[16:19]...)
	subr.Odometer = int(binary.LittleEndian.Uint32(odm)) / 10
	// DIN Digital Inputs
	subr.DigitalInputs = uint8(b[19])
	// SRC Source
	subr.Source = uint8(b[20])
	if subr.AltitudeFlag {
		alt := make([]byte, len(b[21:24]))
		copy(alt, b[21:24])
		alt = append(alt, byte(0))
		subr.Altitude = binary.LittleEndian.Uint32(alt)
	}
}

// Encode Parse EGTS_SR_POS_DATA to array of bytes
func (subr *SRPosData) Encode() (b []byte) {
	return b
}
