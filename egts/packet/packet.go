package packet

import (
	"encoding/binary"
	"fmt"
	"log"
	"strconv"

	"github.com/LdDl/go-egts/egts/subrecord"

	"github.com/LdDl/go-egts/crc"
)

// Packet - Data packet (transport level)
type Packet struct {
	/* Header of packet */
	ProtocolVersion uint8 `json:"PRV"`  // PRV (Protocol Version)
	SecurityKeyID   uint8 `json:"SKID"` // SKID (Security Key ID)
	/*                 */

	/* Flags: PRF, PR, CMP, ENA, RTE */
	PRF string `json:"PRF"` // PRF (Prefix)
	PR  string `json:"PR"`  // PR (Priority)
	CMP string `json:"CMP"` // CMP (Compression)
	ENA string `json:"ENA"` // ENA (Encryption Algorithm)
	RTE string `json:"RTE"` // RTE (Route)
	/*                              */

	HeaderLength     uint8  `json:"HL"`  // HL (Header Length)
	HeaderEncoding   uint8  `json:"HE"`  // HE (Header Encoding)
	FrameDataLength  uint16 `json:"FDL"` // FDL (Frame Data Length)
	PacketID         uint16 `json:"PID"` // PID (Packet Identifier)
	PacketType       uint8  `json:"PT"`  // PT (Packet Type)
	PeerAddress      uint16 `json:"PRA"` // PRA (Peer Address)
	RecipientAddress uint16 `json:"RCA"` // RCA (Recipient Address)
	TimeToLive       uint8  `json:"TTL"` // TTL (Time To Live)
	HeaderCheckSum   uint8  `json:"HCS"` // HCS (Header Check Sum)
	// Data for service level
	ServicesFrameData BytesData `json:"SFRD"` // SFRD (Services Frame Data)
	// Check sum for service level
	ServicesFrameDataCheckSum uint16 `json:"SFRCS"` // SFRCS

	// Response for packet
	ResponseData []byte `json:"-"`

	Error uint8 `json:"-"`
}

//ReadPacket - чтение пакета данных протокола транспортного уровня
func ReadPacket(b []byte) (p Packet, err uint8) {

	// PRV (Protocol Version)
	i := 0
	p.ProtocolVersion = uint8(b[i])
	if p.ProtocolVersion != 1 {
		err = EGTS_PC_UNS_PROTOCOL
		p.Error = err
		return
	}
	// SKID Security Key ID
	i++
	p.SecurityKeyID = uint8(b[i])

	// Flags: PRF, PR, CMP, ENA, RTE
	i++
	flagByteAsBits := fmt.Sprintf("%08b", uint16(b[i]))
	p.PR = flagByteAsBits[6:]   // 2 bits, PR
	p.CMP = flagByteAsBits[5:6] // 1 bit, CMP
	p.ENA = flagByteAsBits[3:5] // 2 bits, ENA
	p.RTE = flagByteAsBits[2:3] // 1 bit, RTE
	p.PRF = flagByteAsBits[:2]  // 1 bit, PRF

	i++
	p.HeaderLength = uint8(b[i]) // HL (Header Length)

	i++
	p.HeaderEncoding = uint8(b[i]) //HE (Header Encoding)

	// FDL (Frame Data Length)
	i++
	p.FrameDataLength = binary.LittleEndian.Uint16(b[i : i+2])

	// PID (Packet Identifier)
	i += 2
	p.PacketID = binary.LittleEndian.Uint16(b[i : i+2])

	// PT (Packet Type)
	i += 2
	p.PacketType = uint8(b[i])

	i++
	if p.RTE == "1" {
		// PRA (Peer Address)
		p.PeerAddress = binary.LittleEndian.Uint16(b[i : i+2])
		// RCA (Recipient Address)
		i += 2
		p.RecipientAddress = binary.LittleEndian.Uint16(b[i : i+2])
		// TTL (Time To Live)
		i += 2
		p.TimeToLive = uint8(b[i])
		i++
	}

	// HCS (Header Check Sum)
	p.HeaderCheckSum = uint8(b[i])

	// SFRCS
	i++
	if len(b[p.HeaderLength:uint16(p.HeaderLength)+p.FrameDataLength]) != int(p.FrameDataLength) {
		err = EGTS_PC_INC_DATAFORM
		p.Error = err
		return
	}
	p.ServicesFrameDataCheckSum = binary.LittleEndian.Uint16(b[uint16(p.HeaderLength)+p.FrameDataLength : uint16(p.HeaderLength)+p.FrameDataLength+2])
	if p.HeaderLength < 11 {
		err = EGTS_PC_INC_HEADERFORM
		p.Error = err
		return
	}

	// Evaluate crc-8
	crcData := crc.Crc(8, b[:p.HeaderLength-1])
	if int(crcData) != int(p.HeaderCheckSum) {
		err = EGTS_PC_HEADERCRC_ERROR
		p.Error = err
		return
	}
	// Evaluate crc-16
	crcData = crc.Crc(16, b[p.HeaderLength:uint16(p.HeaderLength)+p.FrameDataLength])
	if int(crcData) != int(p.ServicesFrameDataCheckSum) {
		err = EGTS_PC_DATACRC_ERROR
		p.Error = err
		return
	}

	switch p.PacketType {
	case EGTS_PT_RESPONSE:
		p.ServicesFrameData = &PTResponse{}
		break
	case EGTS_PT_APPDATA:
		p.ServicesFrameData = &ServicesFrameData{}
		break
	case EGTS_PT_SIGNED_APPDATA:
		// @todo
		break
	default:
		// nothing
		break
	}

	// SFRD (Services Frame Data)
	log.Println("Start parse packet", p.PacketType, p.HeaderLength, p.FrameDataLength, len(b[p.HeaderLength:uint16(p.HeaderLength)+p.FrameDataLength]))

	p.ServicesFrameData.Decode(b[p.HeaderLength : uint16(p.HeaderLength)+p.FrameDataLength])

	// p.ServicesFrameData, err = p.ReadServicesFrameData(b[p.HeaderLength : uint16(p.HeaderLength)+p.FrameDataLength])

	// на EGTS_SR_TERM_IDENTITY в ответ шлем EGTS_SR_RESULT_CODE в остальных случаях шлем EGTS_SR_RECORD_RESPONSE

	// if len(p.ServicesFrameData) == 1 && p.ServicesFrameData[0].RecordData.SubrecordType == 1 {
	// 	p.ResponseData = p.ResponseAuth(err)
	// } else {
	// 	p.ResponseData = p.Response(b[p.HeaderLength:uint16(p.HeaderLength)+p.FrameDataLength], err, flagBytes)
	// }

	return p, err
}

// Encode Parse EGTS_PT_RESPONSE to array of bytes
func (p *Packet) Encode() (b []byte) {

	b = append(b, p.ProtocolVersion)
	b = append(b, p.SecurityKeyID)

	flagsBits := p.PRF + p.RTE + p.ENA + p.CMP + p.PR
	flags := uint64(0)
	flags, _ = strconv.ParseUint(flagsBits, 2, 8)
	b = append(b, uint8(flags))

	b = append(b, p.HeaderLength)
	b = append(b, p.HeaderEncoding)

	fdl := make([]byte, 2)
	binary.LittleEndian.PutUint16(fdl, p.FrameDataLength)
	b = append(b, fdl...)

	pid := make([]byte, 2)
	binary.LittleEndian.PutUint16(pid, p.PacketID)
	b = append(b, pid...)

	b = append(b, p.PacketType)

	if p.RTE == "1" {
		peerA := make([]byte, 2)
		binary.LittleEndian.PutUint16(peerA, p.PeerAddress)
		b = append(b, peerA...)

		recepientA := make([]byte, 2)
		binary.LittleEndian.PutUint16(recepientA, p.RecipientAddress)
		b = append(b, recepientA...)

		b = append(b, p.TimeToLive)
	}

	crc8 := uint8(crc.Crc(8, b))
	b = append(b, crc8)
	if p.ServicesFrameData != nil {
		sfrd := p.ServicesFrameData.Encode()
		if len(sfrd) > 1 {
			b = append(b, sfrd...)
			crc16 := uint16(crc.Crc(16, sfrd))
			crc16hash := make([]byte, 2)
			binary.LittleEndian.PutUint16(crc16hash, crc16)
			b = append(b, crc16hash...)
		}
	}
	return b
}

// PrepareAnswer Prepare answer for incoming packet
func (p *Packet) PrepareAnswer() Packet {

	if p.PacketType == EGTS_PT_APPDATA {

		records := RecordsData{}
		serviceType := uint8(0)
		for _, r := range *p.ServicesFrameData.(*ServicesFrameData) {
			records = append(records, &RecordData{
				SubrecordType:   RecordResponse,
				SubrecordLength: 3,
				SubrecordData: &subrecord.SRRecordResponse{
					ConfirmedRecordNumber: r.RecordNumber,
					RecordStatus:          EGTS_PC_OK,
				},
			})
			serviceType = r.SourceServiceType
		}

		resp := PTResponse{
			ResponsePacketID: p.PacketID,
			ProcessingResult: p.Error,
		}

		if records != nil {
			resp.SDR = &ServicesFrameData{
				&ServiceDataRecord{
					RecordLength:         0, //todo
					RecordNumber:         0, //todo
					SSOD:                 "0",
					RSOD:                 "0",
					GRP:                  "1",
					RPP:                  "00",
					TMFE:                 "0",
					EVFE:                 "0",
					OBFE:                 "0",
					SourceServiceType:    serviceType,
					RecipientServiceType: serviceType,
					RecordsData:          records,
				},
			}
		}

		ans := Packet{
			ProtocolVersion:   1,
			SecurityKeyID:     0,
			PRF:               "00",
			RTE:               "0",
			ENA:               "00",
			CMP:               "0",
			PR:                "00",
			HeaderLength:      11,
			HeaderEncoding:    0,
			FrameDataLength:   0, // todo
			PacketID:          0, //todo
			PacketType:        EGTS_PT_RESPONSE,
			ServicesFrameData: &resp,
		}

		return ans
	}

	return Packet{}
}
