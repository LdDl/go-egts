package packet

import (
	"encoding/binary"

	"github.com/LdDl/go-egts/egts/subrecord"
)

// RecordData Service Data Record
type RecordData struct {
	SubrecordType   uint8       `json:"SRT"` // SRT (Subrecord Туре)
	SubrecordLength uint16      `json:"SRL"` // SRL (Subrecord Length)
	SubrecordData   interface{} `json:"SRD"` // SRD (Subrecord Data)
}

// BytesToRecordData Parse array of bytes to Service Data Record
func BytesToRecordData(b []byte) (p RecordData, err uint8) {
	p.SubrecordType = uint8(b[0])
	p.SubrecordLength = binary.LittleEndian.Uint16(b[1:3])
	p.SubrecordData, err = p.BytesToSubrecordData(b[3:])
	return p, err
}

// BytesToSubrecordData Parse array of bytes to subrecord data
func (rd *RecordData) BytesToSubrecordData(b []byte) (data interface{}, err uint8) {
	switch rd.SubrecordType {
	case RecordResponse:
		data = subrecord.BytesToSRRecordResponse(b)
		break
	case TermIdentity:
		data = subrecord.BytesToSRTermIdentity(b)
		break
	case PosData:
		data = subrecord.BytesToSRPosData(b)
		break
	case ExtPosData:
		data = subrecord.ParseEgtsSrExPosData(b)
		break
	case AdSensorsData:
		data = subrecord.BytesToSRAdSensorsData(b)
		break
	case CountersData:
		data = subrecord.BytesToSRCountersData(b)
		break
	case StateData:
		data = subrecord.BytesToSRStateData(b)
		break
	case LiquidLevelSensor:
		data = subrecord.ParseEgtsSrLiquidLevelSensor(b)
		if data == nil {
			err = EGTS_PC_SRVC_NFOUND
		}
		break
	default:
		err = EGTS_PC_SRVC_NFOUND
		break
	}
	return data, err
}
