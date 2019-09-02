package subrecord

import (
	"encoding/binary"
	"fmt"
	"strconv"
)

// SRTermIdentity EGTS_SR_TERM_IDENTITY
type SRTermIdentity struct {
	TerminalIdentifier uint32 `json:"TID"` // TID (Terminal Identifier)

	/* Flags: MNE, BSE, NIDE, SSRA, LNGCE, IMSIE, IMEIE, HDIDE */
	MNE   string `json:"MNE"`   // MNE
	BSE   string `json:"BSE"`   // BSE
	NIDE  string `json:"NIDE"`  // NIDE
	SSRA  string `json:"SSRA"`  // SSRA
	LNGCE string `json:"LNGCE"` // LNGCE
	IMSIE string `json:"IMSIE"` // IMSIE
	IMEIE string `json:"IMEIE"` // IMEIE
	HDIDE string `json:"HDIDE"` // HDIDE
	/*                                                        */
	HomeDispatcherIdentifier                            uint16 `json:"HDID"`   // HDID (Home Dispatcher Identifier)
	InternationalMobileEquipmentIdentity                string `json:"IMEI"`   // IMEI (International Mobile Equipment Identity)
	InternationalMobileSubscriberIdentity               string `json:"IMSI"`   // IMSI (International Mobile Subscriber Identity)
	LanguageCode                                        string `json:"LNGC"`   // LNGC (Language Code)
	NetworkIdentifier                                   []byte `json:"NID"`    // NID (Network Identifier)
	BufferSize                                          uint16 `json:"BS"`     // BS (Buffer Size)
	MobileStationIntegratedServicesDigitalNetworkNumber string `json:"MSISDN"` // MSISDN (Mobile Station Integrated Services Digital Network Number)
}

// Decode Parse array of bytes to EGTS_SR_TERM_IDENTITY
func (subr *SRTermIdentity) Decode(b []byte) {
	// TID (Terminal Identifier)
	i := 0
	subr.TerminalIdentifier = binary.LittleEndian.Uint32(b[i : i+4])

	// Flags: MNE, BSE, NIDE, SSRA, LNGCE, IMSIE, IMEIE, HDIDE
	i += 4
	flagByte := uint16(b[i])
	flagByteAsBits := fmt.Sprintf("%08b", flagByte)

	// HDIDE
	subr.HDIDE = flagByteAsBits[7:]
	// IMEIE
	subr.IMEIE = flagByteAsBits[6:7]
	// IMSIE
	subr.IMSIE = flagByteAsBits[5:6]
	// LNGCE
	subr.LNGCE = flagByteAsBits[4:5]
	// SSRA
	subr.SSRA = flagByteAsBits[3:4]
	// NIDE
	subr.NIDE = flagByteAsBits[2:3]
	// BSE
	subr.BSE = flagByteAsBits[1:2]
	// MNE
	subr.MNE = flagByteAsBits[:1]

	// HDID (Home Dispatcher Identifier)
	i++
	if subr.HDIDE == "1" {
		subr.HomeDispatcherIdentifier = binary.LittleEndian.Uint16(b[i : i+2])
		i += 2
	}

	// IMEI (International Mobile Equipment Identity)
	if subr.IMEIE == "1" {
		subr.InternationalMobileEquipmentIdentity = string(b[i : i+15])
		i += 15
	}

	// IMSI (International Mobile Subscriber Identity)
	if subr.IMSIE == "1" {
		subr.InternationalMobileSubscriberIdentity = string(b[i : i+16])
		i += 16
	}

	// LNGC (Language Code)
	if subr.LNGCE == "1" {
		subr.LanguageCode = string(b[i : i+3])
		i += 3
	}

	// NID (Network Identifier)
	if subr.NIDE == "1" {
		subr.NetworkIdentifier = make([]byte, 3)
		copy(subr.NetworkIdentifier, b[i:i+3])
		i += 3
	}

	// BS (Buffer Size)
	if subr.BSE == "1" {
		subr.BufferSize = binary.LittleEndian.Uint16(b[i : i+2])
		i += 2
	}

	// MSISDN (Mobile Station Integrated Services Digital Network Number)
	if subr.MNE == "1" {
		subr.MobileStationIntegratedServicesDigitalNetworkNumber = string(b[i : i+15])
		i += 15
	}
}

// Encode Parse EGTS_SR_TERM_IDENTITY to array of bytes
func (subr *SRTermIdentity) Encode() (b []byte) {

	tid := make([]byte, 4)
	binary.LittleEndian.PutUint32(tid, subr.TerminalIdentifier)
	b = append(b, tid...)

	flagsBits := subr.MNE + subr.BSE + subr.NIDE + subr.SSRA + subr.LNGCE + subr.IMSIE + subr.IMEIE + subr.HDIDE
	flags := uint64(0)
	flags, _ = strconv.ParseUint(flagsBits, 2, 8)
	b = append(b, uint8(flags))

	if subr.HDIDE == "1" {
		hdid := make([]byte, 2)
		binary.LittleEndian.PutUint16(hdid, subr.HomeDispatcherIdentifier)
		b = append(b, hdid...)
	}

	if subr.IMEIE == "1" {
		b = append(b, []byte(subr.InternationalMobileEquipmentIdentity)...)
	}

	if subr.IMSIE == "1" {
		b = append(b, []byte(subr.InternationalMobileSubscriberIdentity)...)
	}

	if subr.LNGCE == "1" {
		b = append(b, []byte(subr.LanguageCode)...)
	}

	if subr.NIDE == "1" {
		b = append(b, subr.NetworkIdentifier...)
	}

	if subr.BSE == "1" {
		nid := make([]byte, 2)
		binary.LittleEndian.PutUint16(nid, subr.BufferSize)
		b = append(b, nid...)
	}

	if subr.MNE == "1" {
		b = append(b, []byte(subr.MobileStationIntegratedServicesDigitalNetworkNumber)...)
	}

	return b
}
