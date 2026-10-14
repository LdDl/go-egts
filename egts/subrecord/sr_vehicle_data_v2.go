package subrecord

import (
	"encoding/binary"
	"fmt"
)

// SRVehicleDataV2 implements GOST 33465-2023 table 23.
type SRVehicleDataV2 struct {
	VINLow                string `json:"VINL"`
	VehicleType           uint32 `json:"VHT"`
	PropulsionStorageType uint32 `json:"VPST"`
	VINHigh               string `json:"VINH,omitempty"`
}

func (s *SRVehicleDataV2) Decode(b []byte) error {
	if len(b) < 25 || len(b) > 280 {
		return fmt.Errorf("EGTS_SR_VEHICLE_DATA V2; Invalid length")
	}
	*s = SRVehicleDataV2{VINLow: string(b[:17]), VehicleType: binary.LittleEndian.Uint32(b[17:21]), PropulsionStorageType: binary.LittleEndian.Uint32(b[21:25]), VINHigh: string(b[25:])}
	return nil
}
func (s *SRVehicleDataV2) Encode() ([]byte, error) {
	if s == nil || len(s.VINLow) != 17 || len(s.VINHigh) > 255 {
		return nil, fmt.Errorf("EGTS_SR_VEHICLE_DATA V2; Invalid VIN length")
	}
	b := append([]byte(nil), s.VINLow...)
	b = binary.LittleEndian.AppendUint32(b, s.VehicleType)
	b = binary.LittleEndian.AppendUint32(b, s.PropulsionStorageType)
	return append(b, s.VINHigh...), nil
}
func (s *SRVehicleDataV2) Len() uint16 { b, _ := s.Encode(); return uint16(len(b)) }

// SRVehicleDataAdd implements GOST 33465-2023 table 30.
// Fixed-size strings retain padding bytes; pointers distinguish absent fields.
type SRVehicleDataAdd struct {
	RegistrationMark        string  `json:"VSRM"`
	Model                   *string `json:"VM,omitempty"`
	Brand                   *string `json:"VB,omitempty"`
	OwnerTaxpayerID         *string `json:"VOTIN,omitempty"`
	OwnerRegistrationNumber *string `json:"VOPSRN,omitempty"`
	OwnerName               *string `json:"VON,omitempty"`
}

type vehicleTextField struct {
	value **string
	size  int
}

func (s *SRVehicleDataAdd) fields() []vehicleTextField {
	return []vehicleTextField{{&s.Model, 64}, {&s.Brand, 32}, {&s.OwnerTaxpayerID, 12}, {&s.OwnerRegistrationNumber, 15}, {&s.OwnerName, 64}}
}
func (s *SRVehicleDataAdd) Decode(b []byte) error {
	if len(b) < 33 || b[0]&0xe0 != 0 {
		return fmt.Errorf("EGTS_SR_VEHICLE_DATA_ADD; Invalid header")
	}
	decoded := SRVehicleDataAdd{RegistrationMark: string(b[1:33])}
	offset := 33
	for bit, field := range decoded.fields() {
		if b[0]&(1<<bit) == 0 {
			continue
		}
		if len(b)-offset < field.size {
			return fmt.Errorf("EGTS_SR_VEHICLE_DATA_ADD; Truncated field")
		}
		value := string(b[offset : offset+field.size])
		*field.value = &value
		offset += field.size
	}
	if offset != len(b) {
		return fmt.Errorf("EGTS_SR_VEHICLE_DATA_ADD; Unexpected trailing data")
	}
	*s = decoded
	return nil
}
func (s *SRVehicleDataAdd) Encode() ([]byte, error) {
	if s == nil || len(s.RegistrationMark) != 32 {
		return nil, fmt.Errorf("EGTS_SR_VEHICLE_DATA_ADD; Invalid registration mark length")
	}
	b := append([]byte{0}, s.RegistrationMark...)
	for bit, field := range s.fields() {
		if *field.value == nil {
			continue
		}
		if len(**field.value) != field.size {
			return nil, fmt.Errorf("EGTS_SR_VEHICLE_DATA_ADD; Invalid fixed string length")
		}
		b[0] |= 1 << bit
		b = append(b, **field.value...)
	}
	return b, nil
}
func (s *SRVehicleDataAdd) Len() uint16 { b, _ := s.Encode(); return uint16(len(b)) }
