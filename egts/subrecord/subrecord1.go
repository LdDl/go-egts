package subrecord

import (
	"encoding/binary"
	"fmt"

	"github.com/LdDl/go-egts/egts/utils"
)

// EgtsSrTermIdentity EGTS_SR_TERM_IDENTITY
type EgtsSrTermIdentity struct {
	TerminalIdentifier uint16 // TID (Terminal Identifier)
	/* Flags: MNE, BSE, NIDE, SSRA, LNGCE, IMSIE, IMEIE, HDIDE */
	MNE   bool // MNE
	BSE   bool // BSE
	NIDE  bool // NIDE
	SSRA  bool // SSRA
	LNGCE bool // LNGCE
	IMSIE bool // IMSIE
	IMEIE bool // IMEIE
	HDIDE bool // HDIDE
	/*                                                        */
	HomeDispatcherIdentifier                            uint16 // HDID (Home Dispatcher Identifier)
	InternationalMobileEquipmentIdentity                string // IMEI (International Mobile Equipment Identity)
	InternationalMobileSubscriberIdentity               string // IMSI (International Mobile Subscriber Identity)
	LanguageCode                                        string // LNGC (Language Code)
	NetworkIdentifier                                   uint16 // NID (Network Identifier)
	BufferSize                                          uint16 // BS (Buffer Size)
	MobileStationIntegratedServicesDigitalNetworkNumber string // MSISDN (Mobile Station Integrated Services Digital Network Number)
}

func (subrecord EgtsSrTermIdentity) String() string {
	return fmt.Sprintf("\n\t\t\tEGTS_SR_TERM_IDENTITY:\n\t\t\tTID (Terminal Identifier): %v\n\t\t\tFlags:\n\t\t\t\tMNE: %v\n\t\t\t\tBSE: %v\n\t\t\t\tNIDE: %v\n\t\t\t\tSSRA: %v\n\t\t\t\tLNGCE: %v\n\t\t\t\tIMSIE: %v\n\t\t\t\tIMEIE: %v\n\t\t\t\tHDIDE: %v\n\t\t\tHDID (Home Dispatcher Identifier): %v\n\t\t\tIMEI (International Mobile Equipment Identity): %v\n\t\t\tIMSI (International Mobile Subscriber Identity): %v\n\t\t\tLNGC (Language Code): %v\n\t\t\tNID (Network Identifier): %v\n\t\t\tBS (Buffer Size): %v\n\t\t\tMSISDN (Mobile Station Integrated Services Digital Network Number): %v",
		subrecord.TerminalIdentifier,
		subrecord.MNE,
		subrecord.BSE,
		subrecord.NIDE,
		subrecord.SSRA,
		subrecord.LNGCE,
		subrecord.IMSIE,
		subrecord.IMEIE,
		subrecord.HDIDE,
		subrecord.TerminalIdentifier,
		subrecord.HomeDispatcherIdentifier,
		subrecord.InternationalMobileEquipmentIdentity,
		subrecord.LanguageCode,
		subrecord.NetworkIdentifier,
		subrecord.BufferSize,
		subrecord.MobileStationIntegratedServicesDigitalNetworkNumber,
	)
}

//ParseEgtsSrTermIdentity - EGTS_SR_TERM_IDENTITY
//Подзапись EGTS_SR_TERM_IDENTITY сервиса EGTS_AUTH_SERVICE
//Code subrecord 1
func ParseEgtsSrTermIdentity(b []byte) interface{} {
	var d EgtsSrTermIdentity
	i := 0
	d.TerminalIdentifier = binary.LittleEndian.Uint16(b[i : i+4])
	i += 4
	// Flags
	flagBytes := uint16(b[i])
	i++
	d.HDIDE = utils.BitField(flagBytes, 0).(bool)
	d.IMEIE = utils.BitField(flagBytes, 1).(bool)
	d.IMSIE = utils.BitField(flagBytes, 2).(bool)
	d.LNGCE = utils.BitField(flagBytes, 3).(bool)
	d.SSRA = utils.BitField(flagBytes, 4).(bool)
	d.NIDE = utils.BitField(flagBytes, 5).(bool)
	d.BSE = utils.BitField(flagBytes, 6).(bool)
	d.MNE = utils.BitField(flagBytes, 7).(bool)
	if d.HDIDE {
		d.HomeDispatcherIdentifier = binary.LittleEndian.Uint16(b[i : i+2])
		i += 2
	}
	if d.IMEIE {
		d.InternationalMobileEquipmentIdentity = string(b[i : i+15])
		i += 15
	}
	if d.IMSIE {
		d.InternationalMobileSubscriberIdentity = string(b[i : i+16])
		i += 16
	}
	if d.LNGCE {
		d.LanguageCode = string(b[i : i+3])
		i += 3
	}
	if d.NIDE {
		d.NetworkIdentifier = binary.LittleEndian.Uint16(b[i : i+3])
		i += 3
	}
	if d.BSE {
		d.BufferSize = binary.LittleEndian.Uint16(b[i : i+2])
		i += 2
	}
	if d.MNE {
		d.MobileStationIntegratedServicesDigitalNetworkNumber = string(b[i : i+15])
		i += 15
	}
	return d
}
