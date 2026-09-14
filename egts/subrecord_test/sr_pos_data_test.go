package subrecord

import (
	"encoding/binary"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"testing"
	"time"

	"github.com/LdDl/go-egts/egts/packet"
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

type posDataCoordinatesTestCase struct {
	name      string
	flags     byte
	latitude  float64
	longitude float64
}

type posDataTimeTestCase struct {
	name     string
	seconds  uint32
	expected time.Time
}

type posDataReadTestCase struct {
	name string
	data []byte
}

func TestSRPosDataCoordinates(t *testing.T) {
	cases := []posDataCoordinatesTestCase{
		{name: "north east", flags: 0x01, latitude: 30, longitude: 60},
		{name: "south east", flags: 0x21, latitude: -30, longitude: 60},
		{name: "north west", flags: 0x41, latitude: 30, longitude: -60},
		{name: "south west", flags: 0x61, latitude: -30, longitude: -60},
		{name: "invalid north east", flags: 0x00, latitude: 30, longitude: 60},
		{name: "invalid south east", flags: 0x20, latitude: -30, longitude: 60},
		{name: "invalid north west", flags: 0x40, latitude: 30, longitude: -60},
		{name: "invalid south west", flags: 0x60, latitude: -30, longitude: -60},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			data := make([]byte, 21)
			binary.LittleEndian.PutUint32(data[4:8], 0x55555555)
			binary.LittleEndian.PutUint32(data[8:12], 0x55555555)
			data[12] = tc.flags
			result := subrecord.SRPosData{}
			err := result.Decode(data)
			assert.NoError(t, err)
			if err != nil {
				return
			}
			assert.InDelta(t, tc.latitude, result.Latitude, 1e-8)
			assert.InDelta(t, tc.longitude, result.Longitude, 1e-8)
			assert.Equal(t, fmt.Sprint(tc.flags&1), result.Valid)
			before := result
			encoded, err := result.Encode()
			assert.NoError(t, err)
			assert.Equal(t, data, encoded)
			assert.Equal(t, before, result)
		})
	}
}

func TestSRPosDataCoordinateLimits(t *testing.T) {
	cases := []posDataCoordinatesTestCase{
		{name: "origin", flags: 0x01},
		{name: "north east", flags: 0x01, latitude: 90, longitude: 180},
		{name: "south west", flags: 0x61, latitude: -90, longitude: -180},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			data := make([]byte, 21)
			if tc.latitude != 0 {
				binary.LittleEndian.PutUint32(data[4:8], 0xffffffff)
				binary.LittleEndian.PutUint32(data[8:12], 0xffffffff)
			}
			data[12] = tc.flags
			result := subrecord.SRPosData{}
			err := result.Decode(data)
			assert.NoError(t, err)
			assert.Equal(t, tc.latitude, result.Latitude)
			assert.Equal(t, tc.longitude, result.Longitude)
			encoded, err := result.Encode()
			assert.NoError(t, err)
			assert.Equal(t, data, encoded)
		})
	}
}

func TestSRPosDataNavigationTime(t *testing.T) {
	cases := []posDataTimeTestCase{
		{name: "epoch", seconds: 0, expected: time.Date(2010, 1, 1, 0, 0, 0, 0, time.UTC)},
		{name: "one second", seconds: 1, expected: time.Date(2010, 1, 1, 0, 0, 1, 0, time.UTC)},
		{name: "end of day", seconds: 86399, expected: time.Date(2010, 1, 1, 23, 59, 59, 0, time.UTC)},
		{name: "next day", seconds: 86400, expected: time.Date(2010, 1, 2, 0, 0, 0, 0, time.UTC)},
		{name: "signed limit", seconds: 0x7fffffff, expected: time.Date(2078, 1, 19, 3, 14, 7, 0, time.UTC)},
		{name: "above signed limit", seconds: 0x80000000, expected: time.Date(2078, 1, 19, 3, 14, 8, 0, time.UTC)},
		{name: "maximum", seconds: 0xffffffff, expected: time.Date(2146, 2, 7, 6, 28, 15, 0, time.UTC)},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			data := make([]byte, 21)
			binary.LittleEndian.PutUint32(data[:4], tc.seconds)
			result := subrecord.SRPosData{}
			err := result.Decode(data)
			assert.NoError(t, err)
			assert.Equal(t, tc.seconds, result.NavigationTimeUint)
			assert.Equal(t, tc.expected, result.NavigationTime)
			encoded, err := result.Encode()
			assert.NoError(t, err)
			assert.Equal(t, data, encoded)
		})
	}
}

func TestSRPosDataTruncated(t *testing.T) {
	withAltitude := make([]byte, 24)
	withAltitude[12] = 0x80
	withAltitude[21] = 0x12
	withAltitude[22] = 0x34
	withAltitude[23] = 0x56
	cases := []posDataReadTestCase{
		{name: "without altitude", data: make([]byte, 21)},
		{name: "with altitude", data: withAltitude},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			complete := subrecord.SRPosData{}
			err := complete.Decode(tc.data)
			assert.NoError(t, err)
			for length := 0; length < len(tc.data); length++ {
				t.Run(fmt.Sprintf("length/%d", length), func(t *testing.T) {
					result := subrecord.SRPosData{}
					err := result.Decode(tc.data[:length])
					assert.Error(t, err)
					assert.Equal(t, subrecord.SRPosData{}, result)
				})
			}
			data := []byte{packet.PosData, 0, 0}
			binary.LittleEndian.PutUint16(data[1:3], uint16(len(tc.data)-1))
			data = append(data, tc.data[:len(tc.data)-1]...)
			var records packet.RecordsData
			err = records.Decode(data)
			assert.Error(t, err)
			assert.Nil(t, records)
		})
	}
}

func TestSRPosDataRepeatedDecode(t *testing.T) {
	first := make([]byte, 24)
	first[0] = 1
	binary.LittleEndian.PutUint32(first[4:8], 0x55555555)
	binary.LittleEndian.PutUint32(first[8:12], 0x55555555)
	first[12] = 0xff
	first[13] = 100
	first[14] = 0xc0
	first[15] = 44
	first[16] = 100
	first[19] = 1
	first[20] = 3
	first[21] = 123
	result := subrecord.SRPosData{}
	err := result.Decode(first)
	assert.NoError(t, err)
	if err != nil {
		return
	}
	previous := result
	err = result.Decode(first[:23])
	assert.ErrorIs(t, err, io.ErrUnexpectedEOF)
	assert.Equal(t, previous, result)
	err = result.Decode(nil)
	assert.ErrorIs(t, err, io.EOF)
	assert.Equal(t, previous, result)

	err = result.Decode(make([]byte, 21))
	assert.NoError(t, err)
	expected := subrecord.SRPosData{
		NavigationTime:   time.Date(2010, 1, 1, 0, 0, 0, 0, time.UTC),
		OdometerBytes:    []byte{0, 0, 0},
		Valid:            "0",
		CoordinateSystem: "0",
		Fix:              "0",
		BlackBox:         "0",
		Move:             "0",
		LAHS:             "0",
		LOHS:             "0",
		AltitudeExists:   "0",
	}
	assert.Equal(t, expected, result)
	encoded, err := json.Marshal(result)
	assert.NoError(t, err)
	var fields map[string]json.RawMessage
	err = json.Unmarshal(encoded, &fields)
	assert.NoError(t, err)
	assert.Equal(t, json.RawMessage("null"), fields["ALT"])
}
