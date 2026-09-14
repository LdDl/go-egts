package subrecord

import (
	"encoding/hex"
	"fmt"
	"io"
	"testing"

	"github.com/LdDl/go-egts/egts/subrecord"
	"github.com/stretchr/testify/assert"
)

var (
	SRStateDataCheckIncome = []string{"0286001404"}
)

func TestSRStateDataDecoding(t *testing.T) {
	for i := range SRStateDataCheckIncome {
		pkgHex := SRStateDataCheckIncome[i]
		pkgBytes, err := hex.DecodeString(pkgHex)
		if err != nil {
			t.Errorf("Error: %s", err.Error())
		}
		subr := subrecord.SRStateData{}
		err = subr.Decode(pkgBytes)
		if err != nil {
			t.Errorf("Error: %s", err.Error())
		}
		hexed, err := subr.Encode()
		if err != nil {
			t.Errorf("Error: %s", err.Error())
		}
		if hex.EncodeToString(hexed) != SRStateDataCheckIncome[i] {
			t.Errorf("Have to be %s, but got %s", SRStateDataCheckIncome[i], hex.EncodeToString(hexed))
		}
	}

	stateData := subrecord.SRStateData{}
	if err := stateData.Decode([]byte{0}); err == nil {
		t.Errorf("Error: expected error, but got nil")
	}
	if err := stateData.Decode([]byte{8}); err == nil {
		t.Errorf("Error: expected error, but got nil")
	}
}

type stateVoltageTestCase struct {
	data     byte
	expected float32
}

type stateEncodeErrorTestCase struct {
	name  string
	state uint8
	flags [3]string
}

func TestSRStateDataValues(t *testing.T) {
	names := []string{"Idle", "EraGlonass", "Active", "EmergencyCall", "EmergencyMonitor", "Testing", "Service", "FirmwareUpdate"}
	for state, name := range names {
		for flags := byte(0); flags < 8; flags++ {
			t.Run(fmt.Sprintf("%s/flags/%d", name, flags), func(t *testing.T) {
				data := []byte{byte(state), 134, 255, 20, flags}
				result := subrecord.SRStateData{}
				err := result.Decode(data)
				assert.NoError(t, err)
				assert.Equal(t, uint8(state), result.StateByte)
				assert.Equal(t, name, result.State)
				assert.Equal(t, fmt.Sprint((flags>>2)&1), result.NavigationModuleEnable)
				assert.Equal(t, fmt.Sprint((flags>>1)&1), result.InternalBatteryEnable)
				assert.Equal(t, fmt.Sprint(flags&1), result.BackupBatteryEnable)
				assert.Equal(t, uint8(134), result.MainPowerSourceVoltageByte)
				assert.Equal(t, uint8(255), result.BackupBatteryVoltageByte)
				assert.Equal(t, uint8(20), result.InternalBatteryVoltageByte)
				assert.InDelta(t, 13.4, result.MainPowerSourceVoltage, 0.00001)
				assert.InDelta(t, 25.5, result.BackupBatteryVoltage, 0.00001)
				assert.InDelta(t, 2, result.InternalBatteryVoltage, 0.00001)
				encoded, err := result.Encode()
				assert.NoError(t, err)
				assert.Equal(t, data, encoded)
				assert.Equal(t, uint16(5), result.Len())
			})
		}
	}
}

func TestSRStateDataVoltageLimits(t *testing.T) {
	cases := []stateVoltageTestCase{
		{data: 0, expected: 0},
		{data: 1, expected: 0.1},
		{data: 127, expected: 12.7},
		{data: 255, expected: 25.5},
	}
	for _, tc := range cases {
		t.Run(fmt.Sprint(tc.data), func(t *testing.T) {
			result := subrecord.SRStateData{}
			err := result.Decode([]byte{0, tc.data, tc.data, tc.data, 0})
			assert.NoError(t, err)
			assert.InDelta(t, tc.expected, result.MainPowerSourceVoltage, 0.00001)
			assert.InDelta(t, tc.expected, result.BackupBatteryVoltage, 0.00001)
			assert.InDelta(t, tc.expected, result.InternalBatteryVoltage, 0.00001)
		})
	}
}

func TestSRStateDataInvalidLength(t *testing.T) {
	data := []byte{2, 134, 255, 20, 7}
	for length := 0; length < len(data); length++ {
		t.Run(fmt.Sprintf("length/%d", length), func(t *testing.T) {
			result := subrecord.SRStateData{}
			err := result.Decode(data[:length])
			assert.ErrorIs(t, err, io.EOF)
			assert.Equal(t, subrecord.SRStateData{}, result)
		})
	}
	result := subrecord.SRStateData{}
	err := result.Decode(append(data, 0))
	assert.Error(t, err)
	assert.Equal(t, subrecord.SRStateData{}, result)
}

func TestSRStateDataRepeatedDecode(t *testing.T) {
	result := subrecord.SRStateData{}
	err := result.Decode([]byte{2, 134, 255, 20, 7})
	assert.NoError(t, err)
	previous := result
	err = result.Decode([]byte{1, 0, 0, 0})
	assert.ErrorIs(t, err, io.EOF)
	assert.Equal(t, previous, result)
	err = result.Decode([]byte{1, 0, 0, 0, 0, 0})
	assert.Error(t, err)
	assert.Equal(t, previous, result)
	for state := 8; state <= 255; state++ {
		err = result.Decode([]byte{byte(state), 0, 0, 0, 0})
		assert.Error(t, err, "state %d", state)
		assert.Equal(t, previous, result, "state %d", state)
	}
	err = result.Decode(make([]byte, 5))
	assert.NoError(t, err)
	expected := subrecord.SRStateData{
		State:                  "Idle",
		NavigationModuleEnable: "0",
		InternalBatteryEnable:  "0",
		BackupBatteryEnable:    "0",
	}
	assert.Equal(t, expected, result)
}

func TestSRStateDataEncodeInvalidValues(t *testing.T) {
	cases := []stateEncodeErrorTestCase{
		{name: "state 8", state: 8, flags: [3]string{"0", "0", "0"}},
		{name: "state 255", state: 255, flags: [3]string{"0", "0", "0"}},
		{name: "empty NMS", flags: [3]string{"", "0", "0"}},
		{name: "empty IBU", flags: [3]string{"0", "", "0"}},
		{name: "empty BBU", flags: [3]string{"0", "0", ""}},
		{name: "wide NMS", flags: [3]string{"00", "0", "0"}},
		{name: "wide IBU", flags: [3]string{"0", "00", "0"}},
		{name: "wide BBU", flags: [3]string{"0", "0", "00"}},
		{name: "invalid NMS", flags: [3]string{"2", "0", "0"}},
		{name: "invalid IBU", flags: [3]string{"0", "2", "0"}},
		{name: "invalid BBU", flags: [3]string{"0", "0", "2"}},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			value := subrecord.SRStateData{
				StateByte:              tc.state,
				NavigationModuleEnable: tc.flags[0],
				InternalBatteryEnable:  tc.flags[1],
				BackupBatteryEnable:    tc.flags[2],
			}
			previous := value
			encoded, err := value.Encode()
			assert.Error(t, err)
			assert.Nil(t, encoded)
			assert.Equal(t, previous, value)
		})
	}
}
