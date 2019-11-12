package packet

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"strconv"
)

// BytesData Interface for binary data
type BytesData interface {
	Decode([]byte) error
	Encode() ([]byte, error)
	Len() uint16
}

// ServicesFrameData SFRD (Services Frame Data)
type ServicesFrameData []*ServiceDataRecord

// ServiceDataRecord SDR (Service Data Record)
type ServiceDataRecord struct {
	/* Header section */
	RecordLength uint16 `json:"RL"` // RL (Record Length)
	RecordNumber uint16 `json:"RN"` // RN (Record Number)
	/* Flags section */
	SSOD string `json:"SSOD"` // SSOD Source Service On Device
	RSOD string `json:"RSOD"` // RSOD Recipient Service On Device
	GRP  string `json:"GRP"`  // GRP Group
	RPP  string `json:"RPP"`  // RPP Record Processing Priority
	TMFE string `json:"TMFE"` // TMFE Time Field Exists
	EVFE string `json:"EVFE"` // EVFE Event ID Field Exists
	OBFE string `json:"OBFE"` // OBFE Object ID FieldExists
	/* Data section */
	ObjectIdentifier     uint32      `json:"OID"`  // OID (Object Identifier)
	EventIdentifier      uint32      `json:"EVID"` // EVID (Event Identifier)
	Time                 uint32      `json:"TM"`   // TM (Time)
	SourceServiceType    uint8       `json:"SST"`  // SST (Source Service Type)
	RecipientServiceType uint8       `json:"RST"`  // RST (Recipient Service Type)
	RecordsData          RecordsData `json:"RD"`   // RD (Record Data)
}

// Decode Parse slice of bytes to SFRD
func (sfrd *ServicesFrameData) Decode(b []byte) (err error) {

	buffer := bytes.NewReader(b)

	for buffer.Len() > 0 {
		sdr := ServiceDataRecord{}

		// RL (Record Length)
		rl := make([]byte, 2)
		if _, err = buffer.Read(rl); err != nil {
			return fmt.Errorf("SFRD; Error reading RL")
		}
		sdr.RecordLength = binary.LittleEndian.Uint16(rl)
		if sdr.RecordLength == 0 {
			return fmt.Errorf("SFRD; EGTS_PC_INC_DATAFORM")
		}
		// RN (Record Number)
		rn := make([]byte, 2)
		if _, err = buffer.Read(rn); err != nil {
			return fmt.Errorf("SFRD; Error reading RN")
		}
		sdr.RecordNumber = binary.LittleEndian.Uint16(rn)

		// RecordFlags (RFL): SSOD, RSOD, GRP, RPP, TMFE, EVFE, OBFE
		flagByte := byte(0)
		if flagByte, err = buffer.ReadByte(); err != nil {
			return fmt.Errorf("SFRD; Error reading flags")
		}
		flagByteAsBits := fmt.Sprintf("%08b", flagByte)
		// OBFE Object ID FieldExists
		sdr.OBFE = flagByteAsBits[7:]
		// EVFE Event ID Field Exists
		sdr.EVFE = flagByteAsBits[6:7]
		// TMFE Time Field Exists
		sdr.TMFE = flagByteAsBits[5:6]
		// RPP Record Processing Priority
		sdr.RPP = flagByteAsBits[3:5]
		// GRP Group
		sdr.GRP = flagByteAsBits[2:3]
		// RSOD Recipient Service On Device
		sdr.RSOD = flagByteAsBits[1:2]
		// SSOD Source Service On Device
		sdr.SSOD = flagByteAsBits[:1]

		// OID (Object Identifier)
		if sdr.OBFE == "1" {
			oid := make([]byte, 4)
			if _, err = buffer.Read(oid); err != nil {
				return fmt.Errorf("SFRD; Error reading OID")
			}
			sdr.ObjectIdentifier = binary.LittleEndian.Uint32(oid)
		}

		// EVID (Event Identifier)
		if sdr.EVFE == "1" {
			evid := make([]byte, 4)
			if _, err = buffer.Read(evid); err != nil {
				return fmt.Errorf("SFRD; Error reading EVID")
			}
			sdr.EventIdentifier = binary.LittleEndian.Uint32(evid)
		}

		// TM (Time)
		if sdr.TMFE == "1" {
			tm := make([]byte, 4)
			if _, err = buffer.Read(tm); err != nil {
				return fmt.Errorf("SFRD; Error reading TM")
			}
			sdr.Time = binary.LittleEndian.Uint32(tm)
		}

		// SST (Source Service Type)
		if sdr.SourceServiceType, err = buffer.ReadByte(); err != nil {
			return fmt.Errorf("SFRD; Error reading SST")
		}

		// RST (Recipient Service Type)
		if sdr.RecipientServiceType, err = buffer.ReadByte(); err != nil {
			return fmt.Errorf("SFRD; Error reading RST")
		}

		// RD (Record Data)
		if buffer.Len() != 0 {
			sdr.RecordsData = RecordsData{}
			bb := make([]byte, sdr.RecordLength)
			if _, err = buffer.Read(bb); err != nil {
				return err
			}
			err := sdr.RecordsData.Decode(bb)
			if err != nil {
				return fmt.Errorf("SFRD;" + err.Error())
			}
		}
		*sfrd = append(*sfrd, &sdr)
	}
	return nil
}

// Encode Parse SFRD to slice of bytes
func (sfrd *ServicesFrameData) Encode() (b []byte, err error) {
	buffer := new(bytes.Buffer)
	for _, sdr := range *sfrd {
		if err = binary.Write(buffer, binary.LittleEndian, sdr.RecordLength); err != nil {
			return nil, fmt.Errorf("SFRD; Error writing RL")
		}
		if err = binary.Write(buffer, binary.LittleEndian, sdr.RecordNumber); err != nil {
			return nil, fmt.Errorf("SFRD; Error writing RN")
		}

		flagsBits := sdr.SSOD + sdr.RSOD + sdr.GRP + sdr.RPP + sdr.TMFE + sdr.EVFE + sdr.OBFE
		flags := uint64(0)
		flags, _ = strconv.ParseUint(flagsBits, 2, 8)
		if err = buffer.WriteByte(uint8(flags)); err != nil {
			return nil, fmt.Errorf("SFRD; Error writing flags")
		}

		if sdr.OBFE == "1" {
			if err = binary.Write(buffer, binary.LittleEndian, sdr.ObjectIdentifier); err != nil {
				return nil, fmt.Errorf("SFRD; Error writing OID")
			}
		}
		if sdr.EVFE == "1" {
			if err = binary.Write(buffer, binary.LittleEndian, sdr.EventIdentifier); err != nil {
				return nil, fmt.Errorf("SFRD; Error writing EVID")
			}
		}
		if sdr.TMFE == "1" {
			if err = binary.Write(buffer, binary.LittleEndian, sdr.Time); err != nil {
				return nil, fmt.Errorf("SFRD; Error writing TM")
			}
		}

		if err = buffer.WriteByte(sdr.SourceServiceType); err != nil {
			return nil, fmt.Errorf("SFRD; Error writing SST")
		}
		if err = buffer.WriteByte(sdr.RecipientServiceType); err != nil {
			return nil, fmt.Errorf("SFRD; Error writing RST")
		}

		rd, err := sdr.RecordsData.Encode()
		if err != nil {
			return nil, fmt.Errorf("SFRD;" + err.Error())
		}
		buffer.Write(rd)
	}
	return buffer.Bytes(), nil
}

// Len Returns length of bytes slice
func (sfrd *ServicesFrameData) Len() (l uint16) {
	encoded, _ := sfrd.Encode()
	l = uint16(len(encoded))
	return l
}
