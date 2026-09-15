package packet_test

import (
	"fmt"
	"testing"

	"github.com/LdDl/go-egts/egts/packet"
	"github.com/LdDl/go-egts/egts/subrecord"
	"github.com/stretchr/testify/assert"
)

type recordEncodeErrorTestCase struct {
	name  string
	value packet.RecordsData
}

type serviceEncodeFlagsTestCase struct {
	name  string
	flags [7]string
}

type nilEncodeTestCase struct {
	name  string
	value packet.BytesData
}

func TestRecordsDataEncodeInvalid(t *testing.T) {
	cases := []recordEncodeErrorTestCase{
		{name: "nil record", value: packet.RecordsData{nil}},
		{name: "nil body", value: packet.RecordsData{&packet.RecordData{SubrecordType: packet.ResultCode, SubrecordLength: 1}}},
		{name: "short SRL", value: packet.RecordsData{&packet.RecordData{SubrecordType: packet.RecordResponse, SubrecordLength: 2, SubrecordData: &subrecord.SRRecordResponse{}}}},
		{name: "long SRL", value: packet.RecordsData{&packet.RecordData{SubrecordType: packet.RecordResponse, SubrecordLength: 4, SubrecordData: &subrecord.SRRecordResponse{}}}},
		{name: "zero SRL", value: packet.RecordsData{&packet.RecordData{SubrecordType: packet.ResultCode, SubrecordData: &subrecord.SRResultCode{}}}},
		{
			name: "invalid after valid record",
			value: packet.RecordsData{
				&packet.RecordData{SubrecordType: packet.ResultCode, SubrecordLength: 1, SubrecordData: &subrecord.SRResultCode{}},
				nil,
			},
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			encoded, err := tc.value.Encode()
			assert.Error(t, err)
			assert.Nil(t, encoded)
			assert.Zero(t, tc.value.Len())
		})
	}
}

func TestServicesFrameDataEncodeFlags(t *testing.T) {
	cases := []serviceEncodeFlagsTestCase{
		{name: "empty SSOD", flags: [7]string{"", "0", "0", "00", "0", "0", "0"}},
		{name: "wide SSOD", flags: [7]string{"00", "0", "0", "00", "0", "0", "0"}},
		{name: "invalid SSOD", flags: [7]string{"2", "0", "0", "00", "0", "0", "0"}},
		{name: "empty RSOD", flags: [7]string{"0", "", "0", "00", "0", "0", "0"}},
		{name: "wide RSOD", flags: [7]string{"0", "00", "0", "00", "0", "0", "0"}},
		{name: "invalid RSOD", flags: [7]string{"0", "2", "0", "00", "0", "0", "0"}},
		{name: "empty GRP", flags: [7]string{"0", "0", "", "00", "0", "0", "0"}},
		{name: "wide GRP", flags: [7]string{"0", "0", "00", "00", "0", "0", "0"}},
		{name: "invalid GRP", flags: [7]string{"0", "0", "2", "00", "0", "0", "0"}},
		{name: "short RPP", flags: [7]string{"0", "0", "0", "0", "0", "0", "0"}},
		{name: "wide RPP", flags: [7]string{"0", "0", "0", "000", "0", "0", "0"}},
		{name: "invalid RPP", flags: [7]string{"0", "0", "0", "02", "0", "0", "0"}},
		{name: "empty TMFE", flags: [7]string{"0", "0", "0", "00", "", "0", "0"}},
		{name: "wide TMFE", flags: [7]string{"0", "0", "0", "00", "00", "0", "0"}},
		{name: "invalid TMFE", flags: [7]string{"0", "0", "0", "00", "2", "0", "0"}},
		{name: "empty EVFE", flags: [7]string{"0", "0", "0", "00", "0", "", "0"}},
		{name: "wide EVFE", flags: [7]string{"0", "0", "0", "00", "0", "00", "0"}},
		{name: "invalid EVFE", flags: [7]string{"0", "0", "0", "00", "0", "2", "0"}},
		{name: "empty OBFE", flags: [7]string{"0", "0", "0", "00", "0", "0", ""}},
		{name: "wide OBFE", flags: [7]string{"0", "0", "0", "00", "0", "0", "00"}},
		{name: "invalid OBFE", flags: [7]string{"0", "0", "0", "00", "0", "0", "2"}},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			value := packet.ServicesFrameData{
				&packet.ServiceDataRecord{
					RecordLength: 4,
					SSOD:         tc.flags[0], RSOD: tc.flags[1], GRP: tc.flags[2], RPP: tc.flags[3],
					TMFE: tc.flags[4], EVFE: tc.flags[5], OBFE: tc.flags[6],
					RecordsData: packet.RecordsData{
						&packet.RecordData{SubrecordType: packet.ResultCode, SubrecordLength: 1, SubrecordData: &subrecord.SRResultCode{}},
					},
				},
			}
			encoded, err := value.Encode()
			assert.Error(t, err)
			assert.Nil(t, encoded)
			assert.Zero(t, value.Len())
		})
	}
}

func TestServicesFrameDataEncodeLength(t *testing.T) {
	for _, length := range []uint16{0, 3, 5, 65535} {
		t.Run(fmt.Sprint(length), func(t *testing.T) {
			value := packet.ServicesFrameData{
				&packet.ServiceDataRecord{
					RecordLength: length,
					SSOD:         "0", RSOD: "0", GRP: "0", RPP: "00", TMFE: "0", EVFE: "0", OBFE: "0",
					RecordsData: packet.RecordsData{
						&packet.RecordData{SubrecordType: packet.ResultCode, SubrecordLength: 1, SubrecordData: &subrecord.SRResultCode{}},
					},
				},
			}
			encoded, err := value.Encode()
			assert.Error(t, err)
			assert.Nil(t, encoded)
		})
	}
	value := packet.ServicesFrameData{nil}
	encoded, err := value.Encode()
	assert.Error(t, err)
	assert.Nil(t, encoded)
}

func TestEncodeNilValues(t *testing.T) {
	cases := []nilEncodeTestCase{
		{name: "records", value: (*packet.RecordsData)(nil)},
		{name: "services", value: (*packet.ServicesFrameData)(nil)},
		{name: "response", value: (*packet.PTResponse)(nil)},
		{name: "record response", value: (*subrecord.SRRecordResponse)(nil)},
		{name: "result code", value: (*subrecord.SRResultCode)(nil)},
		{name: "identity", value: (*subrecord.SRTermIdentity)(nil)},
		{name: "position", value: (*subrecord.SRPosData)(nil)},
		{name: "extended position", value: (*subrecord.SRExPosDataRecord)(nil)},
		{name: "sensors", value: (*subrecord.SRAdSensorsData)(nil)},
		{name: "counters", value: (*subrecord.SRCountersData)(nil)},
		{name: "state", value: (*subrecord.SRStateData)(nil)},
		{name: "acceleration", value: (*subrecord.SRAccelerationData)(nil)},
		{name: "acceleration header", value: (*subrecord.SRAccelerationHeader)(nil)},
		{name: "liquid", value: (*subrecord.SRLiquidLevelSensor)(nil)},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			var encoded []byte
			var err error
			assert.NotPanics(t, func() {
				encoded, err = tc.value.Encode()
			})
			assert.Error(t, err)
			assert.Nil(t, encoded)
			assert.Zero(t, tc.value.Len())
		})
	}
}

func TestRecordsDataEncodeOverflow(t *testing.T) {
	var value packet.RecordsData
	for i := 0; i < 16384; i++ {
		value = append(value, &packet.RecordData{SubrecordType: packet.ResultCode, SubrecordLength: 1, SubrecordData: &subrecord.SRResultCode{}})
	}
	encoded, err := value.Encode()
	assert.Error(t, err)
	assert.Nil(t, encoded)
	assert.Zero(t, value.Len())
}
