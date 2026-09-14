package subrecord

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"io"
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
	decoded := SRTermIdentity{}

	// TID (Terminal Identifier)
	tid := make([]byte, 4)
	_, err = io.ReadFull(buffer, tid)
	if err != nil {
		return fmt.Errorf("EGTS_SR_TERM_IDENTITY; Error reading TID: %w", err)
	}
	decoded.TerminalIdentifier = binary.LittleEndian.Uint32(tid)

	// Flags: MNE, BSE, NIDE, SSRA, LNGCE, IMSIE, IMEIE, HDIDE
	flagByte := byte(0)
	flagByte, err = buffer.ReadByte()
	if err != nil {
		return fmt.Errorf("EGTS_SR_TERM_IDENTITY; Error reading flags: %w", err)
	}
	flagByteAsBits := fmt.Sprintf("%08b", flagByte)
	decoded.HDIDE = flagByteAsBits[7:]
	decoded.IMEIE = flagByteAsBits[6:7]
	decoded.IMSIE = flagByteAsBits[5:6]
	decoded.LNGCE = flagByteAsBits[4:5]
	decoded.SSRA = flagByteAsBits[3:4]
	decoded.NIDE = flagByteAsBits[2:3]
	decoded.BSE = flagByteAsBits[1:2]
	decoded.MNE = flagByteAsBits[:1]

	// HDID (Home Dispatcher Identifier)
	if decoded.HDIDE == "1" {
		hdid := make([]byte, 2)
		_, err = io.ReadFull(buffer, hdid)
		if err != nil {
			return fmt.Errorf("EGTS_SR_TERM_IDENTITY; Error reading HDID: %w", err)
		}
		decoded.HomeDispatcherIdentifier = binary.LittleEndian.Uint16(hdid)
	}

	// IMEI (International Mobile Equipment Identity)
	if decoded.IMEIE == "1" {
		imei := make([]byte, 15)
		_, err = io.ReadFull(buffer, imei)
		if err != nil {
			return fmt.Errorf("EGTS_SR_TERM_IDENTITY; Error reading IMEI: %w", err)
		}
		decoded.InternationalMobileEquipmentIdentity = string(imei)
	}

	// IMSI (International Mobile Subscriber Identity)
	if decoded.IMSIE == "1" {
		imsi := make([]byte, 16)
		_, err = io.ReadFull(buffer, imsi)
		if err != nil {
			return fmt.Errorf("EGTS_SR_TERM_IDENTITY; Error reading IMSI: %w", err)
		}
		decoded.InternationalMobileSubscriberIdentity = string(imsi)
	}

	// LNGC (Language Code)
	if decoded.LNGCE == "1" {
		lang := make([]byte, 3)
		_, err = io.ReadFull(buffer, lang)
		if err != nil {
			return fmt.Errorf("EGTS_SR_TERM_IDENTITY; Error reading LNGC: %w", err)
		}
		decoded.LanguageCode = string(lang)
	}

	// NID (Network Identifier)
	if decoded.NIDE == "1" {
		decoded.NetworkIdentifier = make([]byte, 3)
		_, err = io.ReadFull(buffer, decoded.NetworkIdentifier)
		if err != nil {
			return fmt.Errorf("EGTS_SR_TERM_IDENTITY; Error reading NID: %w", err)
		}
	}

	// BS (Buffer Size)
	if decoded.BSE == "1" {
		bufSize := make([]byte, 2)
		_, err = io.ReadFull(buffer, bufSize)
		if err != nil {
			return fmt.Errorf("EGTS_SR_TERM_IDENTITY; Error reading BS: %w", err)
		}
		decoded.BufferSize = binary.LittleEndian.Uint16(bufSize)
	}

	// MSISDN (Mobile Station Integrated Services Digital Network Number)
	if decoded.MNE == "1" {
		mne := make([]byte, 15)
		_, err = io.ReadFull(buffer, mne)
		if err != nil {
			return fmt.Errorf("EGTS_SR_TERM_IDENTITY; Error reading MSISDN: %w", err)
		}
		decoded.MobileStationIntegratedServicesDigitalNetworkNumber = string(mne)
	}

	if buffer.Len() != 0 {
		return fmt.Errorf("EGTS_SR_TERM_IDENTITY; Unexpected trailing data")
	}
	*subr = decoded
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
