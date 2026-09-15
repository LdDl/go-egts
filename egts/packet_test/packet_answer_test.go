package packet_test

import (
	"testing"

	"github.com/LdDl/go-egts/egts/packet"
	"github.com/LdDl/go-egts/egts/subrecord"
	"github.com/stretchr/testify/assert"
)

type answerServiceTestCase struct {
	source    uint8
	recipient uint8
	numbers   []uint16
}

func TestPrepareAnswerServices(t *testing.T) {
	services := []answerServiceTestCase{
		{source: 1, recipient: 1, numbers: []uint16{10}},
		{source: 2, recipient: 2, numbers: []uint16{20, 21}},
		{source: 1, recipient: 1, numbers: []uint16{11}},
		{source: 1, recipient: 2, numbers: []uint16{30}},
	}
	var incoming packet.ServicesFrameData
	for _, service := range services {
		for _, number := range service.numbers {
			incoming = append(incoming, &packet.ServiceDataRecord{
				RecordNumber:         number,
				SourceServiceType:    service.source,
				RecipientServiceType: service.recipient,
				RecordsData: packet.RecordsData{
					&packet.RecordData{SubrecordType: packet.ResultCode, SubrecordLength: 1, SubrecordData: &subrecord.SRResultCode{}},
				},
			})
		}
	}
	value := packet.Packet{PacketType: packet.EGTS_PT_APPDATA, PacketID: 0x1234, ServicesFrameData: &incoming}
	answer := value.PrepareAnswer(65535, 0x5678)
	assert.Equal(t, packet.EGTS_PT_RESPONSE, answer.PacketType)
	assert.Equal(t, uint16(0x5678), answer.PacketID)
	response, ok := answer.ServicesFrameData.(*packet.PTResponse)
	assert.True(t, ok)
	if !ok {
		return
	}
	assert.Equal(t, uint16(0x1234), response.ResponsePacketID)
	assert.Equal(t, packet.EGTS_PC_OK, response.ProcessingResult)
	result, ok := response.SDR.(*packet.ServicesFrameData)
	assert.True(t, ok)
	if !ok {
		return
	}
	expected := []answerServiceTestCase{
		{source: 1, recipient: 1, numbers: []uint16{10, 11}},
		{source: 2, recipient: 2, numbers: []uint16{20, 21}},
		{source: 2, recipient: 1, numbers: []uint16{30}},
	}
	if !assert.Len(t, *result, len(expected)) {
		return
	}
	for i, service := range expected {
		record := (*result)[i]
		assert.Equal(t, uint16(65535+i), record.RecordNumber)
		assert.Equal(t, service.source, record.SourceServiceType)
		assert.Equal(t, service.recipient, record.RecipientServiceType)
		assert.Equal(t, uint16(6*len(service.numbers)), record.RecordLength)
		if !assert.Len(t, record.RecordsData, len(service.numbers)) {
			return
		}
		for j, number := range service.numbers {
			assert.Equal(t, packet.RecordResponse, record.RecordsData[j].SubrecordType)
			assert.Equal(t, uint16(3), record.RecordsData[j].SubrecordLength)
			assert.Equal(t, &subrecord.SRRecordResponse{ConfirmedRecordNumber: number}, record.RecordsData[j].SubrecordData)
		}
	}
	assert.Equal(t, []uint16{10, 20, 21, 11, 30}, []uint16{
		incoming[0].RecordNumber, incoming[1].RecordNumber, incoming[2].RecordNumber,
		incoming[3].RecordNumber, incoming[4].RecordNumber,
	})
	encoded, err := answer.Encode()
	assert.NoError(t, err)
	decoded, err := packet.ReadPacket(encoded)
	assert.NoError(t, err)
	assert.Equal(t, answer.ServicesFrameData, decoded.ServicesFrameData)
}

func TestPrepareAnswerPacketError(t *testing.T) {
	incoming := packet.ServicesFrameData{
		&packet.ServiceDataRecord{RecordNumber: 42, SourceServiceType: packet.SERVICE_DATA, RecipientServiceType: packet.SERVICE_DATA},
	}
	for _, code := range []uint8{packet.EGTS_PC_IN_PROGRESS, packet.EGTS_PC_HEADERCRC_ERROR, packet.EGTS_PC_DATACRC_ERROR, packet.EGTS_PC_INC_DATAFORM, packet.EGTS_PC_IO_ERROR} {
		for _, data := range []packet.BytesData{nil, &incoming} {
			value := packet.Packet{PacketType: packet.EGTS_PT_APPDATA, PacketID: 17, ErrorCode: code, ServicesFrameData: data}
			answer := value.PrepareAnswer(1, 2)
			assert.Equal(t, packet.EGTS_PT_RESPONSE, answer.PacketType)
			assert.Equal(t, uint16(3), answer.FrameDataLength)
			assert.Equal(t, &packet.PTResponse{ResponsePacketID: 17, ProcessingResult: code}, answer.ServicesFrameData)
		}
	}
}

func TestPrepareAnswerEmptyData(t *testing.T) {
	var absent packet.ServicesFrameData
	empty := packet.ServicesFrameData{}
	for _, data := range []packet.BytesData{nil, &absent, &empty} {
		value := packet.Packet{PacketType: packet.EGTS_PT_APPDATA, PacketID: 17, ServicesFrameData: data}
		answer := value.PrepareAnswer(1, 2)
		assert.Equal(t, uint16(3), answer.FrameDataLength)
		assert.Equal(t, &packet.PTResponse{ResponsePacketID: 17}, answer.ServicesFrameData)
	}
}

func TestPrepareAnswerConfirmationOnly(t *testing.T) {
	incoming := packet.ServicesFrameData{
		&packet.ServiceDataRecord{
			RecordNumber: 42,
			RecordsData: packet.RecordsData{
				&packet.RecordData{SubrecordType: packet.RecordResponse, SubrecordLength: 3, SubrecordData: &subrecord.SRRecordResponse{ConfirmedRecordNumber: 1}},
			},
		},
	}
	value := packet.Packet{PacketType: packet.EGTS_PT_APPDATA, PacketID: 17, ServicesFrameData: &incoming}
	answer := value.PrepareAnswer(1, 2)
	assert.Equal(t, uint16(3), answer.FrameDataLength)
	assert.Equal(t, &packet.PTResponse{ResponsePacketID: 17}, answer.ServicesFrameData)
}

func TestPrepareAnswerInvalidData(t *testing.T) {
	nilRecord := packet.ServicesFrameData{nil}
	nilSubrecord := packet.ServicesFrameData{
		&packet.ServiceDataRecord{RecordsData: packet.RecordsData{nil}},
	}
	for _, data := range []packet.BytesData{&packet.PTResponse{}, (*packet.ServicesFrameData)(nil), &nilRecord, &nilSubrecord} {
		value := packet.Packet{PacketType: packet.EGTS_PT_APPDATA, ServicesFrameData: data}
		var answer packet.Packet
		assert.NotPanics(t, func() {
			answer = value.PrepareAnswer(1, 2)
		})
		assert.Equal(t, uint16(3), answer.FrameDataLength)
		assert.Equal(t, &packet.PTResponse{ProcessingResult: packet.EGTS_PC_INC_DATAFORM}, answer.ServicesFrameData)
	}
}
