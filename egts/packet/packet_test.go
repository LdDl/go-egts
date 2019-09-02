package packet

import (
	"encoding/hex"
	"encoding/json"
	"fmt"
	"testing"
)

func TestBackPacket(t *testing.T) {
	egtsAuthHex := "0100020b0020000000014f1900000010010101160000000000523836363130343032393639303030380004417f"
	egtsAuth, _ := hex.DecodeString(egtsAuthHex)

	fmt.Println("Income", egtsAuth)
	parsedAuth, authCode := ReadPacket(egtsAuth)

	str, _ := json.Marshal(parsedAuth)
	fmt.Println(authCode, string(str))

	fmt.Println("Encode:")
	fmt.Println(parsedAuth.Encode(), hex.EncodeToString(parsedAuth.Encode()))
	t.Error(0)
}

func TestIncomingPacket(t *testing.T) {

	egtsAuthHex := "0100020b00b0000e0001779d001100977c8e5702241100009edd050f02021018009edd050f5fb4b49e8d7da2359b00009bc8550f030012010011040018110000120c000000070000000000000000001307000300000000000014050002860014041b0700400000fbff00001b0700000100000000001b0700010100000000001b07000201006c6300001b0700030100000000001b0700040100000000001b0700050100000000001b0700000200000000001b070001020000000000dc85"
	egtsAuth, _ := hex.DecodeString(egtsAuthHex)
	// egtsAuth := []byte{1, 0, 2, 11, 0, 176, 0, 14, 0, 1, 119, 157, 0, 17, 0, 151, 124, 142, 87, 2, 36, 17, 0, 0, 158, 221, 5, 15, 2, 2, 16, 24, 0, 158, 221, 5, 15, 95, 180, 180, 158, 141, 125, 162, 53, 155, 0, 0, 155, 200, 85, 15, 3, 0, 18, 1, 0, 17, 4, 0, 24, 17, 0, 0, 18, 12, 0, 0, 0, 7, 0, 0, 0, 0, 0, 0, 0, 0, 0, 19, 7, 0, 3, 0, 0, 0, 0, 0, 0, 20, 5, 0, 2, 134, 0, 20, 4, 27, 7, 0, 64, 0, 0, 251, 255, 0, 0, 27, 7, 0, 0, 1, 0, 0, 0, 0, 0, 27, 7, 0, 1, 1, 0, 0, 0, 0, 0, 27, 7, 0, 2, 1, 0, 108, 99, 0, 0, 27, 7, 0, 3, 1, 0, 0, 0, 0, 0, 27, 7, 0, 4, 1, 0, 0, 0, 0, 0, 27, 7, 0, 5, 1, 0, 0, 0, 0, 0, 27, 7, 0, 0, 2, 0, 0, 0, 0, 0, 27, 7, 0, 1, 2, 0, 0, 0, 0, 0, 220, 133}

	fmt.Println("Income", egtsAuth)

	parsedAuth, authCode := ReadPacket(egtsAuth)
	fmt.Println("auth code:", authCode)

	str, _ := json.Marshal(parsedAuth)
	fmt.Println(string(str))

	// hexd := "0100020b00b0000e0001779d001100977c8e5702241100009edd050f02021018009edd050f5fb4b49e8d7da2359b00009bc8550f030012010011040018110000120c000000070000000000000000001307000300000000000014050002860014041b0700400000fbff00001b0700000100000000001b0700010100000000001b07000201006c6300001b0700030100000000001b0700040100000000001b0700050100000000001b0700000200000000001b070001020000000000dc85"
	// heas, _ := hex.DecodeString(hexd)
	// log.Println(heas)
	t.Error(0)
}

func TestPTresponsePacket(t *testing.T) {

	egtsAuthHex := "0100030b001000000000b300000006000000580101000300000000d9d1"
	egtsAuth, _ := hex.DecodeString(egtsAuthHex)

	fmt.Println("Income", egtsAuth)
	parsedAuth, authCode := ReadPacket(egtsAuth)

	str, _ := json.Marshal(parsedAuth)
	fmt.Println(authCode, string(str))

	fmt.Println("Encode:")
	enc := parsedAuth.Encode()
	fmt.Println(enc, hex.EncodeToString(enc))
	t.Error(0)
}

func TestAuthResponsePacket(t *testing.T) {

	egtsAuthHex := "0100020b0020000000014f1900000010010101160000000000523836363130343032393639303030380004417f"
	egtsAuth, _ := hex.DecodeString(egtsAuthHex)

	fmt.Println("Income", egtsAuth)
	parsedAuth, authCode := ReadPacket(egtsAuth)

	str, _ := json.Marshal(parsedAuth)
	fmt.Println(authCode, string(str))

	fmt.Println("Encode:")
	enc := parsedAuth.Encode()
	fmt.Println(enc, hex.EncodeToString(enc))
	t.Error(0)
}
