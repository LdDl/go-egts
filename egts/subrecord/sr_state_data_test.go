package subrecord

import (
	"encoding/hex"
	"testing"
)

var (
	SRStateDataCheckIncome = "0286001404"
)

func TestSRStateDataDecoding(t *testing.T) {
	pkgHex := SRStateDataCheckIncome
	pkgBytes, err := hex.DecodeString(pkgHex)
	if err != nil {
		t.Errorf("Error: %s", err.Error())
	}
	subr := SRStateData{}
	subr.Decode(pkgBytes)
	subrCheck := SRStateData{
		State:                      "Active",
		StateByte:                  2,
		NavigationModuleEnable:     "1",
		InternalBatteryEnable:      "0",
		BackupBatteryEnable:        "0",
		MainPowerSourceVoltage:     13.400001,
		BackupBatteryVoltage:       0,
		InternalBatteryVoltage:     2,
		MainPowerSourceVoltageByte: 134,
		BackupBatteryVoltageByte:   0,
		InternalBatteryVoltageByte: 20,
	}
	if subr != subrCheck {
		t.Errorf("Have to be %v, but got %v", subrCheck, subr)
	}
}
