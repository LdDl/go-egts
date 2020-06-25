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
