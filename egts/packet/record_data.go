package packet

import (
	"encoding/binary"

	"github.com/LdDl/go-egts/egts/subrecord"
)

// RecordData Service Data Record
type RecordData struct {
	SubrecordType   uint8     `json:"SRT"` // SRT (Subrecord Туре)
	SubrecordLength uint16    `json:"SRL"` // SRL (Subrecord Length)
	SubrecordData   BytesData `json:"SRD"` // SRD (Subrecord Data)
}

// RecordsData Array of RecordData
type RecordsData []*RecordData

// Decode Parse array of bytes to Service Data Record
func (rd *RecordsData) Decode(b []byte) {
	rdEntity := &RecordData{}
	rdEntity.SubrecordType = uint8(b[0])
	rdEntity.SubrecordLength = binary.LittleEndian.Uint16(b[1:3])
	switch rdEntity.SubrecordType {
	case RecordResponse:
		rdEntity.SubrecordData = &subrecord.SRRecordResponse{}
		break
	case TermIdentity:
		// data = subrecord.BytesToSRTermIdentity(b)
		break
	case PosData:
		// data = subrecord.BytesToSRPosData(b)
		break
	case ExtPosData:
		// data = subrecord.ParseEgtsSrExPosData(b)
		break
	case AdSensorsData:
		// data = subrecord.BytesToSRAdSensorsData(b)
		break
	case CountersData:
		// data = subrecord.BytesToSRCountersData(b)
		break
	case StateData:
		// data = subrecord.BytesToSRStateData(b)
		break
	case LiquidLevelSensor:
		// data = subrecord.ParseEgtsSrLiquidLevelSensor(b)
		// if data == nil {
		// 	err = EGTS_PC_SRVC_NFOUND
		// }
		break
	default:
		// err = EGTS_PC_SRVC_NFOUND
		break
	}
	rdEntity.SubrecordData.Decode(b[3:])
	*rd = append(*rd, rdEntity)
}
