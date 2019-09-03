package subrecord

import (
	"bytes"
	"encoding/binary"
	"fmt"
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
	Latitude           float64   `json:"LAT"`         // LAT , degree,  (WGS - 84) / 90 * 0xFFFFFFFF
	Longitude          float64   `json:"LONG"`        // LONG , degree,  (WGS - 84) / 180 * 0xFFFFFFFF
	Speed              int       `json:"SPD"`         // SPD , 0.1 miles , 14 bit only
	Direction          uint8     `json:"DIR"`         // DIR Direction
	Odometer           int       `json:"ODM"`         // ODM Odometer, 3b
	DigitalInputs      uint8     `json:"DIN"`         // DIN Digital Inputs
	Source             uint8     `json:"SRC"`         // SRC Source
	Altitude           uint32    `json:"ALT"`         // ALT Altitude, 3b

	SourceDataExists uint8 `json:"SRCDE"` // SRCD exists
	SourceData       int16 `json:"SRCD"`  // SRDC Source Data

	// Flags
	Valid            string `json:"VLD"`
	CoordinateSystem string `json:"CS"`
	Fix              string `json:"FIX"`
	BlackBox         string `json:"BB"`
	Move             string `json:"MV"`
	LAHS             string `json:"LAHS"`
	LOHS             string `json:"LOHS"`
	AltitudeExists   string `json:"ALTE"`
	AltsFlag         bool   `json:"ALTS"` //		определяет высоту относительно уровня моря и имеет смысл только при установленном флаге ALTE: 0 - точка выше уровня моря; 1 - ниже уровня моря;
	DirhFlag         bool   `json:"DIRH"` //	(Direction the Highest bit) старший бит (8) параметра DIR;

}

// Decode Parse array of bytes to EGTS_SR_POS_DATA
func (subr *SRPosData) Decode(b []byte) (err error) {
	buffer := bytes.NewReader(b)

	// Navigation Time , seconds since 00:00:00 01.01.2010 UTC - specification from EGTS
	timestamp, _ := time.Parse(time.RFC3339, "2010-01-01T00:00:00+00:00")
	nt := make([]byte, 4)
	if _, err = buffer.Read(nt); err != nil {
		return fmt.Errorf("Error reading NT")
	}
	subr.NavigationTimeUint = binary.LittleEndian.Uint32(nt)
	subr.NavigationTime = timestamp.Add(time.Duration(int(subr.NavigationTimeUint)) * time.Second)

	lat := make([]byte, 4)
	if _, err = buffer.Read(lat); err != nil {
		return fmt.Errorf("Error reading latitude")
	}

	lon := make([]byte, 4)
	if _, err = buffer.Read(lon); err != nil {
		return fmt.Errorf("Error reading longitude")
	}
	// Longitude , degree,  (WGS - 84) / 180 * 0xFFFFFFFF
	subr.Longitude = 180.0 * float64(binary.LittleEndian.Uint32(lon)) / 0xFFFFFFFF

	// Flags
	flagByte := byte(0)
	if flagByte, err = buffer.ReadByte(); err != nil {
		return fmt.Errorf("Error reading flags")
	}
	flagByteAsBits := fmt.Sprintf("%08b", flagByte)
	subr.Valid = flagByteAsBits[7:]
	subr.Fix = flagByteAsBits[6:7]
	subr.CoordinateSystem = flagByteAsBits[5:6]
	subr.BlackBox = flagByteAsBits[4:5]
	subr.Move = flagByteAsBits[3:4]
	subr.LAHS = flagByteAsBits[2:3]
	subr.LOHS = flagByteAsBits[1:2]
	subr.AltitudeExists = flagByteAsBits[:1]

	if subr.Valid == "1" {
		// Latitude , degree,  (WGS - 84) / 90 * 0xFFFFFFFF
		subr.Latitude = 90.0 * float64(binary.LittleEndian.Uint32(lat)) / 0xFFFFFFFF
		if subr.LAHS == "1" {
			subr.Latitude = subr.Latitude * -1
		}
		// Longitude , degree,  (WGS - 84) / 180 * 0xFFFFFFFF
		subr.Longitude = 180.0 * float64(binary.LittleEndian.Uint32(lon)) / 0xFFFFFFFF
		if subr.LOHS == "1" {
			subr.Longitude = subr.Longitude * -1
		}
	}

	speedBytes := make([]byte, 2)
	if _, err = buffer.Read(speedBytes); err != nil {
		return fmt.Errorf("Error reading speed")
	}

	speed := binary.LittleEndian.Uint16(speedBytes)
	subr.Speed = utils.BitField(speed, 0, 1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 11, 12, 13).(int) / 10

	subr.AltsFlag = utils.BitField(speed, 14).(bool)
	subr.DirhFlag = utils.BitField(speed, 15).(bool)
	// DIR Direction
	if subr.Direction, err = buffer.ReadByte(); err != nil {
		return fmt.Errorf("Error reading direction")
	}

	// ODM Odometer, 3b
	odm := make([]byte, 3)
	if _, err = buffer.Read(odm); err != nil {
		return fmt.Errorf("Error reading odm")
	}
	// odm := append([]byte{0}, b[16:19]...)
	subr.Odometer = int(binary.LittleEndian.Uint32(odm)) / 10

	// DIN Digital Inputs
	if subr.DigitalInputs, err = buffer.ReadByte(); err != nil {
		return fmt.Errorf("Error reading digital input")
	}

	// SRC Source
	if subr.Source, err = buffer.ReadByte(); err != nil {
		return fmt.Errorf("Error reading source")
	}

	if subr.AltitudeExists == "1" {
		alt := make([]byte, 3)
		if _, err = buffer.Read(alt); err != nil {
			return fmt.Errorf("Error reading altitude")
		}
		// alt := make([]byte, len(b[21:24]))
		// copy(alt, b[21:24])
		// alt = append(alt, byte(0))
		subr.Altitude = binary.LittleEndian.Uint32(alt)
	}

	return nil
}

// Encode Parse EGTS_SR_POS_DATA to array of bytes
func (subr *SRPosData) Encode() (b []byte) {
	return b
}

// Len Returns length of bytes slice
func (subr *SRPosData) Len() (l uint16) {
	l = uint16(len(subr.Encode()))
	return l
}
