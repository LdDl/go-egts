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

// RecordsData Slice of RecordData
type RecordsData []*RecordData

// Decode Parse slice of bytes to Service Data Record
func (rd *RecordsData) Decode(b []byte) {
	i := 0

	for i != len(b) {
		rdEntity := &RecordData{}

		// SRT (Subrecord Туре)
		rdEntity.SubrecordType = uint8(b[i])

		// SRL (Subrecord Length)
		i++
		rdEntity.SubrecordLength = binary.LittleEndian.Uint16(b[i : i+2])

		// SRD (Subrecord Data)
		i += 2
		switch rdEntity.SubrecordType {
		case RecordResponse:
			rdEntity.SubrecordData = &subrecord.SRRecordResponse{}
			break
		case TermIdentity:
			rdEntity.SubrecordData = &subrecord.SRTermIdentity{}
			break
		case PosData:
			rdEntity.SubrecordData = &subrecord.SRPosData{}
			break
		case ExtPosData:
			rdEntity.SubrecordData = &subrecord.SRExPosDataRecord{}
			break
		case AdSensorsData:
			rdEntity.SubrecordData = &subrecord.SRAdSensorsData{}
			break
		case CountersData:
			rdEntity.SubrecordData = &subrecord.SRCountersData{}
			break
		case StateData:
			rdEntity.SubrecordData = &subrecord.SRStateData{}
			break
		case LiquidLevelSensor:
			rdEntity.SubrecordData = &subrecord.SRLiquidLevelSensor{}
			break
		default:
			// err = EGTS_PC_SRVC_NFOUND
			break
		}
		rdEntity.SubrecordData.Decode(b[i : i+int(rdEntity.SubrecordLength)])
		i += int(rdEntity.SubrecordLength)
		*rd = append(*rd, rdEntity)
	}
}

// Encode Parse Service Data Record to slice of bytes
func (rd *RecordsData) Encode() (b []byte) {
	for _, r := range *rd {
		b = append(b, r.SubrecordType)
		sl := make([]byte, 2)
		binary.LittleEndian.PutUint16(sl, r.SubrecordLength)
		b = append(b, sl...)
		sd := r.SubrecordData.Encode()
		b = append(b, sd...)
	}
	return b
}

// Len Returns length of bytes slice
func (rd *RecordsData) Len() (l uint16) {
	l = uint16(len(rd.Encode()))
	return l
}
