package subrecord

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"io"
	"time"
)

// SRAcceleration EGTS_SR_ACCEL_DATA
/*
	Применяется АСН для передачи на аппаратно-
	программный комплекс данных о
	состоянии
*/

type SRAccelerationData struct {
	RTM  uint16 `json:"RTM"`  // RTM (Relative Time)
	XAAV int16  `json:"XAAV"` // XAAV (X Axis Acceleration Value)
	YAAV int16  `json:"YAAV"` // YAAV (Y Axis Acceleration Value)
	ZAAV int16  `json:"ZAAV"` // ZAAV (Z Axis Acceleration Value)
}

// RecordsData Slice of RecordData
type SRAccelerationsData []*SRAccelerationData

type SRAccelerationHeader struct {
	StructuresAmount uint8               `json:"SA"`          // SA - число передаваемых структур данных
	AbsoluteTimeUint uint32              `json:"ATM"`         // ATM - время проведения измерений первой передаваемой структуры
	AbsoluteTime     time.Time           `json:"ATM_RFC3339"` // ATM - время
	AccelerationData SRAccelerationsData `json:"ADS"`         // ADS - структуры данных показаний акселерометра
}

// Decode Parse array of bytes to EGTS_SR_ACCEL_DATA
func (subr *SRAccelerationData) Decode(b []byte) (err error) {
	buffer := bytes.NewReader(b)
	decoded := SRAccelerationData{}

	rtm := make([]byte, 2)
	_, err = io.ReadFull(buffer, rtm)
	if err != nil {
		return fmt.Errorf("EGTS_SR_ACCEL_DATA; Error reading RTM: %w", err)
	}

	decoded.RTM = binary.LittleEndian.Uint16(rtm)

	x := make([]byte, 2)
	_, err = io.ReadFull(buffer, x)
	if err != nil {
		return fmt.Errorf("EGTS_SR_ACCEL_DATA; Error reading XAAV: %w", err)
	}

	decoded.XAAV = int16(binary.LittleEndian.Uint16(x))

	y := make([]byte, 2)
	_, err = io.ReadFull(buffer, y)
	if err != nil {
		return fmt.Errorf("EGTS_SR_ACCEL_DATA; Error reading YAAV: %w", err)
	}

	decoded.YAAV = int16(binary.LittleEndian.Uint16(y))

	z := make([]byte, 2)
	_, err = io.ReadFull(buffer, z)
	if err != nil {
		return fmt.Errorf("EGTS_SR_ACCEL_DATA; Error reading ZAAV: %w", err)
	}

	decoded.ZAAV = int16(binary.LittleEndian.Uint16(z))

	if buffer.Len() != 0 {
		return fmt.Errorf("EGTS_SR_ACCEL_DATA; Unexpected trailing data in ADS")
	}
	*subr = decoded
	return nil
}

// Encode Parse EGTS_SR_ACCEL_DATA to array of bytes
func (subr *SRAccelerationData) Encode() (b []byte, err error) {
	buffer := new(bytes.Buffer)

	if err = binary.Write(buffer, binary.LittleEndian, subr.RTM); err != nil {
		return nil, fmt.Errorf("EGTS_SR_ACCEL_DATA; Error writing RTM")
	}

	if err = binary.Write(buffer, binary.LittleEndian, subr.XAAV); err != nil {
		return nil, fmt.Errorf("EGTS_SR_ACCEL_DATA; Error writing XAAV")
	}

	if err = binary.Write(buffer, binary.LittleEndian, subr.YAAV); err != nil {
		return nil, fmt.Errorf("EGTS_SR_ACCEL_DATA; Error writing YAAV")
	}

	if err = binary.Write(buffer, binary.LittleEndian, subr.ZAAV); err != nil {
		return nil, fmt.Errorf("EGTS_SR_ACCEL_DATA; Error writing ZAAV")
	}

	return buffer.Bytes(), nil
}

// Len Returns length of bytes slice
func (subr *SRAccelerationData) Len() (l uint16) {
	encoded, _ := subr.Encode()
	l = uint16(len(encoded))

	return l
}

// Decode Parse array of bytes to EGTS_SR_ACCEL_DATA
func (subr *SRAccelerationHeader) Decode(b []byte) (err error) {
	buffer := bytes.NewReader(b)
	decoded := SRAccelerationHeader{}

	decoded.StructuresAmount, err = buffer.ReadByte()
	if err != nil {
		return fmt.Errorf("EGTS_SR_ACCEL_DATA; Error reading SA: %w", err)
	}
	if decoded.StructuresAmount == 0 {
		return fmt.Errorf("EGTS_SR_ACCEL_DATA; SA must be greater than zero")
	}

	timestamp := time.Date(2010, time.January, 1, 0, 0, 0, 0, time.UTC)
	nt := make([]byte, 4)

	_, err = io.ReadFull(buffer, nt)
	if err != nil {
		return fmt.Errorf("EGTS_SR_ACCEL_DATA; Error reading ATM: %w", err)
	}

	decoded.AbsoluteTimeUint = binary.LittleEndian.Uint32(nt)
	decoded.AbsoluteTime = timestamp.Add(time.Duration(decoded.AbsoluteTimeUint) * time.Second)

	if buffer.Len() != int(decoded.StructuresAmount)*8 {
		return fmt.Errorf("EGTS_SR_ACCEL_DATA; Data length does not match SA")
	}

	for i := 0; i < int(decoded.StructuresAmount); i++ {
		ads := &SRAccelerationData{}
		bb := make([]byte, 8)
		_, err = io.ReadFull(buffer, bb)
		if err != nil {
			return fmt.Errorf("EGTS_SR_ACCEL_DATA; Error reading ADS%d: %w", i+1, err)
		}

		err = ads.Decode(bb)
		if err != nil {
			return fmt.Errorf("EGTS_SR_ACCEL_DATA; ADS%d: %w", i+1, err)
		}

		decoded.AccelerationData = append(decoded.AccelerationData, ads)
	}

	*subr = decoded
	return nil
}

// Encode Parse EGTS_SR_ACCEL_DATA to array of bytes
func (subr *SRAccelerationHeader) Encode() (b []byte, err error) {
	buffer := new(bytes.Buffer)

	if subr.StructuresAmount == 0 {
		return nil, fmt.Errorf("EGTS_SR_ACCEL_DATA; SA must be greater than zero")
	}
	if int(subr.StructuresAmount) != len(subr.AccelerationData) {
		return nil, fmt.Errorf("EGTS_SR_ACCEL_DATA; Number of ADS does not match SA")
	}
	err = buffer.WriteByte(subr.StructuresAmount)
	if err != nil {
		return nil, fmt.Errorf("EGTS_SR_ACCEL_DATA; Error writing SA: %w", err)
	}

	timestamp := time.Date(2010, time.January, 1, 0, 0, 0, 0, time.UTC)
	err = binary.Write(buffer, binary.LittleEndian, uint32(subr.AbsoluteTime.Sub(timestamp).Seconds()))
	if err != nil {
		return nil, fmt.Errorf("EGTS_SR_ACCEL_DATA; Error writing ATM: %w", err)
	}

	for i, sr := range subr.AccelerationData {
		if sr == nil {
			return nil, fmt.Errorf("EGTS_SR_ACCEL_DATA; ADS%d is nil", i+1)
		}
		rd, err := sr.Encode()
		if err != nil {
			return nil, fmt.Errorf("EGTS_SR_ACCEL_DATA; ADS%d: %w", i+1, err)
		}

		_, err = buffer.Write(rd)
		if err != nil {
			return nil, fmt.Errorf("EGTS_SR_ACCEL_DATA; Error writing ADS%d: %w", i+1, err)
		}
	}

	return buffer.Bytes(), nil
}

// Len Returns length of bytes slice
func (subr *SRAccelerationHeader) Len() (l uint16) {
	encoded, _ := subr.Encode()
	l = uint16(len(encoded))

	return l
}
