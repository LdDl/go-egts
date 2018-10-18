package main

import (
	"encoding/hex"
	"log"

	"github.com/LdDl/go-egts/egts/paсket"
)

func readFromBytes() {
	log.Println("Reading from bytes")
	maxBuffer := make([]byte, 65535)
	request := []byte{}
	data := maxBuffer[:len(request)]
	_ = data
	// TODO
}

func readFromStringHEX() {
	log.Println("Reading from string (HEX)")

	hexStrings := []string{
		"0100000B0022000100010c170001000196933831010101140096933831023836353930353032343632343131300397",             // auth
		"0100000B0022000100010c1700010001539ffc300101011400539ffc3002383631373835303037323332353736ccff",             // auth
		"0100000B0022000100010c170001000145e71631010101140045e7163102383633353931303236303238373831da25",             // auth
		"0100000B0022000100010c170001000143e71631010101140043e7163102383633353931303233373034393339c266",             // auth
		"0100000B0022000100010c1700010001549ffc300101011400549ffc3002383631373835303038333738303036a6d9",             // auth
		"0100000b0028000200016f1d00020001808c03000202101a0002b0d00f3aae5e9a1e7db24481cc017c00000000107800000000a7e0", // telematic data
		"0100000B002800030001291D00030001808c03000202101A008baed00f8c19609a8038a8448100000000000000107800000000c50d", // telematic data
	}

	for i := range hexStrings {
		maxBuffer := make([]byte, 0, 65535)
		maxBuffer, err := hex.DecodeString(hexStrings[i])
		if err != nil {
			log.Println("Skipping this iteration due the error:", err, hexStrings[i])
		}
		p, responseCode := packet.ReadPacket(maxBuffer)
		log.Println("Code error:", responseCode)
		log.Println(maxBuffer, p)
	}
}

func readFromFile() {
	log.Println("Reading from file")
	maxBuffer := make([]byte, 65535)
	_ = maxBuffer
	// TODO
}

func main() {
	readFromBytes()
	readFromStringHEX()
	readFromFile()
}
