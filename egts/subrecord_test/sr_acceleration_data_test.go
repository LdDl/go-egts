package subrecord

import (
	"encoding/hex"
	"testing"

	"github.com/LdDl/go-egts/egts/subrecord"
)

var (
	SRAccelerationDataCheckIncome = []string{"39300a1a611eb822"}
)

func TestSRAccelerationDataDecoding(t *testing.T) {
	for i := range SRAccelerationDataCheckIncome {
		pkgHex := SRAccelerationDataCheckIncome[i]
		pkgBytes, err := hex.DecodeString(pkgHex)
		if err != nil {
			t.Errorf("Error: %s", err.Error())
		}
		subr := subrecord.SRAccelerationData{}
		err = subr.Decode(pkgBytes)
		if err != nil {
			t.Errorf("Error: %s", err.Error())
		}

		hexed, err := subr.Encode()
		if err != nil {
			t.Errorf("Error: %s", err.Error())
		}
		if hex.EncodeToString(hexed) != SRAccelerationDataCheckIncome[i] {
			t.Errorf("Have to be %s, but got %s", SRAccelerationDataCheckIncome[i], hex.EncodeToString(hexed))
		}
	}
}

var (
	SRAccelerationHeaderCheckIncome = []string{
		"02d854311539300a1a611eb82239300a1a611eb822",
	}
)

func TestSRAccelerationHeaderDecoding(t *testing.T) {
	for i := range SRAccelerationHeaderCheckIncome {
		pkgHex := SRAccelerationHeaderCheckIncome[i]
		pkgBytes, err := hex.DecodeString(pkgHex)
		if err != nil {
			t.Errorf("Error: %s", err.Error())
		}
		subr := subrecord.SRAccelerationHeader{}
		err = subr.Decode(pkgBytes)
		if err != nil {
			t.Errorf("Error: %s", err.Error())
		}

		hexed, err := subr.Encode()
		if err != nil {
			t.Errorf("Error: %s", err.Error())
		}
		if hex.EncodeToString(hexed) != SRAccelerationHeaderCheckIncome[i] {
			t.Errorf("Have to be %s, but got %s", SRAccelerationHeaderCheckIncome[i], hex.EncodeToString(hexed))
		}
	}
}
