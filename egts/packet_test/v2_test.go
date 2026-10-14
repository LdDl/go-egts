package packet_test

import (
	"encoding/hex"
	"testing"

	"github.com/LdDl/go-egts/egts/packet"
	"github.com/LdDl/go-egts/egts/subrecord"
	"github.com/stretchr/testify/assert"
)

const (
	v2AuthHex     = "0100000b006a000a00016b53006400071122334455667788785634122a0000000101010b008877665544332211003032031e003132333435363738394142434445464748010000000300000045585452410c2100004131323342430000000000000000000000000000000000000000000000000000282c"
	v2PositionHex = "0100000b003e000b00018f27006500071122334455667788785634122a00000002021024002a000000ffffffffffffffff837bc02c563412a5042143057856341285ff57e80300feffc582"
)

type v2PacketTestCase struct {
	name   string
	rawHex string
}

func TestV2SpecificationPackets(t *testing.T) {
	for _, tc := range []v2PacketTestCase{{name: "auth", rawHex: v2AuthHex}, {name: "position", rawHex: v2PositionHex}} {
		t.Run(tc.name, func(t *testing.T) {
			raw, err := hex.DecodeString(tc.rawHex)
			if !assert.NoError(t, err) {
				return
			}
			p, err := packet.ReadPacketVersion(raw, packet.ServiceVersion2)
			if !assert.NoError(t, err) {
				return
			}
			assert.Equal(t, uint8(1), p.ProtocolVersion)
			assert.Equal(t, packet.ServiceVersion2, p.ServiceVersion)
			records, ok := p.ServicesFrameData.(*packet.ServicesFrameData)
			if !assert.True(t, ok) {
				return
			}
			if !assert.Len(t, *records, 1) {
				return
			}
			r := (*records)[0]
			assert.Equal(t, uint64(0x8877665544332211), r.ObjectIdentifierV2)
			assert.Zero(t, r.ObjectIdentifier)
			assert.Equal(t, uint32(0x12345678), r.EventIdentifier)
			assert.Equal(t, uint32(42), r.Time)
			if tc.name == "auth" {
				if !assert.Len(t, r.RecordsData, 3) {
					return
				}
				term, ok := r.RecordsData[0].SubrecordData.(*subrecord.SRTermIdentityV2)
				if !assert.True(t, ok) {
					return
				}
				assert.Equal(t, uint64(0x1122334455667788), term.TerminalIdentifier)
				assert.Equal(t, "02", term.ServiceSupportLevelProtocolVersion)
				vehicle, ok := r.RecordsData[1].SubrecordData.(*subrecord.SRVehicleDataV2)
				if !assert.True(t, ok) {
					return
				}
				assert.Equal(t, "123456789ABCDEFGH", vehicle.VINLow)
				assert.Equal(t, "EXTRA", vehicle.VINHigh)
			} else {
				pos, ok := r.RecordsData[0].SubrecordData.(*subrecord.SRPosDataV2)
				if !assert.True(t, ok) {
					return
				}
				assert.Equal(t, 90.0, pos.Latitude)
				assert.Equal(t, 180.0, pos.Longitude)
				assert.Equal(t, uint16(123), pos.SpeedTenths)
				assert.Equal(t, uint16(300), pos.DirectionValue)
				assert.Equal(t, [3]byte{0x21, 0x43, 0x05}, pos.NetworkIdentifier)
				assert.Equal(t, uint32(0x12345678), pos.LocalAreaCode)
				assert.Equal(t, int16(-123), pos.CellIdentifier)
				assert.Equal(t, uint8(87), pos.SignalStrength)
				assert.Equal(t, uint32(1000), pos.Altitude)
				assert.Equal(t, int16(-2), pos.SourceData)
			}
			encoded, err := p.Encode()
			if !assert.NoError(t, err) {
				return
			}
			assert.Equal(t, raw, encoded)
			for _, answer := range []packet.Packet{p.PrepareAnswer(200, 201), p.PrepareSRResultCode(0, 202, 203)} {
				raw, err := answer.Encode()
				if !assert.NoError(t, err) {
					return
				}
				decoded, err := packet.ReadPacketVersion(raw, packet.ServiceVersion2)
				if !assert.NoError(t, err) {
					return
				}
				assert.Equal(t, uint8(1), decoded.ProtocolVersion)
				assert.Equal(t, packet.ServiceVersion2, decoded.ServiceVersion)
			}
			_, err = packet.ReadPacket(raw)
			assert.Error(t, err)
			p.ServiceVersion = 1
			_, err = p.Encode()
			assert.ErrorContains(t, err, "versions differ")
		})
	}
}

func TestV2RejectsMixedFormatsAndUnknownVersions(t *testing.T) {
	raw, err := hex.DecodeString(v2AuthHex)
	if !assert.NoError(t, err) {
		return
	}
	_, err = packet.ReadPacketVersion(raw, 3)
	assert.Error(t, err)
	p, err := packet.ReadPacketVersion(raw, 2)
	if !assert.NoError(t, err) {
		return
	}
	records, ok := p.ServicesFrameData.(*packet.ServicesFrameData)
	if !assert.True(t, ok) {
		return
	}
	r := (*records)[0]
	r.ObjectIdentifier = 1
	_, err = p.Encode()
	assert.Error(t, err)
	r.ObjectIdentifier = 0
	r.RecordsData[0].SubrecordData = &subrecord.SRTermIdentity{}
	_, err = p.Encode()
	assert.ErrorContains(t, err, "V1 subrecord")
}

func TestV2TruncatedRecordsDoNotReplaceReceiver(t *testing.T) {
	raw, err := hex.DecodeString(v2AuthHex)
	if !assert.NoError(t, err) {
		return
	}
	for n := 1; n < len(raw)-13; n++ {
		original := packet.ServicesFrameData{&packet.ServiceDataRecord{RecordNumber: 999}}
		err := original.DecodeVersion(raw[11:11+n], 2)
		assert.Error(t, err, "length %d", n)
		if !assert.Len(t, original, 1) {
			return
		}
		assert.Equal(t, uint16(999), original[0].RecordNumber)
	}
}
