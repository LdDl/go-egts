package subrecord

import (
	"encoding/hex"
	"testing"

	"github.com/LdDl/go-egts/egts/subrecord"
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
		subr.Decode(pkgBytes)
		hexed, err := subr.Encode()
		if err != nil {
			t.Errorf("Error: %s", err.Error())
		}
		if hex.EncodeToString(hexed) != SRStateDataCheckIncome[i] {
			t.Errorf("Have to be %s, but got %s", SRStateDataCheckIncome[i], string(hexed))
		}
	}
}
