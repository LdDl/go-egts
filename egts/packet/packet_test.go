package packet

import (
	"encoding/hex"
	"fmt"
	"io/ioutil"
	"testing"
)

var (
	hexStringsAuth = []string{
		"0100020b0020000000014f1900000010010101160000000000523836363130343032393639303030380004417f",
		"0100000B0022000100010C1700000004E639211201010114000000000002333538343830303837333739343130701C",
		"0100000B0022000100010c170001000196933831010101140096933831023836353930353032343632343131300397",
		"0100000B0022000100010c1700010001539ffc300101011400539ffc3002383631373835303037323332353736ccff",
		"0100000B0022000100010c170001000145e71631010101140045e7163102383633353931303236303238373831da25",
		"0100000B0022000100010c170001000143e71631010101140043e7163102383633353931303233373034393339c266",
		"0100000B0022000100010c1700010001549ffc300101011400549ffc3002383631373835303038333738303036a6d9",
	}
	hexStringsData = []string{
		"0100000b0028000200016f1d00020001808c03000202101a0002b0d00f3aae5e9a1e7db24481cc017c00000000107800000000a7e0",
		"0100000B002800030001291D00030001808c03000202101A008baed00f8c19609a8038a8448100000000000000107800000000c50d",
	}

	bytesAuth = [][]byte{
		[]byte{1, 0, 2, 11, 0, 32, 0, 4, 0, 1, 102, 25, 0, 0, 0, 16, 1, 1, 1, 22, 0, 99, 0, 0, 0, 82, 56, 54, 54, 55, 57, 53, 48, 51, 51, 49, 57, 57, 51, 50, 50, 0, 4, 169, 251},
		[]byte{1, 0, 0, 11, 0, 34, 0, 1, 0, 1, 12, 23, 0, 1, 0, 1, 69, 231, 22, 49, 1, 1, 1, 20, 0, 69, 231, 22, 49, 2, 56, 54, 51, 53, 57, 49, 48, 50, 54, 48, 50, 56, 55, 56, 49, 218, 37},
		[]byte{1, 0, 0, 11, 0, 34, 0, 1, 0, 1, 12, 23, 0, 1, 0, 1, 209, 185, 89, 49, 1, 1, 1, 20, 0, 209, 185, 89, 49, 2, 56, 54, 56, 49, 56, 51, 48, 51, 48, 48, 51, 55, 52, 56, 48, 66, 233},
		[]byte{1, 0, 0, 11, 0, 34, 0, 1, 0, 1, 12, 23, 0, 1, 0, 1, 17, 177, 66, 0, 1, 1, 1, 20, 0, 17, 177, 66, 0, 2, 52, 51, 55, 48, 55, 48, 53, 0, 0, 0, 0, 0, 0, 0, 0, 130, 79},
		[]byte{1, 0, 0, 11, 0, 34, 0, 1, 0, 1, 12, 23, 0, 1, 0, 1, 128, 0, 67, 0, 1, 1, 1, 20, 0, 128, 0, 67, 0, 2, 52, 51, 57, 49, 48, 52, 48, 0, 0, 0, 0, 0, 0, 0, 0, 232, 146},
		[]byte{1, 0, 0, 11, 0, 34, 0, 1, 0, 1, 12, 23, 0, 1, 0, 1, 80, 159, 252, 48, 1, 1, 1, 20, 0, 80, 159, 252, 48, 2, 56, 54, 49, 55, 56, 53, 48, 48, 51, 56, 53, 57, 57, 48, 49, 93, 168},
		[]byte{1, 0, 0, 11, 0, 34, 0, 1, 0, 1, 12, 23, 0, 1, 0, 1, 221, 39, 9, 0, 1, 1, 1, 20, 0, 221, 39, 9, 0, 2, 54, 48, 48, 48, 50, 57, 0, 0, 0, 0, 0, 0, 0, 0, 0, 145, 139},
		[]byte{1, 0, 0, 11, 0, 34, 0, 1, 0, 1, 12, 23, 0, 1, 0, 1, 203, 61, 67, 0, 1, 1, 1, 20, 0, 203, 61, 67, 0, 2, 52, 52, 48, 54, 55, 51, 49, 0, 0, 0, 0, 0, 0, 0, 0, 224, 115},
		[]byte{1, 0, 0, 11, 0, 34, 0, 1, 0, 1, 12, 23, 0, 1, 0, 1, 200, 70, 66, 0, 1, 1, 1, 20, 0, 200, 70, 66, 0, 2, 52, 51, 52, 51, 52, 57, 54, 0, 0, 0, 0, 0, 0, 0, 0, 76, 202},
		[]byte{1, 0, 0, 11, 0, 34, 0, 1, 0, 1, 12, 23, 0, 1, 0, 1, 97, 227, 32, 0, 1, 1, 1, 20, 0, 97, 227, 32, 0, 2, 50, 49, 53, 53, 51, 54, 49, 0, 0, 0, 0, 0, 0, 0, 0, 113, 19},
		[]byte{1, 0, 0, 11, 0, 34, 0, 1, 0, 1, 12, 23, 0, 1, 0, 1, 87, 60, 66, 0, 1, 1, 1, 20, 0, 87, 60, 66, 0, 2, 52, 51, 52, 48, 56, 50, 51, 0, 0, 0, 0, 0, 0, 0, 0, 89, 45},
		[]byte{1, 0, 0, 11, 0, 34, 0, 1, 0, 1, 12, 23, 0, 1, 0, 1, 8, 239, 8, 49, 1, 1, 1, 20, 0, 8, 239, 8, 49, 2, 56, 54, 50, 54, 51, 49, 48, 51, 55, 48, 53, 55, 50, 54, 49, 188, 15},
		[]byte{1, 0, 0, 11, 0, 34, 0, 1, 0, 1, 12, 23, 0, 1, 0, 1, 66, 231, 22, 49, 1, 1, 1, 20, 0, 66, 231, 22, 49, 2, 56, 54, 51, 53, 57, 49, 48, 50, 51, 51, 56, 51, 56, 53, 56, 129, 123},
		[]byte{1, 0, 0, 11, 0, 34, 0, 1, 0, 1, 12, 23, 0, 1, 0, 1, 16, 115, 66, 0, 1, 1, 1, 20, 0, 16, 115, 66, 0, 2, 52, 51, 53, 52, 56, 51, 50, 0, 0, 0, 0, 0, 0, 0, 0, 66, 252},
		[]byte{1, 0, 0, 11, 0, 34, 0, 1, 0, 1, 12, 23, 0, 1, 0, 1, 166, 75, 0, 0, 1, 1, 1, 20, 0, 166, 75, 0, 0, 2, 49, 57, 51, 54, 54, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 153, 179},
		[]byte{1, 0, 0, 11, 0, 34, 0, 1, 0, 1, 12, 23, 0, 1, 0, 1, 66, 231, 22, 49, 1, 1, 1, 20, 0, 66, 231, 22, 49, 2, 56, 54, 51, 53, 57, 49, 48, 50, 51, 51, 56, 56, 50, 54, 49, 37, 102},
		[]byte{1, 0, 0, 11, 0, 34, 0, 1, 0, 1, 12, 23, 0, 1, 0, 1, 220, 138, 69, 49, 1, 1, 1, 20, 0, 220, 138, 69, 49, 2, 56, 54, 54, 55, 57, 54, 48, 51, 53, 55, 57, 56, 54, 54, 53, 151, 87},
		[]byte{1, 0, 0, 11, 0, 34, 0, 1, 0, 1, 12, 23, 0, 1, 0, 1, 67, 231, 22, 49, 1, 1, 1, 20, 0, 67, 231, 22, 49, 2, 56, 54, 51, 53, 57, 49, 48, 50, 51, 54, 56, 54, 54, 53, 54, 95, 187},
		[]byte{1, 0, 0, 11, 0, 34, 0, 1, 0, 1, 12, 23, 0, 1, 0, 1, 5, 192, 66, 0, 1, 1, 1, 20, 0, 5, 192, 66, 0, 2, 52, 51, 55, 52, 53, 51, 51, 0, 0, 0, 0, 0, 0, 0, 0, 126, 220},
	} // Need samples

	bytesData = [][]byte{
		[]byte{
			1, 0, 0, 11, 0, 40, 0, 17, 81, 1, 18, 29, 0, 17, 81, 1, 150, 147, 56, 49, 2, 2, 16, 26, 0, 154, 136, 129, 16, 16, 209, 106, 154, 124, 34, 200, 68, 129, 0, 0, 42, 0, 0, 0, 0, 16, 133, 0, 0, 0, 0, 49, 198,
		},
		[]byte{
			1, 0, 0, 11, 0, 40, 0, 238, 80, 1, 173, 29, 0, 238, 80, 1, 166, 75, 0, 0, 2, 2, 16, 26, 0, 43, 122, 124, 16, 246, 79, 86, 154, 166, 161, 185, 68, 129, 64, 1, 202, 0, 0, 0, 0, 16, 147, 0, 0, 0, 0, 192, 89,
		},
		[]byte{
			1, 0, 0, 11, 0, 40, 0, 254, 80, 1, 9, 29, 0, 254, 80, 1, 84, 159, 252, 48, 2, 2, 16, 26, 0, 47, 61, 119, 16, 25, 132, 94, 154, 161, 85, 186, 68, 129, 200, 0, 185, 0, 0, 0, 0, 16, 124, 0, 0, 0, 0, 73, 117,
		},
	}

	binaryAuth = []string{} // Need samples

	binaryData = []string{
		"binaryData.txt",
	}

	maxBuffer = make([]byte, 0, 65535)
)

func TestReadPacketDataHEX(t *testing.T) {
	for i := range hexStringsData {
		var err error
		maxBuffer, err = hex.DecodeString(hexStringsData[i])
		if err != nil {
			t.Error(
				"Error occurred", err,
			)
		}

		data, responseCode := ReadPacket(maxBuffer)
		if responseCode != 0 {
			t.Error(
				"Response code has to be 0, but it is", responseCode,
			)
		}

		if len(data.ServicesFrameData) != 1 {
			t.Error(
				"Length of data has to be 1, but it is", len(data.ServicesFrameData),
			)
		}

		if data.ServicesFrameData[0].RecordData.SubrecordType != 16 {
			t.Error(
				"Subrecord type has to be 16, but it is", data.ServicesFrameData[0].RecordData.SubrecordType,
			)
		}
	}
}

func TestReadPacketAuthHEX(t *testing.T) {

	for i := range hexStringsAuth {
		var err error
		maxBuffer, err = hex.DecodeString(hexStringsAuth[i])
		if err != nil {
			t.Error(
				"Error occurred", err,
			)
		}

		data, responseCode := ReadPacket(maxBuffer)
		if responseCode != 0 {
			t.Error(
				"Response code has to be 0, but it is", responseCode,
			)
		}

		if len(data.ServicesFrameData) != 1 {
			t.Error(
				"Length of data has to be 1, but it is", len(data.ServicesFrameData),
			)
		}

		if data.ServicesFrameData[0].RecordData.SubrecordType != 1 {
			t.Error(
				"Subrecord type has to be 1, but it is", data.ServicesFrameData[0].RecordData.SubrecordType,
			)
		}
	}
}

func TestReadPacketDataBytes(t *testing.T) {

	for i := range bytesData {

		maxBuffer = bytesData[i]

		data, responseCode := ReadPacket(maxBuffer)
		if responseCode != 0 {
			t.Error(
				"Response code has to be 0, but it is", responseCode,
			)
		}

		if len(data.ServicesFrameData) != 1 {
			t.Error(
				"Length of data has to be 1, but it is", len(data.ServicesFrameData),
			)
		}

		if data.ServicesFrameData[0].RecordData.SubrecordType != 16 {
			t.Error(
				"Subrecord type has to be 16, but it is", data.ServicesFrameData[0].RecordData.SubrecordType,
			)
		}
	}
}

func TestReadPacketAuthBytes(t *testing.T) {
	for i := range bytesAuth {

		maxBuffer = bytesAuth[i]

		data, responseCode := ReadPacket(maxBuffer)
		if responseCode != 0 {
			t.Error(
				"Response code has to be 0, but it is", responseCode,
			)
		}

		if len(data.ServicesFrameData) != 1 {
			t.Error(
				"Length of data has to be 1, but it is", len(data.ServicesFrameData),
			)
		}

		if data.ServicesFrameData[0].RecordData.SubrecordType != 1 {
			t.Error(
				"Subrecord type has to be 16, but it is", data.ServicesFrameData[0].RecordData.SubrecordType,
			)
		}
	}
}

func TestReadPacketDataFile(t *testing.T) {

	for i := range binaryData {
		var err error
		maxBuffer, err := ioutil.ReadFile(binaryData[i])
		if err != nil {
			t.Error(
				"Error occurred", err,
			)
		}

		data, responseCode := ReadPacket(maxBuffer)
		if responseCode != 0 {
			t.Error(
				"Response code has to be 0, but it is", responseCode,
			)
		}

		if len(data.ServicesFrameData) != 137 {
			t.Error(
				"Length of data has to be 1, but it is", len(data.ServicesFrameData),
			)
		}

		for j := range data.ServicesFrameData {
			if data.ServicesFrameData[j].RecordData.SubrecordType != 16 && data.ServicesFrameData[j].RecordData.SubrecordType != 18 {
				t.Error(
					"Subrecord type has to be 16, but it is", data.ServicesFrameData[j].RecordData.SubrecordType,
				)
			}
		}
	}

}

func TestReadPacketAuthFile(t *testing.T) {

	for i := range binaryAuth {
		var err error
		maxBuffer, err := ioutil.ReadFile(binaryAuth[i])
		if err != nil {
			t.Error(
				"Error occurred", err,
			)
		}

		data, responseCode := ReadPacket(maxBuffer)
		if responseCode != 0 {
			t.Error(
				"Response code has to be 0, but it is", responseCode,
			)
		}

		if len(data.ServicesFrameData) != 1 {
			t.Error(
				"Length of data has to be 1, but it is", len(data.ServicesFrameData),
			)
		}

		for j := range data.ServicesFrameData {
			if data.ServicesFrameData[j].RecordData.SubrecordType != 1 {
				t.Error(
					"Subrecord type has to be 16, but it is", data.ServicesFrameData[j].RecordData.SubrecordType,
				)
			}
		}
	}
}

func TestBackPacket(t *testing.T) {

	egtsAuthHex := "0100020b0020000000014f1900000010010101160000000000523836363130343032393639303030380004417f"
	egtsAuth, _ := hex.DecodeString(egtsAuthHex)

	fmt.Println("Income", egtsAuth)

	parsedAuth, authCode := ReadPacket(egtsAuth)
	fmt.Println("auth code:", authCode)
	fmt.Println("parsed auth packet:")
	fmt.Println(parsedAuth.ResponseData, hex.EncodeToString(parsedAuth.ResponseData))

	checkHex := "0100030b001000000000b300000006000000580101000300000000d9d1"

	if hex.EncodeToString(parsedAuth.ResponseData) != checkHex {
		t.Errorf("Response to auth has to be %s, but got: %s\n", checkHex, hex.EncodeToString(parsedAuth.ResponseData))
	}

}
