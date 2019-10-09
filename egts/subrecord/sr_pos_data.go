package subrecord

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"strconv"
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
	Odometer           int       `json:"ODM_value"`   // ODM Odometer, 3b
	OdometerBytes      []byte    `json:"ODM"`
	DigitalInputs      uint8     `json:"DIN"`       // DIN Digital Inputs
	Source             uint8     `json:"SRC"`       // SRC Source
	Altitude           uint32    `json:"ALT_value"` // ALT Altitude, 3b
	AltitudeBytes      []byte    `json:"ALT"`

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
	AltsFlag         uint8  `json:"ALTS"` //		определяет высоту относительно уровня моря и имеет смысл только при установленном флаге ALTE: 0 - точка выше уровня моря; 1 - ниже уровня моря;
	DirhFlag         uint8  `json:"DIRH"` //	(Direction the Highest bit) старший бит (8) параметра DIR;

}

// Decode Parse array of bytes to EGTS_SR_POS_DATA
func (subr *SRPosData) Decode(b []byte) (err error) {
	buffer := bytes.NewReader(b)

	// Navigation Time , seconds since 00:00:00 01.01.2010 UTC - specification from EGTS
	timestamp, _ := time.Parse(time.RFC3339, "2010-01-01T00:00:00+00:00")
	nt := make([]byte, 4)
	if _, err = buffer.Read(nt); err != nil {
		return fmt.Errorf("EGTS_SR_POS_DATA; Error reading NTM")
	}
	subr.NavigationTimeUint = binary.LittleEndian.Uint32(nt)
	subr.NavigationTime = timestamp.Add(time.Duration(int(subr.NavigationTimeUint)) * time.Second)

	lat := make([]byte, 4)
	if _, err = buffer.Read(lat); err != nil {
		return fmt.Errorf("EGTS_SR_POS_DATA; Error reading LAT")
	}

	lon := make([]byte, 4)
	if _, err = buffer.Read(lon); err != nil {
		return fmt.Errorf("EGTS_SR_POS_DATA; Error reading LONG")
	}
	// Longitude , degree,  (WGS - 84) / 180 * 0xFFFFFFFF
	subr.Longitude = 180.0 * float64(binary.LittleEndian.Uint32(lon)) / 0xFFFFFFFF

	// Flags
	flagByte := byte(0)
	if flagByte, err = buffer.ReadByte(); err != nil {
		return fmt.Errorf("EGTS_SR_POS_DATA; Error reading flags")
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
		return fmt.Errorf("EGTS_SR_POS_DATA; Error reading SPD")
	}
	speed := binary.LittleEndian.Uint16(speedBytes)
	subr.Speed = utils.BitField(speed, 0, 1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 11, 12, 13).(int) / 10

	subr.AltsFlag = uint8(speed >> 14 & 0x1)
	subr.DirhFlag = uint8(speed >> 15 & 0x1)
	// DIR Direction
	if subr.Direction, err = buffer.ReadByte(); err != nil {
		return fmt.Errorf("EGTS_SR_POS_DATA; Error reading DIR")
	}

	// ODM Odometer, 3b
	subr.OdometerBytes = make([]byte, 3)
	if _, err = buffer.Read(subr.OdometerBytes); err != nil {
		return fmt.Errorf("EGTS_SR_POS_DATA; Error reading ODM")
	}
	subr.Odometer = int(binary.LittleEndian.Uint32(append([]byte{0}, subr.OdometerBytes...))) / 10
	// DIN Digital Inputs
	if subr.DigitalInputs, err = buffer.ReadByte(); err != nil {
		return fmt.Errorf("EGTS_SR_POS_DATA; Error reading DIN")
	}

	// SRC Source
	if subr.Source, err = buffer.ReadByte(); err != nil {
		return fmt.Errorf("EGTS_SR_POS_DATA; Error reading SRC")
	}

	if subr.AltitudeExists == "1" {
		subr.AltitudeBytes = make([]byte, 3)
		if _, err = buffer.Read(subr.AltitudeBytes); err != nil {
			return fmt.Errorf("EGTS_SR_POS_DATA; Error reading ALT")
		}
		subr.Altitude = binary.LittleEndian.Uint32(append(subr.AltitudeBytes, byte(0)))
	}

	return nil
}

// Encode Parse EGTS_SR_POS_DATA to array of bytes
func (subr *SRPosData) Encode() (b []byte, err error) {
	buffer := new(bytes.Buffer)
	timestamp, _ := time.Parse(time.RFC3339, "2010-01-01T00:00:00+00:00")
	if err = binary.Write(buffer, binary.LittleEndian, uint32(subr.NavigationTime.Sub(timestamp).Seconds())); err != nil {
		return nil, fmt.Errorf("EGTS_SR_POS_DATA; Error writing NTM")
	}
	if err = binary.Write(buffer, binary.LittleEndian, uint32(subr.Latitude/90*0xFFFFFFFF)); err != nil {
		return nil, fmt.Errorf("EGTS_SR_POS_DATA; Error writing LAT")
	}
	if err = binary.Write(buffer, binary.LittleEndian, uint32(subr.Longitude/180*0xFFFFFFFF)); err != nil {
		return nil, fmt.Errorf("EGTS_SR_POS_DATA; Error writing LONG")
	}
	flags := uint64(0)
	flags, err = strconv.ParseUint(subr.AltitudeExists+subr.LOHS+subr.LAHS+subr.Move+subr.BlackBox+subr.CoordinateSystem+subr.Fix+subr.Valid, 2, 8)
	if err != nil {
		return nil, fmt.Errorf("EGTS_SR_POS_DATA; Error writing flags")
	}
	if err = buffer.WriteByte(uint8(flags)); err != nil {
		return nil, fmt.Errorf("EGTS_SR_POS_DATA; Error writing flags byte")
	}

	speed := uint16(subr.Speed*10) | uint16(subr.DirhFlag)<<15
	speed = speed | uint16(subr.AltsFlag)<<14
	spd := make([]byte, 2)
	binary.LittleEndian.PutUint16(spd, speed)
	if _, err = buffer.Write(spd); err != nil {
		return nil, fmt.Errorf("EGTS_SR_POS_DATA; Error writing SPD")
	}

	dir := subr.Direction
	if err = binary.Write(buffer, binary.LittleEndian, dir); err != nil {
		return nil, fmt.Errorf("EGTS_SR_POS_DATA; Error writing DIR")
	}

	if _, err = buffer.Write(subr.OdometerBytes); err != nil {
		return nil, fmt.Errorf("EGTS_SR_POS_DATA; Error writing ODM")
	}

	if err = binary.Write(buffer, binary.LittleEndian, subr.DigitalInputs); err != nil {
		return nil, fmt.Errorf("EGTS_SR_POS_DATA; Error writing DIN")
	}

	if err = binary.Write(buffer, binary.LittleEndian, subr.Source); err != nil {
		return nil, fmt.Errorf("EGTS_SR_POS_DATA; Error writing SRC")
	}

	if subr.AltitudeExists == "1" {
		if _, err = buffer.Write(subr.AltitudeBytes); err != nil {
			return nil, fmt.Errorf("EGTS_SR_POS_DATA; Error writing ALT")
		}
	}

	return buffer.Bytes(), nil
}

// Len Returns length of bytes slice
func (subr *SRPosData) Len() (l uint16) {
	encoded, _ := subr.Encode()
	l = uint16(len(encoded))
	return l
}
