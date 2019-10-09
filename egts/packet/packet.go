package packet

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"strconv"

	"github.com/LdDl/go-egts/egts/subrecord"

	"github.com/LdDl/go-egts/crc"
)

// Packet EGTS packet
type Packet struct {
	/* Header section */
	ProtocolVersion uint8 `json:"PRV"`  // PRV (Protocol Version)
	SecurityKeyID   uint8 `json:"SKID"` // SKID (Security Key ID)
	/* Flags section */
	PRF string `json:"PRF"` // PRF (Prefix)
	PR  string `json:"PR"`  // PR (Priority)
	CMP string `json:"CMP"` // CMP (Compression)
	ENA string `json:"ENA"` // ENA (Encryption Algorithm)
	RTE string `json:"RTE"` // RTE (Route)
	/* Data section */
	HeaderLength              uint8     `json:"HL"`    // HL (Header Length)
	HeaderEncoding            uint8     `json:"HE"`    // HE (Header Encoding)
	FrameDataLength           uint16    `json:"FDL"`   // FDL (Frame Data Length)
	PacketID                  uint16    `json:"PID"`   // PID (Packet Identifier)
	PacketType                uint8     `json:"PT"`    // PT (Packet Type)
	PeerAddress               uint16    `json:"PRA"`   // PRA (Peer Address)
	RecipientAddress          uint16    `json:"RCA"`   // RCA (Recipient Address)
	TimeToLive                uint8     `json:"TTL"`   // TTL (Time To Live)
	HeaderCheckSum            uint8     `json:"HCS"`   // HCS (Header Check Sum)
	ServicesFrameData         BytesData `json:"SFRD"`  // SFRD (Services Frame Data)
	ServicesFrameDataCheckSum uint16    `json:"SFRCS"` // SFRCS

	ErrorCode uint8 `json:"-"`
}

// ReadPacket Parse slice of bytes as EGTS packet
func ReadPacket(b []byte) (p Packet, err error) {

	buffer := bytes.NewBuffer(b)
	// PR Processing Result
	if p.ProtocolVersion, err = buffer.ReadByte(); err != nil {
		p.ErrorCode = EGTS_PC_HEADERCRC_ERROR
		return p, fmt.Errorf("Packet; EGTS_PC_INC_HEADERFORM;" + err.Error())
	}
	if p.SecurityKeyID, err = buffer.ReadByte(); err != nil {
		p.ErrorCode = EGTS_PC_HEADERCRC_ERROR
		return p, fmt.Errorf("Packet; EGTS_PC_INC_HEADERFORM;" + err.Error())
	}
	flagByte := byte(0)
	if flagByte, err = buffer.ReadByte(); err != nil {
		p.ErrorCode = EGTS_PC_HEADERCRC_ERROR
		return p, fmt.Errorf("Packet; EGTS_PC_INC_HEADERFORM;" + err.Error())
	}
	flagByteAsBits := fmt.Sprintf("%08b", flagByte)
	p.PRF = flagByteAsBits[:2]  // flags << 7, flags << 6
	p.RTE = flagByteAsBits[2:3] // flags << 5
	p.ENA = flagByteAsBits[3:5] // flags << 4, flags << 3
	p.CMP = flagByteAsBits[5:6] // flags << 2
	p.PR = flagByteAsBits[6:]   // flags << 1, flags << 0

	if p.HeaderLength, err = buffer.ReadByte(); err != nil {
		p.ErrorCode = EGTS_PC_HEADERCRC_ERROR
		return p, fmt.Errorf("Packet; EGTS_PC_INC_HEADERFORM;" + err.Error())
	}

	if p.HeaderEncoding, err = buffer.ReadByte(); err != nil {
		p.ErrorCode = EGTS_PC_HEADERCRC_ERROR
		return p, fmt.Errorf("Packet; EGTS_PC_INC_HEADERFORM;" + err.Error())
	}

	tmpFDL := make([]byte, 2)
	if _, err = buffer.Read(tmpFDL); err != nil {
		p.ErrorCode = EGTS_PC_HEADERCRC_ERROR
		return p, fmt.Errorf("Packet; EGTS_PC_INC_HEADERFORM;" + err.Error())
	}
	p.FrameDataLength = binary.LittleEndian.Uint16(tmpFDL)

	tmpPID := make([]byte, 2)
	if _, err = buffer.Read(tmpPID); err != nil {
		p.ErrorCode = EGTS_PC_HEADERCRC_ERROR
		return p, fmt.Errorf("Packet; EGTS_PC_INC_HEADERFORM;" + err.Error())
	}
	p.PacketID = binary.LittleEndian.Uint16(tmpPID)

	if p.PacketType, err = buffer.ReadByte(); err != nil {
		p.ErrorCode = EGTS_PC_HEADERCRC_ERROR
		return p, fmt.Errorf("Packet; EGTS_PC_INC_HEADERFORM;" + err.Error())
	}

	if p.RTE == "1" {
		tmpPeer := make([]byte, 2)
		if _, err = buffer.Read(tmpPeer); err != nil {
			p.ErrorCode = EGTS_PC_HEADERCRC_ERROR
			return p, fmt.Errorf("Packet; EGTS_PC_INC_HEADERFORM;" + err.Error())
		}
		p.PeerAddress = binary.LittleEndian.Uint16(tmpPeer)

		tmpRecipient := make([]byte, 2)
		if _, err = buffer.Read(tmpRecipient); err != nil {
			p.ErrorCode = EGTS_PC_HEADERCRC_ERROR
			return p, fmt.Errorf("Packet; EGTS_PC_INC_HEADERFORM;" + err.Error())
		}
		p.RecipientAddress = binary.LittleEndian.Uint16(tmpRecipient)

		if p.TimeToLive, err = buffer.ReadByte(); err != nil {
			p.ErrorCode = EGTS_PC_HEADERCRC_ERROR
			return p, fmt.Errorf("Packet; EGTS_PC_INC_HEADERFORM;" + err.Error())
		}
	}

	if p.HeaderCheckSum, err = buffer.ReadByte(); err != nil {
		p.ErrorCode = EGTS_PC_HEADERCRC_ERROR
		return p, fmt.Errorf("Packet; EGTS_PC_INC_HEADERFORM;" + err.Error())
	}

	// Evaluate crc-8
	if int(p.HeaderCheckSum) != crc.Crc(8, b[:p.HeaderLength-1]) {
		p.ErrorCode = EGTS_PC_HEADERCRC_ERROR
		return p, fmt.Errorf("Packet; EGTS_PC_HEADERCRC_ERROR")
	}
	dataFrameBytes := make([]byte, p.FrameDataLength)
	if _, err = buffer.Read(dataFrameBytes); err != nil {
		p.ErrorCode = EGTS_PC_INC_DATAFORM
		return p, fmt.Errorf("Packet; EGTS_PC_INC_DATAFORM;" + err.Error())
	}

	// Check type of packet
	switch p.PacketType {
	case EGTS_PT_RESPONSE:
		p.ServicesFrameData = &PTResponse{}
		break
	case EGTS_PT_APPDATA:
		p.ServicesFrameData = &ServicesFrameData{}
		break
	case EGTS_PT_SIGNED_APPDATA:
		// @TODO (not implemented yet)
		break
	default:
		// nothing
		break
	}

	if err = p.ServicesFrameData.Decode(dataFrameBytes); err != nil {
		p.ErrorCode = EGTS_PC_DECRYPT_ERROR
		return p, fmt.Errorf("Packet dataFrame; EGTS_PC_DECRYPT_ERROR;" + err.Error())
	}

	crc16Bytes := make([]byte, 2)
	if _, err = buffer.Read(crc16Bytes); err != nil {
		p.ErrorCode = EGTS_PC_DECRYPT_ERROR
		return p, fmt.Errorf("Packet crc16; EGTS_PC_DECRYPT_ERROR;" + err.Error())
	}
	p.ServicesFrameDataCheckSum = binary.LittleEndian.Uint16(crc16Bytes)

	if int(p.ServicesFrameDataCheckSum) != crc.Crc(16, b[p.HeaderLength:uint16(p.HeaderLength)+p.FrameDataLength]) {
		p.ErrorCode = EGTS_PC_DATACRC_ERROR
		return p, fmt.Errorf("Packet; EGTS_PC_DATACRC_ERROR")
	}

	return p, nil
}

// Encode Parse EGTS_PT_RESPONSE to slice of bytes
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
		sfrd, _ := p.ServicesFrameData.Encode()
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
		if p.ServicesFrameData != nil {
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
				ProcessingResult: p.ErrorCode,
			}

			if records != nil {
				resp.SDR = &ServicesFrameData{
					&ServiceDataRecord{
						RecordLength:         records.Len(),
						RecordNumber:         0, // @todo
						SSOD:                 "0",
						RSOD:                 "1",
						GRP:                  "0",
						RPP:                  "11",
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
				PR:                "11",
				HeaderLength:      11,
				HeaderEncoding:    0,
				FrameDataLength:   resp.Len(),
				PacketID:          p.PacketID, // @todo
				PacketType:        EGTS_PT_RESPONSE,
				ServicesFrameData: &resp,
			}
			return ans
		}
	}

	return Packet{}
}
