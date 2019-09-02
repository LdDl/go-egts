package packet

import (
	"encoding/binary"
	"fmt"
	"strconv"
)

type BytesData interface {
	Decode([]byte)
	Encode() []byte
}

// ServicesFrameData SFRD (Services Frame Data)
type ServicesFrameData []*ServiceDataRecord

// ServiceDataRecord - формат отдельной записи Протокола Уровня Поддержки Услуг.
type ServiceDataRecord struct {
	RecordLength uint16 `json:"RL"` // RL (Record Length)
	RecordNumber uint16 `json:"RN"` // RN (Record Number)
	/* RecordFlags (RFL): SSOD, RSOD, GRP, RPP, TMFE, EVFE, OBFE */
	SSOD string `json:"SSOD"` // SSOD Source Service On Device
	RSOD string `json:"RSOD"` // RSOD Recipient Service On Device
	GRP  string `json:"GRP"`  // GRP Group
	RPP  string `json:"RPP"`  // RPP Record Processing Priority
	TMFE string `json:"TMFE"` // TMFE Time Field Exists
	EVFE string `json:"EVFE"` // EVFE Event ID Field Exists
	OBFE string `json:"OBFE"` // OBFE Object ID FieldExists
	/*                                                          */
	ObjectIdentifier     uint32      `json:"OID"`  // OID (Object Identifier)
	EventIdentifier      uint32      `json:"EVID"` // EVID (Event Identifier)
	Time                 uint32      `json:"TM"`   // TM (Time)
	SourceServiceType    uint8       `json:"SST"`  // SST (Source Service Type)
	RecipientServiceType uint8       `json:"RST"`  // RST (Recipient Service Type)
	RecordsData          RecordsData `json:"RD"`   // RD (Record Data)
}

// Decode Parse array of bytes to SFRD
func (sfrd *ServicesFrameData) Decode(b []byte) {
	i := 0
	for {
		sdr := ServiceDataRecord{}
		err := uint8(0)
		// RL (Record Length)
		sdr.RecordLength = binary.LittleEndian.Uint16(b[i : i+2])
		if sdr.RecordLength == 0 {
			err = EGTS_PC_INC_DATAFORM
			break
		}

		// RN (Record Number)
		i += 2
		sdr.RecordNumber = binary.LittleEndian.Uint16(b[i : i+2])

		// RecordFlags (RFL): SSOD, RSOD, GRP, RPP, TMFE, EVFE, OBFE
		i += 2
		flagByte := uint16(b[i])
		i++

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
			sdr.ObjectIdentifier = binary.LittleEndian.Uint32(b[i : i+4])
			i += 4
		}

		// EVID (Event Identifier)
		if sdr.EVFE == "1" {
			sdr.EventIdentifier = binary.LittleEndian.Uint32(b[i : i+4])
			i += 4
		}

		// TM (Time)
		if sdr.TMFE == "1" {
			sdr.Time = binary.LittleEndian.Uint32(b[i : i+4])
			i += 4
		}

		// SST (Source Service Type)
		sdr.SourceServiceType = uint8(b[i])

		// RST (Recipient Service Type)
		i++
		sdr.RecipientServiceType = uint8(b[i])

		// RD (Record Data)
		i++

		if len(b[i:i+int(sdr.RecordLength)]) != 0 {
			sdr.RecordsData = RecordsData{}
			sdr.RecordsData.Decode(b[i : i+int(sdr.RecordLength)])
			i += int(sdr.RecordLength)
		}

		*sfrd = append(*sfrd, &sdr)

		_ = err
		if i == len(b) {
			break
		}
	}
}

// Encode Parse SFRD to array of bytes
func (sfrd *ServicesFrameData) Encode() (b []byte) {
	for _, sdr := range *sfrd {
		rl := make([]byte, 2)
		binary.LittleEndian.PutUint16(rl, sdr.RecordLength)
		b = append(b, rl...)

		rn := make([]byte, 2)
		binary.LittleEndian.PutUint16(rn, sdr.RecordNumber)
		b = append(b, rn...)

		flagsBits := sdr.SSOD + sdr.RSOD + sdr.GRP + sdr.RPP + sdr.TMFE + sdr.EVFE + sdr.OBFE
		flags := uint64(0)
		flags, _ = strconv.ParseUint(flagsBits, 2, 8)
		b = append(b, uint8(flags))

		if sdr.OBFE == "1" {
			obfe := make([]byte, 2)
			binary.LittleEndian.PutUint32(obfe, sdr.ObjectIdentifier)
			b = append(b, obfe...)
		}
		if sdr.EVFE == "1" {
			evfe := make([]byte, 2)
			binary.LittleEndian.PutUint32(evfe, sdr.EventIdentifier)
			b = append(b, evfe...)
		}
		if sdr.TMFE == "1" {
			tmfe := make([]byte, 2)
			binary.LittleEndian.PutUint32(tmfe, sdr.Time)
			b = append(b, tmfe...)
		}

		b = append(b, sdr.SourceServiceType)
		b = append(b, sdr.RecipientServiceType)

		rd := sdr.RecordsData.Encode()

		b = append(b, rd...)
	}
	return b
}
