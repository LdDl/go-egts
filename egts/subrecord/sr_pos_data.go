package subrecord

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"io"
	"math"
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
	DirectionValue     uint16    `json:"DIR_value"`
	Odometer           int       `json:"ODM_value"` // ODM Odometer, 3b
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
	decoded := SRPosData{}

	// Navigation Time , seconds since 00:00:00 01.01.2010 UTC - specification from EGTS
	timestamp := time.Date(2010, time.January, 1, 0, 0, 0, 0, time.UTC)
	nt := make([]byte, 4)
	_, err = io.ReadFull(buffer, nt)
	if err != nil {
		return fmt.Errorf("EGTS_SR_POS_DATA; Error reading NTM: %w", err)
	}
	decoded.NavigationTimeUint = binary.LittleEndian.Uint32(nt)
	decoded.NavigationTime = timestamp.Add(time.Duration(decoded.NavigationTimeUint) * time.Second)

	lat := make([]byte, 4)
	_, err = io.ReadFull(buffer, lat)
	if err != nil {
		return fmt.Errorf("EGTS_SR_POS_DATA; Error reading LAT: %w", err)
	}

	lon := make([]byte, 4)
	_, err = io.ReadFull(buffer, lon)
	if err != nil {
		return fmt.Errorf("EGTS_SR_POS_DATA; Error reading LONG: %w", err)
	}

	// Flags
	flagByte := byte(0)
	flagByte, err = buffer.ReadByte()
	if err != nil {
		return fmt.Errorf("EGTS_SR_POS_DATA; Error reading flags: %w", err)
	}
	flagByteAsBits := fmt.Sprintf("%08b", flagByte)
	decoded.Valid = flagByteAsBits[7:]
	decoded.Fix = flagByteAsBits[6:7]
	decoded.CoordinateSystem = flagByteAsBits[5:6]
	decoded.BlackBox = flagByteAsBits[4:5]
	decoded.Move = flagByteAsBits[3:4]
	decoded.LAHS = flagByteAsBits[2:3]
	decoded.LOHS = flagByteAsBits[1:2]
	decoded.AltitudeExists = flagByteAsBits[:1]

	// Latitude , degree,  (WGS - 84) / 90 * 0xFFFFFFFF
	decoded.Latitude = 90.0 * float64(binary.LittleEndian.Uint32(lat)) / 0xFFFFFFFF
	if decoded.LAHS == "1" {
		decoded.Latitude = decoded.Latitude * -1
	}
	// Longitude , degree,  (WGS - 84) / 180 * 0xFFFFFFFF
	decoded.Longitude = 180.0 * float64(binary.LittleEndian.Uint32(lon)) / 0xFFFFFFFF
	if decoded.LOHS == "1" {
		decoded.Longitude = decoded.Longitude * -1
	}

	speedBytes := make([]byte, 2)
	_, err = io.ReadFull(buffer, speedBytes)
	if err != nil {
		return fmt.Errorf("EGTS_SR_POS_DATA; Error reading SPD: %w", err)
	}
	speed := binary.LittleEndian.Uint16(speedBytes)
	decoded.Speed = utils.BitField(speed, 0, 1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 11, 12, 13).(int) / 10

	decoded.AltsFlag = uint8(speed >> 14 & 0x1)
	decoded.DirhFlag = uint8(speed >> 15 & 0x1)
	// DIR Direction
	decoded.Direction, err = buffer.ReadByte()
	if err != nil {
		return fmt.Errorf("EGTS_SR_POS_DATA; Error reading DIR: %w", err)
	}
	decoded.DirectionValue = uint16(decoded.Direction) | uint16(decoded.DirhFlag)<<8
	// Don't know why it was here, but just keep for the history:
	// subr.Direction = subr.Direction | subr.DirhFlag<<7

	// ODM Odometer, 3b
	decoded.OdometerBytes = make([]byte, 3)
	_, err = io.ReadFull(buffer, decoded.OdometerBytes)
	if err != nil {
		return fmt.Errorf("EGTS_SR_POS_DATA; Error reading ODM: %w", err)
	}
	decoded.Odometer = int(binary.LittleEndian.Uint32(append(decoded.OdometerBytes, 0))) / 10
	// DIN Digital Inputs
	decoded.DigitalInputs, err = buffer.ReadByte()
	if err != nil {
		return fmt.Errorf("EGTS_SR_POS_DATA; Error reading DIN: %w", err)
	}

	// SRC Source
	decoded.Source, err = buffer.ReadByte()
	if err != nil {
		return fmt.Errorf("EGTS_SR_POS_DATA; Error reading SRC: %w", err)
	}

	if decoded.AltitudeExists == "1" {
		decoded.AltitudeBytes = make([]byte, 3)
		_, err = io.ReadFull(buffer, decoded.AltitudeBytes)
		if err != nil {
			return fmt.Errorf("EGTS_SR_POS_DATA; Error reading ALT: %w", err)
		}
		decoded.Altitude = binary.LittleEndian.Uint32(append(decoded.AltitudeBytes, 0))
	}

	*subr = decoded
	return nil
}

// Encode Parse EGTS_SR_POS_DATA to array of bytes
func (subr *SRPosData) Encode() (b []byte, err error) {
	if subr == nil {
		return nil, fmt.Errorf("SRPosData; Subrecord is nil")
	}
	buffer := new(bytes.Buffer)
	timestamp := time.Date(2010, time.January, 1, 0, 0, 0, 0, time.UTC)
	err = binary.Write(buffer, binary.LittleEndian, uint32(subr.NavigationTime.Sub(timestamp).Seconds()))
	if err != nil {
		return nil, fmt.Errorf("EGTS_SR_POS_DATA; Error writing NTM: %w", err)
	}
	err = binary.Write(buffer, binary.LittleEndian, uint32(math.Abs(subr.Latitude)/90*0xFFFFFFFF))
	if err != nil {
		return nil, fmt.Errorf("EGTS_SR_POS_DATA; Error writing LAT: %w", err)
	}
	err = binary.Write(buffer, binary.LittleEndian, uint32(math.Abs(subr.Longitude)/180*0xFFFFFFFF))
	if err != nil {
		return nil, fmt.Errorf("EGTS_SR_POS_DATA; Error writing LONG: %w", err)
	}
	flags := uint64(0)
	flags, err = strconv.ParseUint(subr.AltitudeExists+subr.LOHS+subr.LAHS+subr.Move+subr.BlackBox+subr.CoordinateSystem+subr.Fix+subr.Valid, 2, 8)
	if err != nil {
		return nil, fmt.Errorf("EGTS_SR_POS_DATA; Error writing flags: %w", err)
	}
	err = buffer.WriteByte(uint8(flags))
	if err != nil {
		return nil, fmt.Errorf("EGTS_SR_POS_DATA; Error writing flags byte: %w", err)
	}

	speed := uint16(subr.Speed*10) | uint16(subr.DirhFlag)<<15
	speed = speed | uint16(subr.AltsFlag)<<14
	spd := make([]byte, 2)
	binary.LittleEndian.PutUint16(spd, speed)
	_, err = buffer.Write(spd)
	if err != nil {
		return nil, fmt.Errorf("EGTS_SR_POS_DATA; Error writing SPD: %w", err)
	}

	dir := subr.Direction
	err = binary.Write(buffer, binary.LittleEndian, dir)
	if err != nil {
		return nil, fmt.Errorf("EGTS_SR_POS_DATA; Error writing DIR: %w", err)
	}

	_, err = buffer.Write(subr.OdometerBytes)
	if err != nil {
		return nil, fmt.Errorf("EGTS_SR_POS_DATA; Error writing ODM: %w", err)
	}

	err = binary.Write(buffer, binary.LittleEndian, subr.DigitalInputs)
	if err != nil {
		return nil, fmt.Errorf("EGTS_SR_POS_DATA; Error writing DIN: %w", err)
	}

	err = binary.Write(buffer, binary.LittleEndian, subr.Source)
	if err != nil {
		return nil, fmt.Errorf("EGTS_SR_POS_DATA; Error writing SRC: %w", err)
	}

	if subr.AltitudeExists == "1" {
		_, err = buffer.Write(subr.AltitudeBytes)
		if err != nil {
			return nil, fmt.Errorf("EGTS_SR_POS_DATA; Error writing ALT: %w", err)
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
