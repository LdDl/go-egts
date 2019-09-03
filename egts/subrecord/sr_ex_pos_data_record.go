package subrecord

import (
	"bytes"
	"encoding/binary"
	"fmt"
)

// SRExPosDataRecord EGTS_SR_EXT_POS_DATA
/*
	Используется абонентским терминалом
	при передаче дополнительных данных
	определения местоположения
*/
type SRExPosDataRecord struct {
	VerticalDiluptionOfPrecisionExists   string
	HorizontalDiluptionOfPrecisionExists string
	PositionDiluptionOfPrecisionExists   string
	SatellitesExists                     string
	NavigationSystemExists               string

	VerticalDiluptionOfPrecision   uint16 `json:"VDOP"` /* Vertical Dilution of Precision */
	HorizontalDiluptionOfPrecision uint16 `json:"HDOP"` /* Horizontal Dilution of Precision */
	PositionDiluptionOfPrecision   uint16 `json:"PDOP"` /* Position Dilution of Precision */
	Satellites                     uint8  `json:"SAT"`  /* Satellites */
	NavigationSystem               uint16 `json:"NS"`   /* Navigation System */
}

// Decode Parse array of bytes to EGTS_SR_EXT_POS_DATA
func (subr *SRExPosDataRecord) Decode(b []byte) (err error) {
	buffer := bytes.NewReader(b)

	// Flags
	flagByte := byte(0)
	if flagByte, err = buffer.ReadByte(); err != nil {
		return fmt.Errorf("Error reading flags")
	}
	flagByteAsBits := fmt.Sprintf("%08b", flagByte)
	subr.VerticalDiluptionOfPrecisionExists = flagByteAsBits[7:]
	subr.HorizontalDiluptionOfPrecisionExists = flagByteAsBits[6:7]
	subr.PositionDiluptionOfPrecisionExists = flagByteAsBits[5:6]
	subr.SatellitesExists = flagByteAsBits[4:5]
	subr.NavigationSystemExists = flagByteAsBits[3:4]

	if subr.VerticalDiluptionOfPrecisionExists == "1" {
		vdop := make([]byte, 2)
		if _, err = buffer.Read(vdop); err != nil {
			return fmt.Errorf("Error reading VDOP")
		}
		subr.VerticalDiluptionOfPrecision = binary.LittleEndian.Uint16(vdop)
	}
	if subr.HorizontalDiluptionOfPrecisionExists == "1" {
		hdop := make([]byte, 2)
		if _, err = buffer.Read(hdop); err != nil {
			return fmt.Errorf("Error reading HDOP")
		}
		subr.HorizontalDiluptionOfPrecision = binary.LittleEndian.Uint16(hdop)
	}
	if subr.PositionDiluptionOfPrecisionExists == "1" {
		pdop := make([]byte, 2)
		if _, err = buffer.Read(pdop); err != nil {
			return fmt.Errorf("Error reading PDOP")
		}
		subr.PositionDiluptionOfPrecision = binary.LittleEndian.Uint16(pdop)
	}
	if subr.SatellitesExists == "1" {
		if subr.Satellites, err = buffer.ReadByte(); err != nil {
			return fmt.Errorf("Error reading SAT")
		}
	}
	if subr.NavigationSystemExists == "1" {
		ns := make([]byte, 2)
		if _, err = buffer.Read(ns); err != nil {
			return fmt.Errorf("Error reading NS")
		}
		subr.NavigationSystem = binary.LittleEndian.Uint16(ns)
	}
	/*
		NS:
		0	- система не определена;
		1 - ГЛОНАСС;
		2 - GPS;
		4 - Galileo;
		8 - Compass;
		16 - Beidou;
		32 - DORIS;
		64 - IRNSS;
		128 - QZSS.
		Остальные значения зарезервированы.
	*/
	return nil
}

// Encode Parse EGTS_SR_EXT_POS_DATA to array of bytes
func (subr *SRExPosDataRecord) Encode() (b []byte) {
	return b
}

// Len Returns length of bytes slice
func (subr *SRExPosDataRecord) Len() (l uint16) {
	l = uint16(len(subr.Encode()))
	return l
}
