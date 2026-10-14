package packet

import (
	"fmt"
	"github.com/LdDl/go-egts/egts/subrecord"
)

const (
	ServiceVersion1 uint8 = 1
	ServiceVersion2 uint8 = 2
)

func validateServiceVersion(version uint8) error {
	if version > ServiceVersion2 {
		return fmt.Errorf("Unsupported service layer version %d", version)
	}
	return nil
}

func (rd *RecordsData) validateVersion(version uint8) error {
	for _, r := range *rd {
		if r == nil {
			continue
		}
		switch r.SubrecordData.(type) {
		case *subrecord.SRTermIdentityV2, *subrecord.SRPosDataV2, *subrecord.SRVehicleDataV2, *subrecord.SRVehicleDataAdd:
			if version != ServiceVersion2 {
				return fmt.Errorf("V2 subrecord requires service version 2")
			}
		case *subrecord.SRTermIdentity, *subrecord.SRPosData:
			if version == ServiceVersion2 {
				return fmt.Errorf("V1 subrecord cannot be encoded as V2")
			}
		}
	}
	return nil
}

func (p *Packet) validateServiceVersion() error {
	if err := validateServiceVersion(p.ServiceVersion); err != nil {
		return err
	}
	var records *ServicesFrameData
	switch data := p.ServicesFrameData.(type) {
	case *ServicesFrameData:
		records = data
	case *PTResponse:
		if data != nil {
			records, _ = data.SDR.(*ServicesFrameData)
		}
	}
	if records != nil {
		for _, r := range *records {
			if r != nil && (r.ServiceVersion == ServiceVersion2) != (p.ServiceVersion == ServiceVersion2) {
				return fmt.Errorf("Packet and record service versions differ")
			}
		}
	}
	return nil
}
