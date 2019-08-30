package packet

import (
	"encoding/binary"
	"log"

	"github.com/LdDl/go-egts/crc"
	"github.com/LdDl/go-egts/egts/utils"
)

// Packet - Data packet (transport level)
type Packet struct {
	/* Header of packet */
	ProtocolVersion uint8 `json:"PRV"`  // PRV (Protocol Version)
	SecurityKeyID   uint8 `json:"SKID"` // SKID (Security Key ID)
	/*                 */

	/* Flags: PRF, PR, CMP, ENA, RTE */
	PRF int  `json:"PRF"` // PRF (Prefix)
	PR  int  `json:"PR"`  // PR (Priority)
	CMP bool `json:"CMP"` // CMP (Compression)
	ENA int  `json:"ENA"` // ENA (Encryption Algorithm)
	RTE bool `json:"RTE"` // RTE (Route)
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
	flagBytes := uint16(b[i])
	p.PR = utils.BitField(flagBytes, 0, 1).(int)  // 2 bits, PR
	p.CMP = utils.BitField(flagBytes, 2).(bool)   // 1 bit, CMP
	p.ENA = utils.BitField(flagBytes, 3, 4).(int) // 2 bits, ENA
	p.RTE = utils.BitField(flagBytes, 5).(bool)   // 1 bit, RTE
	p.PRF = utils.BitField(flagBytes, 6, 7).(int) // 1 bit, PRF

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
	if p.RTE {
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
