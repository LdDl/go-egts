package subrecord

import (
	"encoding/binary"

	"github.com/LdDl/go-egts/egts/utils"
)

// SRTermIdentity EGTS_SR_TERM_IDENTITY
type SRTermIdentity struct {
	TerminalIdentifier uint16 `json:"TID"` // TID (Terminal Identifier)

	/* Flags: MNE, BSE, NIDE, SSRA, LNGCE, IMSIE, IMEIE, HDIDE */
	MNE   bool `json:"MNE"`   // MNE
	BSE   bool `json:"BSE"`   // BSE
	NIDE  bool `json:"NIDE"`  // NIDE
	SSRA  bool `json:"SSRA"`  // SSRA
	LNGCE bool `json:"LNGCE"` // LNGCE
	IMSIE bool `json:"IMSIE"` // IMSIE
	IMEIE bool `json:"IMEIE"` // IMEIE
	HDIDE bool `json:"HDIDE"` // HDIDE
	/*                                                        */
	HomeDispatcherIdentifier                            uint16 `json:"HDID"`   // HDID (Home Dispatcher Identifier)
	InternationalMobileEquipmentIdentity                string `json:"IMEI"`   // IMEI (International Mobile Equipment Identity)
	InternationalMobileSubscriberIdentity               string `json:"IMSI"`   // IMSI (International Mobile Subscriber Identity)
	LanguageCode                                        string `json:"LNGC"`   // LNGC (Language Code)
	NetworkIdentifier                                   uint16 `json:"NID"`    // NID (Network Identifier)
	BufferSize                                          uint16 `json:"BS"`     // BS (Buffer Size)
	MobileStationIntegratedServicesDigitalNetworkNumber string `json:"MSISDN"` // MSISDN (Mobile Station Integrated Services Digital Network Number)
}

func (subr *SRTermIdentity) Decode(b []byte) {
	// TID (Terminal Identifier)
	i := 0
	subr.TerminalIdentifier = binary.LittleEndian.Uint16(b[i : i+4])

	// Flags: MNE, BSE, NIDE, SSRA, LNGCE, IMSIE, IMEIE, HDIDE
	i += 4
	flagBytes := uint16(b[i])
	// HDIDE
	subr.HDIDE = utils.BitField(flagBytes, 0).(bool)
	// IMEIE
	subr.IMEIE = utils.BitField(flagBytes, 1).(bool)
	// IMSIE
	subr.IMSIE = utils.BitField(flagBytes, 2).(bool)
	// LNGCE
	subr.LNGCE = utils.BitField(flagBytes, 3).(bool)
	// SSRA
	subr.SSRA = utils.BitField(flagBytes, 4).(bool)
	// NIDE
	subr.NIDE = utils.BitField(flagBytes, 5).(bool)
	// BSE
	subr.BSE = utils.BitField(flagBytes, 6).(bool)
	// MNE
	subr.MNE = utils.BitField(flagBytes, 7).(bool)

	// HDID (Home Dispatcher Identifier)
	i++
	if subr.HDIDE {
		subr.HomeDispatcherIdentifier = binary.LittleEndian.Uint16(b[i : i+2])
		i += 2
	}

	// IMEI (International Mobile Equipment Identity)
	if subr.IMEIE {
		subr.InternationalMobileEquipmentIdentity = string(b[i : i+15])
		i += 15
	}

	// IMSI (International Mobile Subscriber Identity)
	if subr.IMSIE {
		subr.InternationalMobileSubscriberIdentity = string(b[i : i+16])
		i += 16
	}

	// LNGC (Language Code)
	if subr.LNGCE {
		subr.LanguageCode = string(b[i : i+3])
		i += 3
	}

	// NID (Network Identifier)
	if subr.NIDE {
		subr.NetworkIdentifier = binary.LittleEndian.Uint16(b[i : i+3])
		i += 3
	}

	// BS (Buffer Size)
	if subr.BSE {
		subr.BufferSize = binary.LittleEndian.Uint16(b[i : i+2])
		i += 2
	}

	// MSISDN (Mobile Station Integrated Services Digital Network Number)
	if subr.MNE {
		subr.MobileStationIntegratedServicesDigitalNetworkNumber = string(b[i : i+15])
		i += 15
	}
}

// BytesToSRTermIdentity Parse array of bytes to EGTS_SR_TERM_IDENTITY
func BytesToSRTermIdentity(b []byte) (subr SRTermIdentity) {
	// TID (Terminal Identifier)
	i := 0
	subr.TerminalIdentifier = binary.LittleEndian.Uint16(b[i : i+4])

	// Flags: MNE, BSE, NIDE, SSRA, LNGCE, IMSIE, IMEIE, HDIDE
	i += 4
	flagBytes := uint16(b[i])
	// HDIDE
	subr.HDIDE = utils.BitField(flagBytes, 0).(bool)
	// IMEIE
	subr.IMEIE = utils.BitField(flagBytes, 1).(bool)
	// IMSIE
	subr.IMSIE = utils.BitField(flagBytes, 2).(bool)
	// LNGCE
	subr.LNGCE = utils.BitField(flagBytes, 3).(bool)
	// SSRA
	subr.SSRA = utils.BitField(flagBytes, 4).(bool)
	// NIDE
	subr.NIDE = utils.BitField(flagBytes, 5).(bool)
	// BSE
	subr.BSE = utils.BitField(flagBytes, 6).(bool)
	// MNE
	subr.MNE = utils.BitField(flagBytes, 7).(bool)

	// HDID (Home Dispatcher Identifier)
	i++
	if subr.HDIDE {
		subr.HomeDispatcherIdentifier = binary.LittleEndian.Uint16(b[i : i+2])
		i += 2
	}

	// IMEI (International Mobile Equipment Identity)
	if subr.IMEIE {
		subr.InternationalMobileEquipmentIdentity = string(b[i : i+15])
		i += 15
	}

	// IMSI (International Mobile Subscriber Identity)
	if subr.IMSIE {
		subr.InternationalMobileSubscriberIdentity = string(b[i : i+16])
		i += 16
	}

	// LNGC (Language Code)
	if subr.LNGCE {
		subr.LanguageCode = string(b[i : i+3])
		i += 3
	}

	// NID (Network Identifier)
	if subr.NIDE {
		subr.NetworkIdentifier = binary.LittleEndian.Uint16(b[i : i+3])
		i += 3
	}

	// BS (Buffer Size)
	if subr.BSE {
		subr.BufferSize = binary.LittleEndian.Uint16(b[i : i+2])
		i += 2
	}

	// MSISDN (Mobile Station Integrated Services Digital Network Number)
	if subr.MNE {
		subr.MobileStationIntegratedServicesDigitalNetworkNumber = string(b[i : i+15])
		i += 15
	}
	return subr
}
