package packet

import (
	"encoding/binary"

	"github.com/LdDl/go-egts/egts/utils"
)

// BytesToPTAppData - считывает данные поля SFRD в формате EGTS_PT_APPDATA
func BytesToPTAppData(b []byte) (sfrd ServicesFrameData, err uint8) {
	i := 0
	for {
		sdr := ServiceDataRecord{}

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
		sdr.RecordData, err = BytesToRecordData(b[i : i+int(sdr.RecordLength)])
		i += int(sdr.RecordLength)
		sfrd = append(sfrd, &sdr)
		if i == len(b) {
			break
		}
	}
	return
}
