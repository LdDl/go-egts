package subrecord

import (
	"encoding/binary"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"testing"

	"github.com/LdDl/go-egts/egts/packet"
	"github.com/LdDl/go-egts/egts/subrecord"
	"github.com/stretchr/testify/assert"
)

var (
	SRLiquidLevelSensorCheckIncome = []string{"400000fbff0000", "00010000000000", "01010000000000", "0201006c630000", "03010000000000", "04010000000000", "05010000000000", "00020000000000", "01020000000000"}
)

func TestSRLiquidLevelSensorDecoding(t *testing.T) {
	for i := range SRLiquidLevelSensorCheckIncome {
		pkgHex := SRLiquidLevelSensorCheckIncome[i]
		pkgBytes, err := hex.DecodeString(pkgHex)
		if err != nil {
			t.Errorf("Error: %s", err.Error())
		}
		subr := subrecord.SRLiquidLevelSensor{}
		err = subr.Decode(pkgBytes)
		if err != nil {
			t.Errorf("Error: %s", err.Error())
		}
		hexed, err := subr.Encode()
		if err != nil {
			t.Errorf("Error: %s", err.Error())
		}
		if hex.EncodeToString(hexed) != SRLiquidLevelSensorCheckIncome[i] {
			t.Errorf("Have to be %s, but got %s", SRLiquidLevelSensorCheckIncome[i], hex.EncodeToString(hexed))
		}
	}
}

type liquidLevelValueTestCase struct {
	name     string
	data     []byte
	expected uint32
}

type liquidLevelEncodeErrorTestCase struct {
	name   string
	sensor uint8
	unit   string
	raw    string
	error  string
}

func TestSRLiquidLevelSensorRawData(t *testing.T) {
	for _, length := range []int{4, 5, 255, 256, 511, 512} {
		t.Run(fmt.Sprintf("length/%d", length), func(t *testing.T) {
			raw := make([]byte, length)
			for i := range raw {
				raw[i] = byte(i*3 + 1)
			}
			data := append([]byte{0x6f, 0x34, 0x12}, raw...)
			result := subrecord.SRLiquidLevelSensor{}
			err := result.Decode(data)
			assert.NoError(t, err)
			assert.Equal(t, "1", result.RawbFlag)
			assert.Equal(t, "1", result.LiquidLevelSensorErrorFlag)
			assert.Equal(t, "10", result.LiquidLevelSensorValueUnit)
			assert.Equal(t, uint8(7), result.LiquidLevelSensorNumber)
			assert.Equal(t, uint16(0x1234), result.MADDR)
			assert.Zero(t, result.LiquidLevelSensorb)
			assert.Equal(t, raw, result.RawData)
			encoded, err := result.Encode()
			assert.NoError(t, err)
			assert.Equal(t, data, encoded)
			assert.Equal(t, uint16(length+3), result.Len())
			assert.Equal(t, raw, result.RawData)
			data[3] = 0xff
			assert.Equal(t, raw, result.RawData)
			assert.Equal(t, byte(1), encoded[3])

			value := subrecord.SRLiquidLevelSensor{
				LiquidLevelSensorNumber:    7,
				RawbFlag:                   "1",
				LiquidLevelSensorValueUnit: "10",
				LiquidLevelSensorErrorFlag: "1",
				MADDR:                      0x1234,
				RawData:                    raw,
			}
			encoded, err = value.Encode()
			assert.NoError(t, err)
			assert.Equal(t, append([]byte{0x6f, 0x34, 0x12}, raw...), encoded)
		})
	}
}

func TestSRLiquidLevelSensorRawInvalidLength(t *testing.T) {
	for _, length := range []int{0, 1, 2, 3, 513} {
		t.Run(fmt.Sprintf("length/%d", length), func(t *testing.T) {
			data := append([]byte{8, 0, 0}, make([]byte, length)...)
			result := subrecord.SRLiquidLevelSensor{}
			err := result.Decode(data)
			assert.Error(t, err)
			assert.Equal(t, subrecord.SRLiquidLevelSensor{}, result)
			value := subrecord.SRLiquidLevelSensor{
				RawbFlag:                   "1",
				LiquidLevelSensorValueUnit: "00",
				LiquidLevelSensorErrorFlag: "0",
				RawData:                    make([]byte, length),
			}
			encoded, err := value.Encode()
			assert.Error(t, err)
			assert.Nil(t, encoded)
		})
	}
	value := subrecord.SRLiquidLevelSensor{
		RawbFlag:                   "1",
		LiquidLevelSensorValueUnit: "00",
		LiquidLevelSensorErrorFlag: "0",
	}
	encoded, err := value.Encode()
	assert.Error(t, err)
	assert.Nil(t, encoded)
}

func TestSRLiquidLevelSensorRepeatedDecode(t *testing.T) {
	numeric := []byte{0x67, 0x34, 0x12, 0x78, 0x56, 0x34, 0x12}
	raw := []byte{8, 1, 0, 0xde, 0xad, 0xbe, 0xef, 1}
	result := subrecord.SRLiquidLevelSensor{}
	err := result.Decode(numeric)
	assert.NoError(t, err)
	err = result.Decode(raw)
	assert.NoError(t, err)
	expected := subrecord.SRLiquidLevelSensor{
		RawbFlag:                   "1",
		LiquidLevelSensorValueUnit: "00",
		LiquidLevelSensorErrorFlag: "0",
		MADDR:                      1,
		RawData:                    []byte{0xde, 0xad, 0xbe, 0xef, 1},
	}
	assert.Equal(t, expected, result)
	for length := 0; length < len(numeric); length++ {
		err = result.Decode(numeric[:length])
		assert.Error(t, err)
		assert.Equal(t, expected, result)
	}
	err = result.Decode(append(numeric, 0))
	assert.Error(t, err)
	assert.Equal(t, expected, result)
	err = result.Decode(raw[:6])
	assert.Error(t, err)
	assert.Equal(t, expected, result)
	err = result.Decode(append([]byte{8, 0, 0}, make([]byte, 513)...))
	assert.Error(t, err)
	assert.Equal(t, expected, result)
	err = result.Decode(make([]byte, 7))
	assert.NoError(t, err)
	expected = subrecord.SRLiquidLevelSensor{
		RawbFlag:                   "0",
		LiquidLevelSensorValueUnit: "00",
		LiquidLevelSensorErrorFlag: "0",
	}
	assert.Equal(t, expected, result)
	encoded, err := json.Marshal(result)
	assert.NoError(t, err)
	assert.Contains(t, string(encoded), `"LLSD_RAW":null`)
}

func TestSRLiquidLevelSensorEncodeInvalidFlags(t *testing.T) {
	cases := []liquidLevelEncodeErrorTestCase{
		{name: "sensor 8", sensor: 8, unit: "00", raw: "0", error: "0"},
		{name: "sensor 255", sensor: 255, unit: "00", raw: "0", error: "0"},
		{name: "empty RDF", unit: "00", raw: "", error: "0"},
		{name: "wide RDF", unit: "00", raw: "00", error: "0"},
		{name: "invalid RDF", unit: "00", raw: "2", error: "0"},
		{name: "empty LLSEF", unit: "00", raw: "0", error: ""},
		{name: "wide LLSEF", unit: "00", raw: "0", error: "00"},
		{name: "invalid LLSEF", unit: "00", raw: "0", error: "2"},
		{name: "empty LLSVU", unit: "", raw: "0", error: "0"},
		{name: "short LLSVU", unit: "0", raw: "0", error: "0"},
		{name: "wide LLSVU", unit: "000", raw: "0", error: "0"},
		{name: "invalid LLSVU", unit: "02", raw: "0", error: "0"},
		{name: "negative LLSVU", unit: "-1", raw: "0", error: "0"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			value := subrecord.SRLiquidLevelSensor{
				LiquidLevelSensorNumber:    tc.sensor,
				LiquidLevelSensorValueUnit: tc.unit,
				RawbFlag:                   tc.raw,
				LiquidLevelSensorErrorFlag: tc.error,
			}
			previous := value
			encoded, err := value.Encode()
			assert.Error(t, err)
			assert.Nil(t, encoded)
			assert.Equal(t, previous, value)
		})
	}
}

func TestSRLiquidLevelSensorInRecordsData(t *testing.T) {
	for _, length := range []int{4, 512} {
		t.Run(fmt.Sprintf("length/%d", length), func(t *testing.T) {
			raw := make([]byte, length)
			for i := range raw {
				raw[i] = byte(i + 1)
			}
			data := []byte{packet.LiquidLevelSensor, 0, 0, 8, 1, 0}
			binary.LittleEndian.PutUint16(data[1:3], uint16(length+3))
			data = append(data, raw...)
			data = append(data, packet.StateData, 5, 0, 2, 134, 0, 20, 4)
			var records packet.RecordsData
			err := records.Decode(data)
			assert.NoError(t, err)
			if !assert.Len(t, records, 2) {
				return
			}
			liquid, ok := records[0].SubrecordData.(*subrecord.SRLiquidLevelSensor)
			assert.True(t, ok)
			if !ok {
				return
			}
			assert.Equal(t, raw, liquid.RawData)
			assert.IsType(t, &subrecord.SRStateData{}, records[1].SubrecordData)
			encoded, err := records.Encode()
			assert.NoError(t, err)
			assert.Equal(t, data, encoded)
		})
	}
}

func TestSRLiquidLevelSensorInvalidInRecordsData(t *testing.T) {
	for _, data := range [][]byte{
		{packet.LiquidLevelSensor, 6, 0, 0, 0, 0, 1, 2, 3},
		{packet.LiquidLevelSensor, 8, 0, 0, 0, 0, 1, 2, 3, 4, 5},
		{packet.LiquidLevelSensor, 6, 0, 8, 0, 0, 1, 2, 3},
	} {
		var records packet.RecordsData
		err := records.Decode(data)
		assert.Error(t, err)
		assert.Nil(t, records)
	}
}

func TestSRLiquidLevelSensorNumericValues(t *testing.T) {
	cases := []liquidLevelValueTestCase{
		{name: "zero", data: []byte{0, 0, 0, 0}, expected: 0},
		{name: "one", data: []byte{1, 0, 0, 0}, expected: 1},
		{name: "byte order", data: []byte{0x78, 0x56, 0x34, 0x12}, expected: 0x12345678},
		{name: "maximum", data: []byte{0xff, 0xff, 0xff, 0xff}, expected: 0xffffffff},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			data := append([]byte{0, 0x34, 0x12}, tc.data...)
			result := subrecord.SRLiquidLevelSensor{}
			err := result.Decode(data)
			assert.NoError(t, err)
			assert.Equal(t, uint16(0x1234), result.MADDR)
			assert.Equal(t, tc.expected, result.LiquidLevelSensorb)
			encoded, err := result.Encode()
			assert.NoError(t, err)
			assert.Equal(t, data, encoded)
			assert.Equal(t, uint16(7), result.Len())
			value := subrecord.SRLiquidLevelSensor{
				RawbFlag:                   "0",
				LiquidLevelSensorValueUnit: "00",
				LiquidLevelSensorErrorFlag: "0",
				MADDR:                      0x1234,
				LiquidLevelSensorb:         tc.expected,
			}
			encoded, err = value.Encode()
			assert.NoError(t, err)
			assert.Equal(t, data, encoded)
		})
	}
}

func TestSRLiquidLevelSensorNumericFlags(t *testing.T) {
	for sensor := byte(0); sensor < 8; sensor++ {
		for unit := byte(0); unit < 3; unit++ {
			for errorFlag := byte(0); errorFlag < 2; errorFlag++ {
				t.Run(fmt.Sprintf("sensor/%d/unit/%d/error/%d", sensor, unit, errorFlag), func(t *testing.T) {
					data := []byte{sensor | unit<<4 | errorFlag<<6, 0xff, 0xff, 1, 0, 0, 0}
					result := subrecord.SRLiquidLevelSensor{}
					err := result.Decode(data)
					assert.NoError(t, err)
					assert.Equal(t, sensor, result.LiquidLevelSensorNumber)
					assert.Equal(t, fmt.Sprintf("%02b", unit), result.LiquidLevelSensorValueUnit)
					assert.Equal(t, fmt.Sprint(errorFlag), result.LiquidLevelSensorErrorFlag)
					assert.Equal(t, "0", result.RawbFlag)
					assert.Equal(t, uint16(65535), result.MADDR)
					assert.Equal(t, uint32(1), result.LiquidLevelSensorb)
					encoded, err := result.Encode()
					assert.NoError(t, err)
					assert.Equal(t, data, encoded)
				})
			}
		}
	}
}

func TestSRLiquidLevelSensorNumericInvalidLength(t *testing.T) {
	data := []byte{0x67, 0x34, 0x12, 0x78, 0x56, 0x34, 0x12}
	for length := 0; length < len(data); length++ {
		t.Run(fmt.Sprintf("length/%d", length), func(t *testing.T) {
			result := subrecord.SRLiquidLevelSensor{}
			err := result.Decode(data[:length])
			assert.Error(t, err)
			assert.Equal(t, subrecord.SRLiquidLevelSensor{}, result)
			if length == 2 || length >= 4 {
				assert.ErrorIs(t, err, io.ErrUnexpectedEOF)
			} else {
				assert.ErrorIs(t, err, io.EOF)
			}
		})
	}
	result := subrecord.SRLiquidLevelSensor{}
	err := result.Decode(append(data, 0))
	assert.Error(t, err)
	assert.Equal(t, subrecord.SRLiquidLevelSensor{}, result)
}
