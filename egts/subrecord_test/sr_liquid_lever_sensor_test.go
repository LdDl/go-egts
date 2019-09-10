package subrecord

import (
	"encoding/hex"
	"testing"

	"github.com/LdDl/go-egts/egts/subrecord"
)

var (
	SRLiquidLevelSensorCheckIncome = []string{"400000fbff0000", "00010000000000", "01010000000000", "0201006c630000", "03010000000000", "04010000000000", "05010000000000", "00020000000000", "01020000000000"}
)

func TestSRLiquidLevelSensorDecoding(t *testing.T) {
	for i := range SRLiquidLevelSensorCheckIncome {
		pkgHex := SRLiquidLevelSensorCheckIncome[i]
		pkgBytes, err := hex.DecodeString(pkgHex)
		if err != nil {
			t.Errorf("Error: %s", err.Error())
		}
		subr := subrecord.SRLiquidLevelSensor{}
		err = subr.Decode(pkgBytes)
		if err != nil {
			t.Errorf("Error: %s", err.Error())
		}
		hexed, err := subr.Encode()
		if err != nil {
			t.Errorf("Error: %s", err.Error())
		}
		if hex.EncodeToString(hexed) != SRLiquidLevelSensorCheckIncome[i] {
			t.Errorf("Have to be %s, but got %s", SRLiquidLevelSensorCheckIncome[i], hex.EncodeToString(hexed))
		}
	}
}
