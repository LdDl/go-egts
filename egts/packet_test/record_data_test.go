package packet_test

import (
	"encoding/hex"
	"testing"

	"github.com/LdDl/go-egts/egts/packet"
)

var (
	RecordsDataCheckIncome = []string{"1018009edd050f5fb4b49e8d7da2359b00009bc8550f030012010011040018110000120c000000070000000000000000001307000300000000000014050002860014041b0700400000fbff00001b0700000100000000001b0700010100000000001b07000201006c6300001b0700030100000000001b0700040100000000001b0700050100000000001b0700000200000000001b070001020000000000"}
)

func TestRecordsDataDecoding(t *testing.T) {
	for i := range RecordsDataCheckIncome {
		pkgHex := RecordsDataCheckIncome[i]
		pkgBytes, err := hex.DecodeString(pkgHex)
		if err != nil {
			t.Errorf("Error: %s", err.Error())
		}
		subr := packet.RecordsData{}
		err = subr.Decode(pkgBytes)
		if err != nil {
			t.Errorf("Error: %s", err.Error())
		}
		hexed, err := subr.Encode()
		if err != nil {
			t.Errorf("Error: %s", err.Error())
		}
		if hex.EncodeToString(hexed) != RecordsDataCheckIncome[i] {
			t.Errorf("Have to be %s, but got %s", RecordsDataCheckIncome[i], hex.EncodeToString(hexed))
		}
	}
}
