package packet

import (
	"encoding/binary"
	"log"

	"github.com/LdDl/go-egts/egts/utils"
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
	SSOD bool `json:"SSOD"` // SSOD Source Service On Device
	RSOD bool `json:"RSOD"` // RSOD Recipient Service On Device
	GRP  bool `json:"GRP"`  // GRP Group
	RPP  int  `json:"RPP"`  // RPP Record Processing Priority
	TMFE bool `json:"TMFE"` // TMFE Time Field Exists
	EVFE bool `json:"EVFE"` // EVFE Event ID Field Exists
	OBFE bool `json:"OBFE"` // OBFE Object ID FieldExists
	/*                                                          */
	ObjectIdentifier     uint32      `json:"OID"`  // OID (Object Identifier)
	EventIdentifier      uint32      `json:"EVID"` // EVID (Event Identifier)
	Time                 uint32      `json:"TM"`   // TM (Time)
	SourceServiceType    uint8       `json:"SST"`  // SST (Source Service Type)
	RecipientServiceType uint8       `json:"RST"`  // RST (Recipient Service Type)
	RecordData           RecordsData `json:"RD"`   // RD (Record Data)
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
		flagBytes := uint16(b[i])
		i++

		// OBFE Object ID FieldExists
		sdr.OBFE = utils.BitField(flagBytes, 0).(bool)
		// EVFE Event ID Field Exists
		sdr.EVFE = utils.BitField(flagBytes, 1).(bool)
		// TMFE Time Field Exists
		sdr.TMFE = utils.BitField(flagBytes, 2).(bool)
		// RPP Record Processing Priority
		sdr.RPP = utils.BitField(flagBytes, 3, 4).(int)
		// GRP Group
		sdr.GRP = utils.BitField(flagBytes, 5).(bool)
		// RSOD Recipient Service On Device
		sdr.RSOD = utils.BitField(flagBytes, 6).(bool)
		// SSOD Source Service On Device
		sdr.SSOD = utils.BitField(flagBytes, 7).(bool)

		// OID (Object Identifier)
		if sdr.OBFE {
			sdr.ObjectIdentifier = binary.LittleEndian.Uint32(b[i : i+4])
			i += 4
		}

		// EVID (Event Identifier)
		if sdr.EVFE {
			sdr.EventIdentifier = binary.LittleEndian.Uint32(b[i : i+4])
			i += 4
		}

		// TM (Time)
		if sdr.TMFE {
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
			sdr.RecordData = RecordsData{}
			sdr.RecordData.Decode(b[i : i+int(sdr.RecordLength)])
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
	log.Println("encoding sfrd")
	for _, sdr := range *sfrd {
		rl := make([]byte, 2)
		binary.LittleEndian.PutUint16(rl, sdr.RecordLength)
		b = append(b, rl...)

		rn := make([]byte, 2)
		binary.LittleEndian.PutUint16(rn, sdr.RecordNumber)
		b = append(b, rn...)

		flags := 0
		if sdr.OBFE {
			flags = utils.SetBit(flags, 0, 1)
		} else {
			flags = utils.SetBit(flags, 0, 0)
		}
		if sdr.EVFE {
			flags = utils.SetBit(flags, 1, 1)
		} else {
			flags = utils.SetBit(flags, 1, 0)
		}
		if sdr.TMFE {
			flags = utils.SetBit(flags, 2, 1)
		} else {
			flags = utils.SetBit(flags, 3, 0)
		}
		flags = utils.SetBit(flags, 4, sdr.RPP)
		if sdr.GRP {
			flags = utils.SetBit(flags, 5, 1)
		} else {
			flags = utils.SetBit(flags, 5, 0)
		}
		if sdr.RSOD {
			flags = utils.SetBit(flags, 6, 1)
		} else {
			flags = utils.SetBit(flags, 6, 0)
		}
		if sdr.SSOD {
			flags = utils.SetBit(flags, 7, 1)
		} else {
			flags = utils.SetBit(flags, 7, 0)
		}

		b = append(b, byte(flags))

		if sdr.OBFE {
			obfe := make([]byte, 2)
			binary.LittleEndian.PutUint32(obfe, sdr.ObjectIdentifier)
			b = append(b, obfe...)
		}
		if sdr.EVFE {
			evfe := make([]byte, 2)
			binary.LittleEndian.PutUint32(evfe, sdr.EventIdentifier)
			b = append(b, evfe...)
		}
		if sdr.TMFE {
			tmfe := make([]byte, 2)
			binary.LittleEndian.PutUint32(tmfe, sdr.Time)
			b = append(b, tmfe...)
		}

		b = append(b, sdr.SourceServiceType)
		b = append(b, sdr.RecipientServiceType)

		log.Println(b)

		rd := sdr.RecordData.Encode()
		b = append(b, rd...)
	}
	return b
}
