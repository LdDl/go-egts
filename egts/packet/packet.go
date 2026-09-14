package packet

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"io"
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
	p.ProtocolVersion, err = buffer.ReadByte()
	if err != nil {
		p.ErrorCode = EGTS_PC_INC_HEADERFORM
		return p, fmt.Errorf("Packet; EGTS_PC_INC_HEADERFORM; %w", err)
	}
	p.SecurityKeyID, err = buffer.ReadByte()
	if err != nil {
		p.ErrorCode = EGTS_PC_INC_HEADERFORM
		return p, fmt.Errorf("Packet; EGTS_PC_INC_HEADERFORM; %w", err)
	}
	flagByte := byte(0)
	flagByte, err = buffer.ReadByte()
	if err != nil {
		p.ErrorCode = EGTS_PC_INC_HEADERFORM
		return p, fmt.Errorf("Packet; EGTS_PC_INC_HEADERFORM; %w", err)
	}
	flagByteAsBits := fmt.Sprintf("%08b", flagByte)
	// Prefix
	p.PRF = flagByteAsBits[:2]
	// Route
	p.RTE = flagByteAsBits[2:3]
	// Encryption algorithm
	p.ENA = flagByteAsBits[3:5]
	// Compression
	p.CMP = flagByteAsBits[5:6]
	// Priority
	p.PR = flagByteAsBits[6:]

	p.HeaderLength, err = buffer.ReadByte()
	if err != nil {
		p.ErrorCode = EGTS_PC_INC_HEADERFORM
		return p, fmt.Errorf("Packet; EGTS_PC_INC_HEADERFORM; %w", err)
	}
	if p.RTE == "0" && p.HeaderLength != 11 {
		p.ErrorCode = EGTS_PC_INC_HEADERFORM
		return p, fmt.Errorf("Packet; EGTS_PC_INC_HEADERFORM; Header length must be 11 without routing")
	}
	if p.RTE == "1" && p.HeaderLength != 16 {
		p.ErrorCode = EGTS_PC_INC_HEADERFORM
		return p, fmt.Errorf("Packet; EGTS_PC_INC_HEADERFORM; Header length must be 16 with routing")
	}

	p.HeaderEncoding, err = buffer.ReadByte()
	if err != nil {
		p.ErrorCode = EGTS_PC_INC_HEADERFORM
		return p, fmt.Errorf("Packet; EGTS_PC_INC_HEADERFORM; %w", err)
	}

	tmpFDL := make([]byte, 2)
	_, err = io.ReadFull(buffer, tmpFDL)
	if err != nil {
		p.ErrorCode = EGTS_PC_INC_HEADERFORM
		return p, fmt.Errorf("Packet; EGTS_PC_INC_HEADERFORM; %w", err)
	}
	p.FrameDataLength = binary.LittleEndian.Uint16(tmpFDL)

	tmpPID := make([]byte, 2)
	_, err = io.ReadFull(buffer, tmpPID)
	if err != nil {
		p.ErrorCode = EGTS_PC_INC_HEADERFORM
		return p, fmt.Errorf("Packet; EGTS_PC_INC_HEADERFORM; %w", err)
	}
	p.PacketID = binary.LittleEndian.Uint16(tmpPID)

	p.PacketType, err = buffer.ReadByte()
	if err != nil {
		p.ErrorCode = EGTS_PC_INC_HEADERFORM
		return p, fmt.Errorf("Packet; EGTS_PC_INC_HEADERFORM; %w", err)
	}

	if p.RTE == "1" {
		tmpPeer := make([]byte, 2)
		_, err = io.ReadFull(buffer, tmpPeer)
		if err != nil {
			p.ErrorCode = EGTS_PC_INC_HEADERFORM
			return p, fmt.Errorf("Packet; EGTS_PC_INC_HEADERFORM; %w", err)
		}
		p.PeerAddress = binary.LittleEndian.Uint16(tmpPeer)

		tmpRecipient := make([]byte, 2)
		_, err = io.ReadFull(buffer, tmpRecipient)
		if err != nil {
			p.ErrorCode = EGTS_PC_INC_HEADERFORM
			return p, fmt.Errorf("Packet; EGTS_PC_INC_HEADERFORM; %w", err)
		}
		p.RecipientAddress = binary.LittleEndian.Uint16(tmpRecipient)

		p.TimeToLive, err = buffer.ReadByte()
		if err != nil {
			p.ErrorCode = EGTS_PC_INC_HEADERFORM
			return p, fmt.Errorf("Packet; EGTS_PC_INC_HEADERFORM; %w", err)
		}
	}

	p.HeaderCheckSum, err = buffer.ReadByte()
	if err != nil {
		p.ErrorCode = EGTS_PC_INC_HEADERFORM
		return p, fmt.Errorf("Packet; EGTS_PC_INC_HEADERFORM; %w", err)
	}

	// Evaluate crc-8
	if int(p.HeaderCheckSum) != crc.Crc(8, b[:int(p.HeaderLength)-1]) {
		p.ErrorCode = EGTS_PC_HEADERCRC_ERROR
		return p, fmt.Errorf("Packet; EGTS_PC_HEADERCRC_ERROR")
	}
	if p.ProtocolVersion != 1 {
		p.ErrorCode = EGTS_PC_UNS_PROTOCOL
		return p, fmt.Errorf("Packet; EGTS_PC_UNS_PROTOCOL")
	}
	if p.PRF != "00" || p.HeaderEncoding != 0 {
		p.ErrorCode = EGTS_PC_INC_HEADERFORM
		return p, fmt.Errorf("Packet; EGTS_PC_INC_HEADERFORM; Unsupported header format")
	}
	if p.ENA != "00" {
		p.ErrorCode = EGTS_PC_DECRYPT_ERROR
		return p, fmt.Errorf("Packet; EGTS_PC_DECRYPT_ERROR; Encryption is not supported")
	}
	if p.CMP != "0" {
		p.ErrorCode = EGTS_PC_UNS_TYPE
		return p, fmt.Errorf("Packet; EGTS_PC_UNS_TYPE; Compression is not supported")
	}

	// SFRCS is present only when the frame contains data.
	packetLength := int(p.HeaderLength) + int(p.FrameDataLength)
	if p.FrameDataLength > 0 {
		packetLength += 2
	}
	if packetLength != len(b) || packetLength > 65535 {
		p.ErrorCode = EGTS_PC_INVDATALEN
		return p, fmt.Errorf("Packet; EGTS_PC_INVDATALEN")
	}

	dataFrameBytes := make([]byte, p.FrameDataLength)
	_, err = io.ReadFull(buffer, dataFrameBytes)
	if err != nil {
		p.ErrorCode = EGTS_PC_INVDATALEN
		return p, fmt.Errorf("Packet; EGTS_PC_INVDATALEN; %w", err)
	}

	// Verify crc-16 before decoding the service data.
	if p.FrameDataLength > 0 {
		crc16Bytes := make([]byte, 2)
		_, err = io.ReadFull(buffer, crc16Bytes)
		if err != nil {
			p.ErrorCode = EGTS_PC_INVDATALEN
			return p, fmt.Errorf("Packet crc16; EGTS_PC_INVDATALEN; %w", err)
		}
		p.ServicesFrameDataCheckSum = binary.LittleEndian.Uint16(crc16Bytes)

		if int(p.ServicesFrameDataCheckSum) != crc.Crc(16, dataFrameBytes) {
			p.ErrorCode = EGTS_PC_DATACRC_ERROR
			return p, fmt.Errorf("Packet; EGTS_PC_DATACRC_ERROR")
		}
	}

	// Check type of packet
	switch p.PacketType {
	case EGTS_PT_RESPONSE:
		p.ServicesFrameData = &PTResponse{}
		break
	case EGTS_PT_APPDATA:
		p.ServicesFrameData = new(ServicesFrameData)
		break
	case EGTS_PT_SIGNED_APPDATA:
		p.ErrorCode = EGTS_PC_UNS_TYPE
		return p, fmt.Errorf("Packet; EGTS_PC_UNS_TYPE; Signed packets are not supported")
	default:
		p.ErrorCode = EGTS_PC_UNS_TYPE
		return p, fmt.Errorf("Packet; EGTS_PC_UNS_TYPE; Packet type %d", p.PacketType)
	}

	err = p.ServicesFrameData.Decode(dataFrameBytes)
	if err != nil {
		p.ErrorCode = EGTS_PC_INC_DATAFORM
		return p, fmt.Errorf("Packet dataFrame; EGTS_PC_INC_DATAFORM; %w", err)
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
func (p *Packet) PrepareAnswer(recordNum, pid uint16) Packet {

	if p.PacketType == EGTS_PT_APPDATA {
		var records RecordsData
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
						RecordNumber:         recordNum,
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
				PacketID:          pid,
				PacketType:        EGTS_PT_RESPONSE,
				ServicesFrameData: &resp,
			}
			return ans
		}
	}

	return Packet{}
}

// PrepareSRResultCode Prepare result code (SR_Result_Code) for incoming packet
func (p *Packet) PrepareSRResultCode(c uint8, recordNum, pid uint16) Packet {

	data := RecordsData{
		&RecordData{
			SubrecordType:   ResultCode,
			SubrecordLength: 1,
			SubrecordData: &subrecord.SRResultCode{
				RCD: c,
			},
		},
	}

	sfrd := ServicesFrameData{
		&ServiceDataRecord{
			RecordLength:         data.Len(),
			RecordNumber:         recordNum,
			SSOD:                 "0",
			RSOD:                 "0",
			GRP:                  "1",
			RPP:                  "00",
			TMFE:                 "0",
			EVFE:                 "0",
			OBFE:                 "0",
			SourceServiceType:    SERVICE_AUTH,
			RecipientServiceType: SERVICE_AUTH,
			RecordsData:          data,
		},
	}

	resp := Packet{
		ProtocolVersion:   1,
		SecurityKeyID:     0,
		PRF:               "00",
		RTE:               "0",
		ENA:               "00",
		CMP:               "0",
		PR:                "00",
		HeaderLength:      11,
		HeaderEncoding:    0,
		FrameDataLength:   sfrd.Len(),
		PacketID:          pid,
		PacketType:        EGTS_PT_APPDATA,
		ServicesFrameData: &sfrd,
	}

	return resp
}
