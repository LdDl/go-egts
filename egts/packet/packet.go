package packet

import (
	"encoding/binary"

	"github.com/LdDl/go-egts/crc"
	"github.com/LdDl/go-egts/egts/subrecord"
	"github.com/LdDl/go-egts/egts/utils"
)

// Packet - Data packet (transport level)
type Packet struct {
	// Header of packet
	ProtocolVersion  uint8  // PRV
	SecurityKeyID    uint8  // SKID
	HeaderLength     uint8  // HL
	HeaderEncoding   uint8  // HE
	FrameDataLength  uint16 // FDL
	PacketID         uint16 // PID
	PacketType       uint8  // PT
	PeerAddress      uint16 // PRA
	RecipientAddress uint16 //RCA
	TimeToLive       uint8  // TTL
	HeaderCheckSum   uint8  // HCS
	// Data for service level
	ServicesFrameData []ServicesFrameData
	// Check sum for service level
	ServicesFrameDataCheckSum uint16 // SFRCS
	// Flag: PRF, PR, CMP, ENA, RTE.
	PRF int
	RTE bool
	ENA int
	CMP bool
	PR  int
	// Response for packet
	ResponseData []byte
}

/*
Пакет данных протокола транспортного уровня
Описание заголовочного поля Flag:
	Name	Bit Value
	PRF		7-6	префикс заголовка Транспортного Уровня и для данной версии должен содержать значение 00
	RTE		5		определяет необходимость дальнейшей маршрутизации = 1, то необходима
	ENA		4-3	шифрование данных из поля SFRD, значение 0 0 , то данные в поле SFRD не шифруются
	CMP		2		сжатие данных из поля SFRD, = 1, то данные в поле SFRD считаются сжатыми
	PR		1-0	приоритет маршрутизации, 1 0 – средний
Описание заголовочного поля PacketType(PT):
	0 - EGTS_PT_RESPONSE (подтверждение на пакет Транспортного Уровня);
	1 – EGTS_PT_APPDATA (пакет содержащий данные  ППУ);
	2 – EGTS_PT_SIGNED_APPDATA (пакет содержащий данные  ППУс цифровой подписью)

*/

//ServicesFrameData - формат отдельной записи Протокола Уровня Поддержки Услуг.
type ServicesFrameData struct {
	RecordLength         uint16     // RL
	RecordNumber         uint16     // RN
	ObjectIdentifier     uint32     // OID
	EventIdentifier      uint32     // EVID
	Time                 uint32     //TM
	SourceServiceType    uint8      // SST
	RecipientServiceType uint8      // RST
	RecordData           RecordData // RD
	// RecordFlags (RFL)
	OBFE bool
	EVFE bool
	TMFE bool
	RPP  int
	GRP  bool
	RSOD bool
	SSOD bool
}

//RecordData - формат отдельной подзаписи Протокола Уровня Поддержки Услуг.
type RecordData struct {
	SubrecordType   uint8  // SRT
	SubrecordLength uint16 // SRL
	SubrecordData   interface{}
}

//ReadPacket - чтение пакета данных протокола транспортного уровня
func ReadPacket(b []byte) (p Packet, err uint8) {
	i := 0
	p.ProtocolVersion = uint8(b[i]) //PRV
	i++
	if p.ProtocolVersion != 1 {
		err = 128 //EGTS_PC_UNS_PROTOCOL (неподдерживаемый протокол)
		return
	}
	p.SecurityKeyID = uint8(b[i]) //SKID
	i++
	flagBytes := uint16(b[i]) //flag
	i++
	p.PR = utils.BitField(flagBytes, 0, 1).(int)
	p.CMP = utils.BitField(flagBytes, 2).(bool)
	p.ENA = utils.BitField(flagBytes, 3, 4).(int)
	p.RTE = utils.BitField(flagBytes, 5).(bool)
	p.PRF = utils.BitField(flagBytes, 6, 7).(int)

	p.HeaderLength = uint8(b[i]) //HL
	i++
	p.HeaderEncoding = uint8(b[i]) //HE
	i++
	p.FrameDataLength = binary.LittleEndian.Uint16(b[i : i+2]) //FDL
	i += 2
	p.PacketID = binary.LittleEndian.Uint16(b[i : i+2]) //PID
	i += 2
	p.PacketType = uint8(b[i]) //PT
	i++
	if p.RTE {
		p.PeerAddress = binary.LittleEndian.Uint16(b[i : i+2]) //PRA
		i += 2
		p.RecipientAddress = binary.LittleEndian.Uint16(b[i : i+2]) //RCA
		i += 2
		p.TimeToLive = uint8(b[i]) //TTL
		i++
	}
	p.HeaderCheckSum = uint8(b[i]) //HCS
	i++
	p.ServicesFrameDataCheckSum = binary.LittleEndian.Uint16(b[uint16(p.HeaderLength)+p.FrameDataLength : uint16(p.HeaderLength)+p.FrameDataLength+2])
	if p.HeaderLength == 0 {
		err = 131 //EGTS_PC_INC_HEADERFORM (неверный формат заголовка)
		return
	} else if p.HeaderLength < 11 {
		err = 131 //EGTS_PC_INC_HEADERFORM (неверный формат заголовка)
		return
	}
	if len(b[p.HeaderLength:uint16(p.HeaderLength)+p.FrameDataLength]) != int(p.FrameDataLength) {
		err = 132 //EGTS_PC_INC_DATAFORM (неверный формат данных)
		return
	}
	crcData := crc.Crc(8, b[:p.HeaderLength-1])
	if int(crcData) != int(p.HeaderCheckSum) {
		err = 137 //EGTS_PC_HEADERCRC_ERROR (ошибка контрольной суммы заголовка)
		return
	}
	crcData = crc.Crc(16, b[p.HeaderLength:uint16(p.HeaderLength)+p.FrameDataLength])
	if int(crcData) != int(p.ServicesFrameDataCheckSum) {
		err = 138 //EGTS_PC_DATACRC_ERROR (ошибка контрольной суммы данных)
		return
	}
	p.ServicesFrameData, err = p.ReadServicesFrameData(b[p.HeaderLength : uint16(p.HeaderLength)+p.FrameDataLength])
	p.ResponseData = p.Response(b[p.HeaderLength:uint16(p.HeaderLength)+p.FrameDataLength], err, flagBytes)

	return
}

//ReadServicesFrameData - считывает данные поля SFRD - структура данных, зависящая от типа пакета и содержащая информацию
//Протокола уровня поддержки услуг
func (p *Packet) ReadServicesFrameData(b []byte) (sdf []ServicesFrameData, err uint8) {
	switch p.PacketType {
	case 0:
		//EGTS_PT_RESPONSE
		PacketID := binary.LittleEndian.Uint16(b[0:2])
		ProcessingResult := uint8(b[2])
		_ = PacketID
		err = ProcessingResult
		sdf, err = p.ReadSDR(b[3:])
	case 1:
		// EGTS_PT_APPDATA
		sdf, err = p.ReadSDR(b)
	case 2:
		//EGTS_PT_SIGNED_APPDATA (с электронной подписью)
	}

	return
}

//ReadSDR - считывает данные поля SFRD в формате EGTS_PT_APPDATA
func (p *Packet) ReadSDR(b []byte) (sdfs []ServicesFrameData, err uint8) {
	i := 0
	for {
		var sdf ServicesFrameData
		sdf.RecordLength = binary.LittleEndian.Uint16(b[i : i+2])
		i += 2
		sdf.RecordNumber = binary.LittleEndian.Uint16(b[i : i+2])
		i += 2
		flagBytes := uint16(b[i])
		i++
		sdf.OBFE = utils.BitField(flagBytes, 0).(bool)
		sdf.EVFE = utils.BitField(flagBytes, 1).(bool)
		sdf.TMFE = utils.BitField(flagBytes, 2).(bool)
		sdf.RPP = utils.BitField(flagBytes, 3, 4).(int)
		sdf.GRP = utils.BitField(flagBytes, 5).(bool)
		sdf.RSOD = utils.BitField(flagBytes, 6).(bool)
		sdf.SSOD = utils.BitField(flagBytes, 7).(bool)
		if sdf.OBFE {
			sdf.ObjectIdentifier = binary.LittleEndian.Uint32(b[i : i+4])
			i += 4
		}
		if sdf.EVFE {
			sdf.EventIdentifier = binary.LittleEndian.Uint32(b[i : i+4])
			i += 4
		}
		if sdf.TMFE {
			sdf.Time = binary.LittleEndian.Uint32(b[i : i+4])
			i += 4
		}
		sdf.SourceServiceType = uint8(b[i])
		i++
		sdf.RecipientServiceType = uint8(b[i])
		i++
		sdf.RecordData, err = ReadRecordData(b[i : i+int(sdf.RecordLength)])
		i += int(sdf.RecordLength)
		sdfs = append(sdfs, sdf)
		if i == len(b) {
			break
		}
	}
	return
}

//ReadRecordData - считывание отдельной записи Протокола Уровня Поддержки Услуг
func ReadRecordData(b []byte) (p RecordData, err uint8) {
	p.SubrecordType = uint8(b[0])
	p.SubrecordLength = binary.LittleEndian.Uint16(b[1:3])
	p.SubrecordData, err = p.ReadSubrecordData(b[3:])
	return
}

//ReadSubrecordData - опредение типа записи Протокола Уровня Поддержки Услуг
func (rd *RecordData) ReadSubrecordData(b []byte) (data interface{}, err uint8) {
	switch rd.SubrecordType {
	case 1:
		data = subrecord.ParseEgtsSrTermIdentity(b) // EGTS_SR_TERM_IDENTITY
	case 16:
		data = subrecord.ParseEgtsSrPosData(b) // EGTS_SR_POS_DATA
	case 17:
		data = subrecord.ParseEgtsSrExPosData(b) // EGTS_SR_EXT_POS_DATA
	case 18:
		data = subrecord.ParseEgtsSrAdSensorsData(b) // EGTS_SR_AD_SENSORS_DATA
	case 19:
		data = subrecord.ParseEgtsSrCountersData(b) // EGTS_SR_COUNTERS_DATA
	case 20:
		data = subrecord.ParseEgtsSrStateData(b) // EGTS_SR_STATE_DATA
	case 27:
		data = subrecord.ParseEgtsSrLiquidLevelSensor(b) //EGTS_SR_LIQUID_LEVEL_SENSOR
		if data == nil {
			err = 148
		}
	default:
		err = 148 //EGTS_PC_SRVC_NFOUND (Сервис не найден)
	}
	return
}

// Response - составляем ответ к полученному пакету с кодом обработки pr
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
