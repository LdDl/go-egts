package subrecord

import (
	"encoding/binary"
	"fmt"
)

// SRTermIdentityV2 implements GOST 33465-2023 table 20.
// The embedded V1 identifier must be zero; TerminalIdentifier holds the full TID.
type SRTermIdentityV2 struct {
	SRTermIdentity
	TerminalIdentifier                 uint64 `json:"TID"`
	ServiceSupportLevelProtocolVersion string `json:"SSLPV,omitempty"`
}

func (s *SRTermIdentityV2) Decode(b []byte) error {
	if len(b) < 9 {
		return fmt.Errorf("EGTS_SR_TERM_IDENTITY V2; Truncated TID or flags")
	}
	size := 9
	flags := b[8]
	sizes := [...]int{2, 15, 16, 3, 0, 3, 2, 15}
	for bit, n := range sizes {
		if flags&(1<<bit) != 0 {
			size += n
		}
	}
	if len(b) != size && len(b) != size+2 {
		return fmt.Errorf("EGTS_SR_TERM_IDENTITY V2; Invalid length")
	}
	decoded := SRTermIdentityV2{TerminalIdentifier: binary.LittleEndian.Uint64(b[:8])}
	legacy := make([]byte, 4, size-4)
	legacy = append(legacy, b[8:size]...)
	if err := decoded.SRTermIdentity.Decode(legacy); err != nil {
		return err
	}
	if len(b) == size+2 {
		decoded.ServiceSupportLevelProtocolVersion = string(b[size:])
		if decoded.ServiceSupportLevelProtocolVersion != "02" {
			return fmt.Errorf("EGTS_SR_TERM_IDENTITY V2; Unsupported SSLPV")
		}
	}
	*s = decoded
	return nil
}

func (s *SRTermIdentityV2) Encode() ([]byte, error) {
	if s == nil {
		return nil, fmt.Errorf("EGTS_SR_TERM_IDENTITY V2; Nil subrecord")
	}
	if s.SRTermIdentity.TerminalIdentifier != 0 {
		return nil, fmt.Errorf("EGTS_SR_TERM_IDENTITY V2; Embedded V1 TID must be zero")
	}
	if s.ServiceSupportLevelProtocolVersion != "" && s.ServiceSupportLevelProtocolVersion != "02" {
		return nil, fmt.Errorf("EGTS_SR_TERM_IDENTITY V2; Unsupported SSLPV")
	}
	flags := []string{s.HDIDE, s.IMEIE, s.IMSIE, s.LNGCE, s.SSRA, s.NIDE, s.BSE, s.MNE}
	for _, flag := range flags {
		if flag != "0" && flag != "1" {
			return nil, fmt.Errorf("EGTS_SR_TERM_IDENTITY V2; Invalid flag")
		}
	}
	fields := []identityV2FieldLength{
		{s.IMEIE, len(s.InternationalMobileEquipmentIdentity), 15},
		{s.IMSIE, len(s.InternationalMobileSubscriberIdentity), 16},
		{s.LNGCE, len(s.LanguageCode), 3},
		{s.NIDE, len(s.NetworkIdentifier), 3},
		{s.MNE, len(s.MobileStationIntegratedServicesDigitalNetworkNumber), 15},
	}
	for _, field := range fields {
		if field.enabled == "1" && field.actual != field.expected {
			return nil, fmt.Errorf("EGTS_SR_TERM_IDENTITY V2; Invalid fixed field length")
		}
	}
	legacy, err := s.SRTermIdentity.Encode()
	if err != nil {
		return nil, err
	}
	b := make([]byte, 8)
	binary.LittleEndian.PutUint64(b, s.TerminalIdentifier)
	b = append(b, legacy[4:]...)
	b = append(b, s.ServiceSupportLevelProtocolVersion...)
	var check SRTermIdentityV2
	if err := check.Decode(b); err != nil {
		return nil, err
	}
	return b, nil
}

func (s *SRTermIdentityV2) Len() uint16 { b, _ := s.Encode(); return uint16(len(b)) }

type identityV2FieldLength struct {
	enabled  string
	actual   int
	expected int
}
