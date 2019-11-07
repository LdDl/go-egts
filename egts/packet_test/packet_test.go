package packet_test

import (
	"encoding/hex"
	"fmt"
	"testing"

	"github.com/LdDl/go-egts/egts/packet"
)

var (
	AllDataCheckIncome = []string{
		"0100000b002300000001991800000001ef0000000202101500d2312b104fba3a9ed227bc35030000b200000000006a8d",
	}
	AllResponseDataCheckIncome = []string{"0100030b001000000000b3000000060000005802020003000000002ec1"}
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
		ans := pkg.PrepareAnswer(0, pkg.PacketID)
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
		ans := pkg.PrepareAnswer(0, pkg.PacketID)
		hexedAns := ans.Encode()
		if hex.EncodeToString(hexedAns) != AuthResponseDataCheckIncome[i] {
			t.Errorf("Have to be %s, but got %s", AuthResponseDataCheckIncome[i], hex.EncodeToString(hexedAns))
		}
	}
}
