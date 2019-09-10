package subrecord

import (
	"encoding/hex"
	"testing"

	"github.com/LdDl/go-egts/egts/subrecord"
)

var (
	SRCountersDataCheckIncome = []string{"03000000000000"}
)

func TestSRCountersDataDecoding(t *testing.T) {
	for i := range SRCountersDataCheckIncome {
		pkgHex := SRCountersDataCheckIncome[i]
		pkgBytes, err := hex.DecodeString(pkgHex)
		if err != nil {
			t.Errorf("Error: %s", err.Error())
		}
		subr := subrecord.SRCountersData{}
		err = subr.Decode(pkgBytes)
		if err != nil {
			t.Errorf("Error: %s", err.Error())
		}
		hexed, err := subr.Encode()
		if err != nil {
			t.Errorf("Error: %s", err.Error())
		}
		if hex.EncodeToString(hexed) != SRCountersDataCheckIncome[i] {
			t.Errorf("Have to be %s, but got %s", SRCountersDataCheckIncome[i], hex.EncodeToString(hexed))
		}
	}
}
