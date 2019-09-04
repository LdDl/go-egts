package subrecord

import (
	"bytes"
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
func (subr *SRTermIdentity) Decode(b []byte) (err error) {
	buffer := bytes.NewReader(b)

	// TID (Terminal Identifier)
	tid := make([]byte, 4)
	if _, err = buffer.Read(tid); err != nil {
		return fmt.Errorf("EGTS_SR_TERM_IDENTITY; Error reading TID")
	}
	subr.TerminalIdentifier = binary.LittleEndian.Uint32(tid)

	// Flags: MNE, BSE, NIDE, SSRA, LNGCE, IMSIE, IMEIE, HDIDE
	flagByte := byte(0)
	if flagByte, err = buffer.ReadByte(); err != nil {
		return fmt.Errorf("EGTS_SR_TERM_IDENTITY; Error reading flags")
	}
	flagByteAsBits := fmt.Sprintf("%08b", flagByte)
	subr.HDIDE = flagByteAsBits[7:]
	subr.IMEIE = flagByteAsBits[6:7]
	subr.IMSIE = flagByteAsBits[5:6]
	subr.LNGCE = flagByteAsBits[4:5]
	subr.SSRA = flagByteAsBits[3:4]
	subr.NIDE = flagByteAsBits[2:3]
	subr.BSE = flagByteAsBits[1:2]
	subr.MNE = flagByteAsBits[:1]

	// HDID (Home Dispatcher Identifier)
	if subr.HDIDE == "1" {
		hdid := make([]byte, 2)
		if _, err = buffer.Read(hdid); err != nil {
			return fmt.Errorf("EGTS_SR_TERM_IDENTITY; Error reading HDID")
		}
		subr.HomeDispatcherIdentifier = binary.LittleEndian.Uint16(hdid)
	}

	// IMEI (International Mobile Equipment Identity)
	if subr.IMEIE == "1" {
		imei := make([]byte, 15)
		if _, err = buffer.Read(imei); err != nil {
			return fmt.Errorf("EGTS_SR_TERM_IDENTITY; Error reading IMEI")
		}
		subr.InternationalMobileEquipmentIdentity = string(imei)
	}

	// IMSI (International Mobile Subscriber Identity)
	if subr.IMSIE == "1" {
		imsi := make([]byte, 16)
		if _, err = buffer.Read(imsi); err != nil {
			return fmt.Errorf("EGTS_SR_TERM_IDENTITY; Error reading IMSI")
		}
		subr.InternationalMobileSubscriberIdentity = string(imsi)
	}

	// LNGC (Language Code)
	if subr.LNGCE == "1" {
		lang := make([]byte, 3)
		if _, err = buffer.Read(lang); err != nil {
			return fmt.Errorf("EGTS_SR_TERM_IDENTITY; Error reading LNGC")
		}
		subr.LanguageCode = string(lang)
	}

	// NID (Network Identifier)
	if subr.NIDE == "1" {
		subr.NetworkIdentifier = make([]byte, 3)
		if _, err = buffer.Read(subr.NetworkIdentifier); err != nil {
			return fmt.Errorf("EGTS_SR_TERM_IDENTITY; Error reading NID")
		}
	}

	// BS (Buffer Size)
	if subr.BSE == "1" {
		bufSize := make([]byte, 2)
		if _, err = buffer.Read(bufSize); err != nil {
			return fmt.Errorf("EGTS_SR_TERM_IDENTITY; Error reading BS")
		}
		subr.BufferSize = binary.LittleEndian.Uint16(bufSize)
	}

	// MSISDN (Mobile Station Integrated Services Digital Network Number)
	if subr.MNE == "1" {
		mne := make([]byte, 15)
		if _, err = buffer.Read(mne); err != nil {
			return fmt.Errorf("EGTS_SR_TERM_IDENTITY; Error reading MSISDN")
		}
		subr.MobileStationIntegratedServicesDigitalNetworkNumber = string(mne)
	}

	return nil
}

// Encode Parse EGTS_SR_TERM_IDENTITY to array of bytes
func (subr *SRTermIdentity) Encode() (b []byte, err error) {
	buffer := new(bytes.Buffer)

	if err = binary.Write(buffer, binary.LittleEndian, subr.TerminalIdentifier); err != nil {
		return nil, fmt.Errorf("EGTS_SR_TERM_IDENTITY; Error writing TID")
	}

	flagsBits := subr.MNE + subr.BSE + subr.NIDE + subr.SSRA + subr.LNGCE + subr.IMSIE + subr.IMEIE + subr.HDIDE
	flags := uint64(0)
	flags, _ = strconv.ParseUint(flagsBits, 2, 8)
	if err = buffer.WriteByte(uint8(flags)); err != nil {
		return nil, fmt.Errorf("EGTS_SR_TERM_IDENTITY; Error writing flags")
	}

	if subr.HDIDE == "1" {
		if err = binary.Write(buffer, binary.LittleEndian, subr.HomeDispatcherIdentifier); err != nil {
			return nil, fmt.Errorf("EGTS_SR_TERM_IDENTITY; Error writing HDID")
		}
	}

	if subr.IMEIE == "1" {
		if _, err = buffer.Write([]byte(subr.InternationalMobileEquipmentIdentity)); err != nil {
			return nil, fmt.Errorf("EGTS_SR_TERM_IDENTITY; Error writing IMEI")
		}
	}

	if subr.IMSIE == "1" {
		if _, err = buffer.Write([]byte(subr.InternationalMobileSubscriberIdentity)); err != nil {
			return nil, fmt.Errorf("EGTS_SR_TERM_IDENTITY; Error writing IMSI")
		}
	}

	if subr.LNGCE == "1" {
		if _, err = buffer.Write([]byte(subr.LanguageCode)); err != nil {
			return nil, fmt.Errorf("EGTS_SR_TERM_IDENTITY; Error writing LNGC")
		}
	}

	if subr.NIDE == "1" {
		if _, err = buffer.Write(subr.NetworkIdentifier); err != nil {
			return nil, fmt.Errorf("EGTS_SR_TERM_IDENTITY; Error writing NID")
		}
	}

	if subr.BSE == "1" {
		if err = binary.Write(buffer, binary.LittleEndian, subr.BufferSize); err != nil {
			return nil, fmt.Errorf("EGTS_SR_TERM_IDENTITY; Error writing NID")
		}
	}

	if subr.MNE == "1" {
		if _, err = buffer.Write([]byte(subr.MobileStationIntegratedServicesDigitalNetworkNumber)); err != nil {
			return nil, fmt.Errorf("EGTS_SR_TERM_IDENTITY; Error writing MSISDN")
		}
	}

	return buffer.Bytes(), nil
}

// Len Returns length of bytes slice
func (subr *SRTermIdentity) Len() (l uint16) {
	encoded, _ := subr.Encode()
	l = uint16(len(encoded))
	return l
}
