package packet

import (
	"encoding/binary"

	"github.com/LdDl/go-egts/crc"
)

// ResponseAuth Returns EGTS_PT_RESPONSE
func (p *Packet) ResponseAuth(pr uint8) (b []byte) {
	if p.PacketType == 1 {
		b := make([]byte, uint16(p.HeaderLength))
		i := 0
		b[i] = byte(p.ProtocolVersion) // PRV (Protocol Version)
		i++
		b[i] = byte(p.SecurityKeyID) // SKID (Security Key ID)
		i++
		b[i] = byte(1) // Flags: PRF (Prefix), RTE, ENA, CMP, PR
		i++
		b[i] = byte(11) // HL (Header Length)
		i++
		b[i] = byte(0) // HE (Header Encoding)
		i++
		binary.LittleEndian.PutUint16(b[i:i+2], 16) // FDL (Frame Data Length)
		i += 2
		binary.LittleEndian.PutUint16(b[i:i+2], p.PacketID) // PID (Packet Identifier)
		i += 2
		b[i] = byte(0) // PT (Packet Type) - 0, EGTS_PT_RESPONSE
		i++
		if p.RTE {
			binary.LittleEndian.PutUint16(b[i:i+2], p.PeerAddress) // PRA (Peer Address)
			i += 2
			binary.LittleEndian.PutUint16(b[i:i+2], p.RecipientAddress) // RCA (Recipient Address)
			i += 2
			b[i] = byte(p.TimeToLive) // TTL (Time To Live)
			i++
		}
		crcData := crc.Crc(8, b[:p.HeaderLength-1])
		b[i] = byte(uint8(crcData)) // HCS (Header Check Sum)
		i++

		/* SFRD (Services Frame Data) */
		bb := make([]byte, 16)
		ii := 0
		binary.LittleEndian.PutUint16(bb[ii:ii+2], p.PacketID) // RPID (Response Packet ID)
		ii += 2
		bb[ii] = byte(pr) // PR (Processing Result)
		ii++

		/* Service Data Record */
		binary.LittleEndian.PutUint16(bb[ii:ii+2], 6) // RL (Record Length)
		ii += 2
		binary.LittleEndian.PutUint16(bb[ii:ii+2], p.ServicesFrameData[0].RecordNumber) // RN (Record Number)
		ii += 2
		bb[ii] = byte(88) // RFL (Record Flags)
		ii++
		bb[ii] = byte(1) // SST (Source Service Type)
		ii++
		bb[ii] = byte(1) // RST (Recipient Service Type)
		ii++

		/* RD (Record Data) */
		bb[ii] = byte(0) // SRT (Subrecord Туре)
		ii++
		binary.LittleEndian.PutUint16(bb[ii:ii+2], 3) // SRL(Subrecord Length)
		ii += 2

		/* SRD (Subrecord Data) */
		binary.LittleEndian.PutUint16(bb[ii:ii+2], p.ServicesFrameData[0].RecordNumber) // CRN (Confirmed Record Number)
		ii += 2
		bb[ii] = byte(pr) // RST (Record Status)
		ii++

		crcData = crc.Crc(16, bb)
		crcByte := make([]byte, 2)
		binary.LittleEndian.PutUint16(crcByte, uint16(crcData))
		b = append(b, bb...)
		b = append(b, crcByte...)

		return b
	}
	return b
}

// Response - составляем ответ к полученному пакету с кодом обработки pr
// EGTS_SR_RECORD_RESPONSE - Подзапись применяется для осуществления подтверждения процесса обработки записи протокола уровня поддержки услуг. Данный тип подзаписи должен поддерживаться всеми сервисами
func (p *Packet) Response(sfd []byte, pr uint8, flag uint16) (b []byte) {
	if p.PacketType == 1 {
		b := make([]byte, uint16(p.HeaderLength)+p.FrameDataLength+5)
		i := 0
		b[i] = byte(p.ProtocolVersion) //PRV
		i++
		b[i] = byte(p.SecurityKeyID) //SKID
		i++
		b[i] = byte(flag) //flag
		i++
		b[i] = byte(p.HeaderLength) //HL
		i++
		b[i] = byte(p.HeaderEncoding) //HE
		i++
		binary.LittleEndian.PutUint16(b[i:i+2], p.FrameDataLength+3) //FDL //+3 byte (response info)
		i += 2
		binary.LittleEndian.PutUint16(b[i:i+2], p.PacketID) //PID
		i += 2
		b[i] = byte(0) //EGTS_PT_RESPONSE (packet type)
		i++
		if p.RTE {
			binary.LittleEndian.PutUint16(b[i:i+2], p.PeerAddress) //PRA
			i += 2
			binary.LittleEndian.PutUint16(b[i:i+2], p.RecipientAddress) //RCA
			i += 2
			b[i] = byte(p.TimeToLive) //TTL
			i++
		}
		crcData := crc.Crc(8, b[:p.HeaderLength-1])
		b[i] = byte(uint8(crcData)) //HCS
		i++
		bb := make([]byte, 3)
		binary.LittleEndian.PutUint16(bb[0:2], p.PacketID)
		bb[2] = byte(pr) // code rezult
		sfd := append(bb, sfd...)
		for j := 0; j < len(sfd); j++ {
			b[i] = sfd[j]
			i++
		}
		crcData = crc.Crc(16, sfd)
		binary.LittleEndian.PutUint16(b[i:i+2], uint16(crcData))
		return b
	}
	return b
}
