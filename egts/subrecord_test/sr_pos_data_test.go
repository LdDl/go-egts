package subrecord

import (
	"encoding/hex"
	"fmt"
	"testing"

	"github.com/LdDl/go-egts/egts/subrecord"
	"github.com/stretchr/testify/assert"
)

var (
	SRPosDataCheckIncome = []string{"9edd050f5fb4b49e8d7da2359b00009bc8550f0300120100"}
)

func TestSRPosDataDecoding(t *testing.T) {
	for i := range SRPosDataCheckIncome {
		pkgHex := SRPosDataCheckIncome[i]
		pkgBytes, err := hex.DecodeString(pkgHex)
		if err != nil {
			t.Errorf("Error: %s", err.Error())
		}
		subr := subrecord.SRPosData{}
		err = subr.Decode(pkgBytes)
		if err != nil {
			t.Errorf("Error: %s", err.Error())
		}
		hexed, err := subr.Encode()
		if err != nil {
			t.Errorf("Error: %s", err.Error())
		}
		if hex.EncodeToString(hexed) != SRPosDataCheckIncome[i] {
			t.Errorf("Have to be %s, but got %s", SRPosDataCheckIncome[i], hex.EncodeToString(hexed))
		}
	}
}

type posDataDirectionTestCase struct {
	name      string
	direction byte
	highBit   byte
	value     uint16
}

type posDataUint24TestCase struct {
	name     string
	data     []byte
	odometer int
	altitude uint32
}

var posDataUint24Cases = []posDataUint24TestCase{
	{name: "zero", data: []byte{0, 0, 0}, odometer: 0, altitude: 0},
	{name: "low byte", data: []byte{100, 0, 0}, odometer: 10, altitude: 100},
	{name: "middle byte", data: []byte{0, 1, 0}, odometer: 25, altitude: 256},
	{name: "high byte", data: []byte{0, 0, 1}, odometer: 6553, altitude: 65536},
	{name: "mixed bytes", data: []byte{0x56, 0x34, 0x12}, odometer: 119304, altitude: 1193046},
	{name: "maximum", data: []byte{0xff, 0xff, 0xff}, odometer: 1677721, altitude: 16777215},
}

func TestSRPosDataDirection(t *testing.T) {
	cases := []posDataDirectionTestCase{
		{name: "north", direction: 0, highBit: 0, value: 0},
		{name: "127 degrees", direction: 127, highBit: 0, value: 127},
		{name: "128 degrees", direction: 128, highBit: 0, value: 128},
		{name: "255 degrees", direction: 255, highBit: 0, value: 255},
		{name: "256 degrees", direction: 0, highBit: 1, value: 256},
		{name: "west", direction: 14, highBit: 1, value: 270},
		{name: "300 degrees", direction: 44, highBit: 1, value: 300},
		{name: "359 degrees", direction: 103, highBit: 1, value: 359},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			data := make([]byte, 21)
			data[12] = 1
			data[13] = 100
			data[14] = tc.highBit << 7
			data[15] = tc.direction
			result := subrecord.SRPosData{}
			err := result.Decode(data)
			assert.NoError(t, err)
			if err != nil {
				return
			}
			assert.Equal(t, tc.direction, result.Direction)
			assert.Equal(t, tc.highBit, result.DirhFlag)
			assert.Equal(t, tc.value, result.DirectionValue)
			assert.Equal(t, 10, result.Speed)
			assert.Equal(t, uint8(0), result.AltsFlag)
			encoded, err := result.Encode()
			assert.NoError(t, err)
			assert.Equal(t, data, encoded)
		})
	}
}

func TestSRPosDataOdometer(t *testing.T) {
	for _, tc := range posDataUint24Cases {
		t.Run(tc.name, func(t *testing.T) {
			data := make([]byte, 21)
			data[12] = 1
			copy(data[16:19], tc.data)
			data[19] = 0xa5
			data[20] = 3
			result := subrecord.SRPosData{}
			err := result.Decode(data)
			assert.NoError(t, err)
			if err != nil {
				return
			}
			assert.Equal(t, tc.odometer, result.Odometer)
			assert.Equal(t, tc.data, result.OdometerBytes)
			assert.Equal(t, uint8(0xa5), result.DigitalInputs)
			assert.Equal(t, uint8(3), result.Source)
			encoded, err := result.Encode()
			assert.NoError(t, err)
			assert.Equal(t, data, encoded)
		})
	}
}

func TestSRPosDataAltitude(t *testing.T) {
	for _, tc := range posDataUint24Cases {
		t.Run(tc.name, func(t *testing.T) {
			for _, sign := range []byte{0, 1} {
				t.Run(fmt.Sprintf("sign/%d", sign), func(t *testing.T) {
					data := make([]byte, 24)
					data[12] = 0x81
					data[13] = 100
					data[14] = 0x80 | sign<<6
					data[15] = 44
					copy(data[21:24], tc.data)
					result := subrecord.SRPosData{}
					err := result.Decode(data)
					assert.NoError(t, err)
					if err != nil {
						return
					}
					assert.Equal(t, tc.altitude, result.Altitude)
					assert.Equal(t, tc.data, result.AltitudeBytes)
					assert.Equal(t, "1", result.AltitudeExists)
					assert.Equal(t, sign, result.AltsFlag)
					assert.Equal(t, uint8(1), result.DirhFlag)
					assert.Equal(t, uint16(300), result.DirectionValue)
					assert.Equal(t, 10, result.Speed)
					encoded, err := result.Encode()
					assert.NoError(t, err)
					assert.Equal(t, data, encoded)
				})
			}
		})
	}
}
