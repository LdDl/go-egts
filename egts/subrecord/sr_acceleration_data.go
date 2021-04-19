package subrecord

import (
	"bytes"
	"encoding/binary"
	"fmt"
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

	rtm := make([]byte, 2)
	if _, err = buffer.Read(rtm); err != nil {
		return fmt.Errorf("EGTS_SR_ACCEL_DATA; Error reading RTM")
	}

	subr.RTM = binary.LittleEndian.Uint16(rtm)

	x := make([]byte, 2)
	if _, err = buffer.Read(x); err != nil {
		return fmt.Errorf("EGTS_SR_ACCEL_DATA; Error reading XAAV")
	}

	subr.XAAV = int16(binary.LittleEndian.Uint16(x))

	y := make([]byte, 2)
	if _, err = buffer.Read(y); err != nil {
		return fmt.Errorf("EGTS_SR_ACCEL_DATA; Error reading YAAV")
	}

	subr.YAAV = int16(binary.LittleEndian.Uint16(y))

	z := make([]byte, 2)
	if _, err = buffer.Read(z); err != nil {
		return fmt.Errorf("EGTS_SR_ACCEL_DATA; Error reading ZAAV")
	}

	subr.ZAAV = int16(binary.LittleEndian.Uint16(z))

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

	if subr.StructuresAmount, err = buffer.ReadByte(); err != nil {
		return fmt.Errorf("EGTS_SR_ACCEL_DATA; Error reading SA")
	}

	timestamp, _ := time.Parse(time.RFC3339, "2010-01-01T00:00:00+00:00")
	nt := make([]byte, 4)

	if _, err = buffer.Read(nt); err != nil {
		return fmt.Errorf("EGTS_SR_ACCEL_DATA; Error reading NTM")
	}

	subr.AbsoluteTimeUint = binary.LittleEndian.Uint32(nt)
	subr.AbsoluteTime = timestamp.Add(time.Duration(int(subr.AbsoluteTimeUint)) * time.Second)

	subr.AccelerationData = SRAccelerationsData{}

	ads := &SRAccelerationData{}

	for buffer.Len() > 0 {
		bb := make([]byte, 8)
		if _, err = buffer.Read(bb); err != nil {
			return err
		}

		err := ads.Decode(bb)
		if err != nil {
			return fmt.Errorf("EGTS_SR_ACCEL_DATA;" + err.Error())
		}

		subr.AccelerationData = append(subr.AccelerationData, ads)
	}

	return nil
}

// Encode Parse EGTS_SR_ACCEL_DATA to array of bytes
func (subr *SRAccelerationHeader) Encode() (b []byte, err error) {
	buffer := new(bytes.Buffer)

	if err = buffer.WriteByte(subr.StructuresAmount); err != nil {
		return nil, fmt.Errorf("EGTS_SR_ACCEL_DATA; Error writing SA")
	}

	timestamp, _ := time.Parse(time.RFC3339, "2010-01-01T00:00:00+00:00")
	if err = binary.Write(buffer, binary.LittleEndian, uint32(subr.AbsoluteTime.Sub(timestamp).Seconds())); err != nil {
		return nil, fmt.Errorf("EGTS_SR_ACCEL_DATA; Error writing NTM")
	}

	for _, sr := range subr.AccelerationData {
		rd, err := sr.Encode()
		if err != nil {
			return nil, fmt.Errorf("EGTS_SR_ACCEL_DATA;" + err.Error())
		}

		buffer.Write(rd)
	}

	return buffer.Bytes(), nil
}

// Len Returns length of bytes slice
func (subr *SRAccelerationHeader) Len() (l uint16) {
	encoded, _ := subr.Encode()
	l = uint16(len(encoded))

	return l
}
