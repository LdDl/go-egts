package packet

import (
	"encoding/hex"
	"io/ioutil"
	"testing"
)

var (
	hexStringsAuth = []string{
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

	bytesAuth = [][]byte{} // Need samples

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
