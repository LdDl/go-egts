package subrecord

import (
	"encoding/binary"
	"fmt"
)

// SRPosDataV2 implements GOST 33465-2023 table И.2.
// SpeedTenths is the wire speed in 0.1 km/h units, including its fractional part.
type SRPosDataV2 struct {
	SRPosData
	NetworkIdentifier [3]byte `json:"NID"`
	LocalAreaCode     uint32  `json:"LAC"`
	CellIdentifier    int16   `json:"CID"`
	SignalStrength    uint8   `json:"SS"`
	SpeedTenths       uint16  `json:"SPD_tenths"`
}

func (s *SRPosDataV2) Decode(b []byte) error {
	if len(b) < 31 {
		return fmt.Errorf("EGTS_SR_POS_DATA V2; Truncated mandatory fields")
	}
	size := 31
	if b[12]&0x80 != 0 {
		size += 3
	}
	if len(b) != size && len(b) != size+2 {
		return fmt.Errorf("EGTS_SR_POS_DATA V2; Invalid length")
	}
	legacy := append([]byte(nil), b[:21]...)
	legacy = append(legacy, b[31:size]...)
	var decoded SRPosDataV2
	if err := decoded.SRPosData.Decode(legacy); err != nil {
		return err
	}
	copy(decoded.NetworkIdentifier[:], b[21:24])
	decoded.LocalAreaCode = binary.LittleEndian.Uint32(b[24:28])
	decoded.CellIdentifier = int16(binary.LittleEndian.Uint16(b[28:30]))
	decoded.SignalStrength = b[30]
	decoded.SpeedTenths = binary.LittleEndian.Uint16(b[13:15]) & 0x3fff
	if len(b) == size+2 {
		decoded.SourceDataExists = 1
		decoded.SourceData = int16(binary.LittleEndian.Uint16(b[size:]))
	}
	*s = decoded
	return nil
}

func (s *SRPosDataV2) Encode() ([]byte, error) {
	if s == nil {
		return nil, fmt.Errorf("EGTS_SR_POS_DATA V2; Nil subrecord")
	}
	if s.SpeedTenths > 0x3fff || s.SourceDataExists > 1 {
		return nil, fmt.Errorf("EGTS_SR_POS_DATA V2; Field out of range")
	}
	if len(s.OdometerBytes) != 3 || (s.AltitudeExists == "1" && len(s.AltitudeBytes) != 3) {
		return nil, fmt.Errorf("EGTS_SR_POS_DATA V2; Invalid ODM or ALT length")
	}
	for _, flag := range []string{s.Valid, s.CoordinateSystem, s.Fix, s.BlackBox, s.Move, s.LAHS, s.LOHS, s.AltitudeExists} {
		if flag != "0" && flag != "1" {
			return nil, fmt.Errorf("EGTS_SR_POS_DATA V2; Invalid flag")
		}
	}
	if s.AltsFlag > 1 || s.DirhFlag > 1 {
		return nil, fmt.Errorf("EGTS_SR_POS_DATA V2; Invalid speed flags")
	}
	legacy, err := s.SRPosData.Encode()
	if err != nil {
		return nil, err
	}
	binary.LittleEndian.PutUint16(legacy[13:15], s.SpeedTenths|uint16(s.AltsFlag)<<14|uint16(s.DirhFlag)<<15)
	b := append([]byte(nil), legacy[:21]...)
	b = append(b, s.NetworkIdentifier[:]...)
	b = binary.LittleEndian.AppendUint32(b, s.LocalAreaCode)
	b = binary.LittleEndian.AppendUint16(b, uint16(s.CellIdentifier))
	b = append(b, s.SignalStrength)
	b = append(b, legacy[21:]...)
	if s.SourceDataExists == 1 {
		b = binary.LittleEndian.AppendUint16(b, uint16(s.SourceData))
	}
	return b, nil
}

func (s *SRPosDataV2) Len() uint16 { b, _ := s.Encode(); return uint16(len(b)) }
