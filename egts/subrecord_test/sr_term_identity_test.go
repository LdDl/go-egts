package subrecord_test

import (
	"encoding/hex"
	"testing"

	"github.com/LdDl/go-egts/egts/packet"
)

type IdentityCheck struct {
	Incoming  string
	Code      uint8
	Outcoming string
}

var (
	identities = []IdentityCheck{
		IdentityCheck{
			Incoming:  "0100020b0020000000014f1900000010010101160000000000523836363130343032393639303030380004417f",
			Code:      0,
			Outcoming: "0100030b001000000000b300000006000000580101000300000000d9d1",
		},
		IdentityCheck{
			Incoming:  "0100020b0020000000014f19000000100101011600000000005238363637393530333034383630353200045155",
			Code:      0,
			Outcoming: "0100030b001000000000b300000006000000580101000300000000d9d1",
		},
	}
)

func TestIdentity(t *testing.T) {
	for i := range identities {
		egtsAuthHex := identities[i].Incoming
		egtsAuth, err := hex.DecodeString(egtsAuthHex)
		if err != nil {
			t.Errorf("Error: %s", err.Error())
		}

		parsedAuth, err := packet.ReadPacket(egtsAuth)
		if err != nil {
			t.Errorf("Error: %s", err.Error())
		}
		resp := parsedAuth.PrepareAnswer()
		encodedResp := resp.Encode()
		hexResp := hex.EncodeToString(encodedResp)
		trueResp := identities[i].Outcoming
		if hexResp != trueResp {
			t.Errorf("Response should be %s, but got %s", trueResp, hexResp)
		}
	}
}
