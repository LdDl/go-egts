package packet_test

import (
	"encoding/hex"
	"fmt"
	"testing"

	"github.com/LdDl/go-egts/egts/packet"
)

var (
	AllDataCheckIncome = []string{"0100020b00b0000e0001779d001100977c8e5702241100009edd050f02021018009edd050f5fb4b49e8d7da2359b00009bc8550f030012010011040018110000120c000000070000000000000000001307000300000000000014050002860014041b0700400000fbff00001b0700000100000000001b0700010100000000001b07000201006c6300001b0700030100000000001b0700040100000000001b0700050100000000001b0700000200000000001b070001020000000000dc85",
		"0100000b002300000001991800000001ef0000000202101500d2312b104fba3a9ed227bc35030000b200000000006a8d"}
	AllResponseDataCheckIncome = []string{"0100030b0010000e0000440e0000060000005802020003001100004364", "0100030b001000000000b3000000060000005802020003000000002ec1"}
)

func TestIncomingPacket(t *testing.T) {
	for i := range AllDataCheckIncome {
		pkgHex := AllDataCheckIncome[i]
		pkgBytes, err := hex.DecodeString(pkgHex)
		if err != nil {
			t.Errorf("Error: %s", err.Error())
		}
		pkg, err := packet.ReadPacket(pkgBytes)
		if err != nil {
			t.Errorf("Error: %s", err.Error())
		}
		ans := pkg.PrepareAnswer()
		hexedAns := ans.Encode()
		if hex.EncodeToString(hexedAns) != AllResponseDataCheckIncome[i] {
			t.Errorf("Have to be %s, but got %s", AllResponseDataCheckIncome[i], hex.EncodeToString(hexedAns))
		}
	}
}

var (
	AuthDataCheckIncome         = []string{"0100020b0020000000014f1900000010010101160000000000523836363130343032393639303030380004417f", "0100020b0020000000014f19000000100101011600000000005238363637393530333034383630353200045155"}
	AuthResponseDataCheckIncome = []string{"0100030b001000000000b300000006000000580101000300000000d9d1", "0100030b001000000000b300000006000000580101000300000000d9d1"}
)

func TestAuthResponsePacket(t *testing.T) {
	for i := range AuthDataCheckIncome {
		pkgHex := AuthDataCheckIncome[i]
		pkgBytes, err := hex.DecodeString(pkgHex)
		if err != nil {
			t.Errorf("Error: %s", err.Error())
		}
		pkg, err := packet.ReadPacket(pkgBytes)
		if err != nil {
			fmt.Println(pkgBytes)
			t.Errorf("Error: %s", err.Error())
		}
		ans := pkg.PrepareAnswer()
		hexedAns := ans.Encode()
		if hex.EncodeToString(hexedAns) != AuthResponseDataCheckIncome[i] {
			t.Errorf("Have to be %s, but got %s", AuthResponseDataCheckIncome[i], hex.EncodeToString(hexedAns))
		}
	}
}
