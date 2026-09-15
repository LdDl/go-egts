package packet_test

import (
	"encoding/hex"
	"strconv"
	"testing"

	"github.com/LdDl/go-egts/egts/packet"
	"github.com/LdDl/go-egts/egts/subrecord"
	"github.com/stretchr/testify/assert"
)

type packetEncodeHeaderTestCase struct {
	name           string
	version        uint8
	flags          [5]string
	headerLength   uint8
	headerEncoding uint8
	packetType     uint8
	frameLength    uint16
	data           packet.BytesData
}

type packetEncodeSizeTestCase struct {
	name         string
	headerLength uint8
	route        string
	count        int
	rawLength    int
	valid        bool
}

func TestPacketEncodeHeaders(t *testing.T) {
	flags := [5]string{"00", "0", "00", "0", "00"}
	cases := []packetEncodeHeaderTestCase{
		{name: "zero packet"},
		{name: "version", version: 2, flags: flags, headerLength: 11, packetType: 1},
		{name: "prefix", version: 1, flags: [5]string{"01", "0", "00", "0", "00"}, headerLength: 11, packetType: 1},
		{name: "encoding", version: 1, flags: flags, headerLength: 11, headerEncoding: 1, packetType: 1},
		{name: "encryption", version: 1, flags: [5]string{"00", "0", "01", "0", "00"}, headerLength: 11, packetType: 1},
		{name: "compression", version: 1, flags: [5]string{"00", "0", "00", "1", "00"}, headerLength: 11, packetType: 1},
		{name: "empty route", version: 1, flags: [5]string{"00", "", "00", "0", "00"}, headerLength: 11, packetType: 1},
		{name: "wide route", version: 1, flags: [5]string{"00", "00", "00", "0", "00"}, headerLength: 11, packetType: 1},
		{name: "invalid route", version: 1, flags: [5]string{"00", "2", "00", "0", "00"}, headerLength: 11, packetType: 1},
		{name: "empty priority", version: 1, flags: [5]string{"00", "0", "00", "0", ""}, headerLength: 11, packetType: 1},
		{name: "wide priority", version: 1, flags: [5]string{"00", "0", "00", "0", "000"}, headerLength: 11, packetType: 1},
		{name: "invalid priority", version: 1, flags: [5]string{"00", "0", "00", "0", "02"}, headerLength: 11, packetType: 1},
		{name: "short header", version: 1, flags: flags, headerLength: 10, packetType: 1},
		{name: "route fields without flag", version: 1, flags: flags, headerLength: 16, packetType: 1},
		{name: "route flag without fields", version: 1, flags: [5]string{"00", "1", "00", "0", "00"}, headerLength: 11, packetType: 1},
		{name: "signed packet", version: 1, flags: flags, headerLength: 11, packetType: 2},
		{name: "unknown packet", version: 1, flags: flags, headerLength: 11, packetType: 255},
		{name: "missing frame", version: 1, flags: flags, headerLength: 11, packetType: 1, frameLength: 1},
		{name: "response without body", version: 1, flags: flags, headerLength: 11, packetType: 0},
		{name: "response with appdata body", version: 1, flags: flags, headerLength: 11, packetType: 0, data: &packet.ServicesFrameData{}},
		{name: "appdata with response body", version: 1, flags: flags, headerLength: 11, packetType: 1, data: &packet.PTResponse{}},
		{name: "nil appdata pointer", version: 1, flags: flags, headerLength: 11, packetType: 1, data: (*packet.ServicesFrameData)(nil)},
		{name: "nil response pointer", version: 1, flags: flags, headerLength: 11, packetType: 0, data: (*packet.PTResponse)(nil)},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			value := packet.Packet{
				ProtocolVersion:   tc.version,
				PRF:               tc.flags[0],
				RTE:               tc.flags[1],
				ENA:               tc.flags[2],
				CMP:               tc.flags[3],
				PR:                tc.flags[4],
				HeaderLength:      tc.headerLength,
				HeaderEncoding:    tc.headerEncoding,
				PacketType:        tc.packetType,
				FrameDataLength:   tc.frameLength,
				ServicesFrameData: tc.data,
			}
			previous := value
			encoded, err := value.Encode()
			assert.Error(t, err)
			assert.Nil(t, encoded)
			assert.Equal(t, previous, value)
		})
	}
	var absent *packet.Packet
	encoded, err := absent.Encode()
	assert.Error(t, err)
	assert.Nil(t, encoded)
}

func TestPacketEncodeRoundTrip(t *testing.T) {
	for _, encoded := range []string{AllDataCheckIncome[0], AllResponseDataCheckIncome[0], AuthDataCheckIncome[0]} {
		data, err := hex.DecodeString(encoded)
		assert.NoError(t, err)
		value, err := packet.ReadPacket(data)
		assert.NoError(t, err)
		if err != nil {
			return
		}
		previous := value
		result, err := value.Encode()
		assert.NoError(t, err)
		assert.Equal(t, data, result)
		assert.Equal(t, previous, value)
	}
	for _, priority := range []string{"00", "01", "10", "11"} {
		for _, route := range []string{"0", "1"} {
			value := packet.Packet{
				ProtocolVersion: 1, PRF: "00", RTE: route, ENA: "00", CMP: "0", PR: priority,
				HeaderLength: 11, PacketType: packet.EGTS_PT_APPDATA, PacketID: 65535,
			}
			if route == "1" {
				value.HeaderLength = 16
				value.PeerAddress = 0x1234
				value.RecipientAddress = 0xabcd
				value.TimeToLive = 3
			}
			encoded, err := value.Encode()
			assert.NoError(t, err)
			assert.Len(t, encoded, int(value.HeaderLength))
			decoded, err := packet.ReadPacket(encoded)
			assert.NoError(t, err)
			assert.Equal(t, priority, decoded.PR)
			assert.Equal(t, route, decoded.RTE)
			assert.Equal(t, value.PeerAddress, decoded.PeerAddress)
			assert.Equal(t, value.RecipientAddress, decoded.RecipientAddress)
			assert.Equal(t, value.TimeToLive, decoded.TimeToLive)
		}
	}
}

func TestPacketEncodeNestedError(t *testing.T) {
	liquid := &subrecord.SRLiquidLevelSensor{
		RawbFlag: "0", LiquidLevelSensorErrorFlag: "0", LiquidLevelSensorValueUnit: "02",
	}
	records := packet.RecordsData{
		&packet.RecordData{SubrecordType: packet.LiquidLevelSensor, SubrecordLength: 7, SubrecordData: liquid},
	}
	services := packet.ServicesFrameData{
		&packet.ServiceDataRecord{
			RecordLength: 10, RecordNumber: 42,
			SSOD: "0", RSOD: "1", GRP: "0", RPP: "00", TMFE: "0", EVFE: "0", OBFE: "0",
			SourceServiceType: packet.SERVICE_DATA, RecipientServiceType: packet.SERVICE_DATA,
			RecordsData: records,
		},
	}
	value := packet.Packet{
		ProtocolVersion: 1, PRF: "00", RTE: "0", ENA: "00", CMP: "0", PR: "00",
		HeaderLength: 11, PacketType: packet.EGTS_PT_APPDATA,
		FrameDataLength: 17, ServicesFrameData: &services,
	}
	encoded, err := value.Encode()
	assert.Error(t, err)
	assert.Nil(t, encoded)
	if err == nil {
		return
	}
	var numberError *strconv.NumError
	assert.ErrorAs(t, err, &numberError)
	assert.Contains(t, err.Error(), "RN 42")
	assert.Contains(t, err.Error(), "SRT 27")

	liquid.LiquidLevelSensorValueUnit = "00"
	encoded, err = value.Encode()
	assert.NoError(t, err)
	assert.Len(t, encoded, 30)
	value.FrameDataLength = 16
	encoded, err = value.Encode()
	assert.Error(t, err)
	assert.Nil(t, encoded)
	value.FrameDataLength = 18
	encoded, err = value.Encode()
	assert.Error(t, err)
	assert.Nil(t, encoded)
}

func TestPacketEncodeSizeLimit(t *testing.T) {
	cases := []packetEncodeSizeTestCase{
		{name: "maximum", headerLength: 11, route: "0", count: 16376, rawLength: 5, valid: true},
		{name: "too large", headerLength: 11, route: "0", count: 16376, rawLength: 6},
		{name: "routed maximum", headerLength: 16, route: "1", count: 16375, rawLength: 4, valid: true},
		{name: "routed too large", headerLength: 16, route: "1", count: 16375, rawLength: 5},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			var records packet.RecordsData
			for i := 0; i < tc.count; i++ {
				records = append(records, &packet.RecordData{
					SubrecordType: packet.ResultCode, SubrecordLength: 1, SubrecordData: &subrecord.SRResultCode{},
				})
			}
			records = append(records, &packet.RecordData{
				SubrecordType: packet.LiquidLevelSensor, SubrecordLength: uint16(tc.rawLength + 3),
				SubrecordData: &subrecord.SRLiquidLevelSensor{
					RawbFlag: "1", LiquidLevelSensorValueUnit: "00", LiquidLevelSensorErrorFlag: "0", RawData: make([]byte, tc.rawLength),
				},
			})
			services := packet.ServicesFrameData{
				&packet.ServiceDataRecord{
					RecordLength: uint16(tc.count*4 + tc.rawLength + 6),
					SSOD:         "0", RSOD: "1", GRP: "0", RPP: "00", TMFE: "0", EVFE: "0", OBFE: "0",
					SourceServiceType: 1, RecipientServiceType: 1, RecordsData: records,
				},
			}
			value := packet.Packet{
				ProtocolVersion: 1, PRF: "00", RTE: tc.route, ENA: "00", CMP: "0", PR: "00",
				HeaderLength: tc.headerLength, PacketType: packet.EGTS_PT_APPDATA,
				FrameDataLength: uint16(tc.count*4 + tc.rawLength + 13), ServicesFrameData: &services,
			}
			encoded, err := value.Encode()
			if !tc.valid {
				assert.Error(t, err)
				assert.Nil(t, encoded)
				return
			}
			assert.NoError(t, err)
			assert.Len(t, encoded, 65535)
			decoded, err := packet.ReadPacket(encoded)
			assert.NoError(t, err)
			assert.Equal(t, value.FrameDataLength, decoded.FrameDataLength)
		})
	}
}
