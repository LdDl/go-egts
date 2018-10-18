package subrecord

import (
	"encoding/binary"
	"time"

	"github.com/LdDl/go-egts/egts/utils"
)

//EgtsSrPosData - Используется абонентским терминалом при передаче
//основных данных определения местоположения
type EgtsSrPosData struct {
	NavigationTimeUint uint32    // NTM , seconds since 00:00:00 01.01.2010 UTC
	NavigationTime     time.Time // Navigation Time
	Latitude           float32   // LAT , degree,  (WGS - 84) / 90 * 0xFFFFFFFF
	Longitude          float32   // LONG , degree,  (WGS - 84) / 180 * 0xFFFFFFFF
	Speed              int       // SPD , 0.1 miles , 14 bit only
	Direction          uint8     // DIR Direction
	Odometer           int       // ODM Odometer, 3b
	DigitalInputs      uint8     // DIN Digital Inputs
	Source             uint8     // SRC Source
	Altitude           uint32    // ALT Altitude, 3b

	SourceDataExists uint8 // SRCD exists
	SourceData       int16 // SRDC Source Data

	// Flags
	valid            bool
	CoordinateSystem bool
	Fix              bool
	BlackBox         bool
	Move             bool
	lahs             bool
	lohs             bool
	altitudeFlag     bool
	AltsFlag         bool //		определяет высоту относительно уровня моря и имеет смысл только при установленном флаге ALTE: 0 - точка выше уровня моря; 1 - ниже уровня моря;
	DirhFlag         bool //	(Direction the Highest bit) старший бит (8) параметра DIR;

}

//ParseEgtsSrPosData - EGTS_SR_POS_DATA
//Positioning data
func ParseEgtsSrPosData(b []byte) interface{} {
	var d EgtsSrPosData
	// Navigation Time , seconds since 00:00:00 01.01.2010 UTC
	t1, _ := time.Parse(
		time.RFC3339,
		"2010-01-01T00:00:00+00:00")
	d.NavigationTimeUint = binary.LittleEndian.Uint32(b[0:4])
	d.NavigationTime = t1.Add(time.Duration(int(d.NavigationTimeUint)) * time.Second)
	// Flags
	flagBytes := uint16(b[12])
	d.valid = utils.BitField(flagBytes, 0).(bool)
	d.CoordinateSystem = utils.BitField(flagBytes, 1).(bool)
	d.Fix = utils.BitField(flagBytes, 2).(bool)
	d.BlackBox = utils.BitField(flagBytes, 3).(bool)
	d.Move = utils.BitField(flagBytes, 4).(bool)
	d.lahs = utils.BitField(flagBytes, 5).(bool)
	d.lohs = utils.BitField(flagBytes, 6).(bool)
	d.altitudeFlag = utils.BitField(flagBytes, 7).(bool)
	if d.valid {
		// Latitude , degree,  (WGS - 84) / 90 * 0xFFFFFFFF
		d.Latitude = 90 * float32(binary.LittleEndian.Uint32(b[4:8])) / float32(0xFFFFFFFF)
		if d.lahs == true {
			d.Latitude = d.Latitude * -1
		}
		// Longitude , degree,  (WGS - 84) / 180 * 0xFFFFFFFF
		d.Longitude = 180 * float32(binary.LittleEndian.Uint32(b[8:12])) / float32(0xFFFFFFFF)
		if d.lohs == true {
			d.Longitude = d.Longitude * -1
		}
	}
	speedBytes := binary.LittleEndian.Uint16(b[13:15])
	d.Speed = utils.BitField(speedBytes, 0, 1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 11, 12, 13).(int) / 10
	d.AltsFlag = utils.BitField(speedBytes, 14).(bool)
	d.DirhFlag = utils.BitField(speedBytes, 15).(bool)
	// DIR Direction
	d.Direction = uint8(b[15])
	// ODM Odometer, 3b
	odm := append([]byte{0}, b[16:19]...)
	d.Odometer = int(binary.LittleEndian.Uint32(odm)) / 10
	// DIN Digital Inputs
	d.DigitalInputs = uint8(b[19])
	// SRC Source
	d.Source = uint8(b[20])
	if d.altitudeFlag {
		alt := b[21:24]
		alt = append(alt, byte(0))
		d.Altitude = binary.LittleEndian.Uint32(alt)
	}
	return d
}
