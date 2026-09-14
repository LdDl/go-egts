package subrecord

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"io"
	"strconv"
	"strings"
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
	decoded := SRExPosDataRecord{}

	// Flags
	flagByte := byte(0)
	flagByte, err = buffer.ReadByte()
	if err != nil {
		return fmt.Errorf("EGTS_SR_EXT_POS_DATA; Error reading flags: %w", err)
	}
	flagByteAsBits := fmt.Sprintf("%08b", flagByte)
	decoded.NavigationSystemExists = flagByteAsBits[3:4]
	decoded.SatellitesExists = flagByteAsBits[4:5]
	decoded.PositionDiluptionOfPrecisionExists = flagByteAsBits[5:6]
	decoded.HorizontalDiluptionOfPrecisionExists = flagByteAsBits[6:7]
	decoded.VerticalDiluptionOfPrecisionExists = flagByteAsBits[7:]

	if decoded.VerticalDiluptionOfPrecisionExists == "1" {
		vdop := make([]byte, 2)
		_, err = io.ReadFull(buffer, vdop)
		if err != nil {
			return fmt.Errorf("EGTS_SR_EXT_POS_DATA; Error reading VDOP: %w", err)
		}
		decoded.VerticalDiluptionOfPrecision = binary.LittleEndian.Uint16(vdop)
	}
	if decoded.HorizontalDiluptionOfPrecisionExists == "1" {
		hdop := make([]byte, 2)
		_, err = io.ReadFull(buffer, hdop)
		if err != nil {
			return fmt.Errorf("EGTS_SR_EXT_POS_DATA; Error reading HDOP: %w", err)
		}
		decoded.HorizontalDiluptionOfPrecision = binary.LittleEndian.Uint16(hdop)
	}
	if decoded.PositionDiluptionOfPrecisionExists == "1" {
		pdop := make([]byte, 2)
		_, err = io.ReadFull(buffer, pdop)
		if err != nil {
			return fmt.Errorf("EGTS_SR_EXT_POS_DATA; Error reading PDOP: %w", err)
		}
		decoded.PositionDiluptionOfPrecision = binary.LittleEndian.Uint16(pdop)
	}
	if decoded.SatellitesExists == "1" {
		decoded.Satellites, err = buffer.ReadByte()
		if err != nil {
			return fmt.Errorf("EGTS_SR_EXT_POS_DATA; Error reading SAT: %w", err)
		}
	}
	if decoded.NavigationSystemExists == "1" {
		ns := make([]byte, 2)
		_, err = io.ReadFull(buffer, ns)
		if err != nil {
			return fmt.Errorf("EGTS_SR_EXT_POS_DATA; Error reading NS: %w", err)
		}
		decoded.NavigationSystem = binary.LittleEndian.Uint16(ns)
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
	if buffer.Len() != 0 {
		return fmt.Errorf("EGTS_SR_EXT_POS_DATA; Unexpected trailing data")
	}
	*subr = decoded
	return nil
}

// Encode Parse EGTS_SR_EXT_POS_DATA to array of bytes
func (subr *SRExPosDataRecord) Encode() (b []byte, err error) {

	buffer := new(bytes.Buffer)

	flags := uint64(0)
	flags, err = strconv.ParseUint(strings.Repeat("0", 3)+subr.NavigationSystemExists+subr.SatellitesExists+subr.PositionDiluptionOfPrecisionExists+subr.HorizontalDiluptionOfPrecisionExists+subr.VerticalDiluptionOfPrecisionExists, 2, 8)
	// flags, err = strconv.ParseUint(subr.VerticalDiluptionOfPrecisionExists+subr.HorizontalDiluptionOfPrecisionExists+subr.PositionDiluptionOfPrecisionExists+subr.SatellitesExists+subr.NavigationSystemExists+strings.Repeat("0", 3), 2, 8)

	if err != nil {
		return nil, fmt.Errorf("EGTS_SR_EXT_POS_DATA; Error writing flags")
	}
	if err = buffer.WriteByte(uint8(flags)); err != nil {
		return nil, fmt.Errorf("EGTS_SR_EXT_POS_DATA; Error writing flags byte")
	}

	if subr.VerticalDiluptionOfPrecisionExists == "1" {
		if err = binary.Write(buffer, binary.LittleEndian, subr.VerticalDiluptionOfPrecision); err != nil {
			return nil, fmt.Errorf("EGTS_SR_EXT_POS_DATA; Error writing VDOP")
		}
	}

	if subr.HorizontalDiluptionOfPrecisionExists == "1" {
		if err = binary.Write(buffer, binary.LittleEndian, subr.HorizontalDiluptionOfPrecision); err != nil {
			return nil, fmt.Errorf("EGTS_SR_EXT_POS_DATA; Error writing HDOP")
		}
	}

	if subr.PositionDiluptionOfPrecisionExists == "1" {
		if err = binary.Write(buffer, binary.LittleEndian, subr.PositionDiluptionOfPrecision); err != nil {
			return nil, fmt.Errorf("EGTS_SR_EXT_POS_DATA; Error writing PDOP")
		}
	}

	if subr.SatellitesExists == "1" {
		if err = buffer.WriteByte(subr.Satellites); err != nil {
			return nil, fmt.Errorf("EGTS_SR_EXT_POS_DATA; Error writing SAT")
		}
	}

	if subr.NavigationSystemExists == "1" {
		if err = binary.Write(buffer, binary.LittleEndian, subr.NavigationSystem); err != nil {
			return nil, fmt.Errorf("EGTS_SR_EXT_POS_DATA; Error writing NS")
		}
	}

	return buffer.Bytes(), nil
}

// Len Returns length of bytes slice
func (subr *SRExPosDataRecord) Len() (l uint16) {
	encoded, _ := subr.Encode()
	l = uint16(len(encoded))
	return l
}

func reverse(s string) string {
	runes := []rune(s)
	for i, j := 0, len(runes)-1; i < j; i, j = i+1, j-1 {
		runes[i], runes[j] = runes[j], runes[i]
	}
	return string(runes)
}
