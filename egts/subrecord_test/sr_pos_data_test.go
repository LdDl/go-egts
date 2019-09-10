package subrecord

import (
	"encoding/hex"
	"testing"

	"github.com/LdDl/go-egts/egts/subrecord"
)

var (
	SRPosDataCheckIncome = []string{"9edd050f5fb4b49e8d7da2359b00009bc8550f0300120100"}
)

func TestSRPosDataDecoding(t *testing.T) {
	for i := range SRPosDataCheckIncome {
		pkgHex := SRPosDataCheckIncome[i]
		pkgBytes, err := hex.DecodeString(pkgHex)
		if err != nil {
			t.Errorf("Error: %s", err.Error())
		}
		subr := subrecord.SRPosData{}
		err = subr.Decode(pkgBytes)
		if err != nil {
			t.Errorf("Error: %s", err.Error())
		}
		hexed, err := subr.Encode()
		if err != nil {
			t.Errorf("Error: %s", err.Error())
		}
		if hex.EncodeToString(hexed) != SRPosDataCheckIncome[i] {
			t.Errorf("Have to be %s, but got %s", SRPosDataCheckIncome[i], hex.EncodeToString(hexed))
		}
	}
}
