package packet

import (
	"bytes"
	"encoding/binary"
	"fmt"

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
func (rd *RecordsData) Decode(b []byte) (err error) {
	buffer := bytes.NewBuffer(b)

	for buffer.Len() > 0 {
		rdEntity := &RecordData{}

		// SRT (Subrecord Туре)
		if rdEntity.SubrecordType, err = buffer.ReadByte(); err != nil {
			return fmt.Errorf("SRD; Error reading SRT")
		}

		// SRL (Subrecord Length)
		srl := make([]byte, 2)
		if _, err = buffer.Read(srl); err != nil {
			return fmt.Errorf("SRD; Error reading SRL")
		}
		rdEntity.SubrecordLength = binary.LittleEndian.Uint16(srl)

		// SRD (Subrecord Data)
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
		case AccelerationData:
			rdEntity.SubrecordData = &subrecord.SRAccelerationHeader{}
			break
		case StateData:
			rdEntity.SubrecordData = &subrecord.SRStateData{}
			break
		case LiquidLevelSensor:
			rdEntity.SubrecordData = &subrecord.SRLiquidLevelSensor{}
			break
		default:
			err = fmt.Errorf("RD;EGTS_PC_SRVC_NFOUND")
			return err
		}

		bb := buffer.Next(int(rdEntity.SubrecordLength))
		err := rdEntity.SubrecordData.Decode(bb)
		if err != nil {
			return fmt.Errorf("SRD;" + err.Error())
		}
		*rd = append(*rd, rdEntity)
	}
	return nil
}

// Encode Parse Service Data Record to slice of bytes
func (rd *RecordsData) Encode() (b []byte, err error) {
	buffer := new(bytes.Buffer)
	for _, r := range *rd {
		if err = buffer.WriteByte(r.SubrecordType); err != nil {
			return nil, fmt.Errorf("SRD; Error writing SRT")
		}
		if err = binary.Write(buffer, binary.LittleEndian, r.SubrecordLength); err != nil {
			return nil, fmt.Errorf("SRD; Error writing SRL")
		}
		sd, err := r.SubrecordData.Encode()
		if err != nil {
			return nil, fmt.Errorf("SRD;" + err.Error())
		}
		buffer.Write(sd)
	}
	return buffer.Bytes(), nil
}

// Len Returns length of bytes slice
func (rd *RecordsData) Len() (l uint16) {
	encoded, _ := rd.Encode()
	l = uint16(len(encoded))
	return l
}
