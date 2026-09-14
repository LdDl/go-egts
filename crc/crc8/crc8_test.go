package crc8

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestKnownChecks(t *testing.T) {
	cases := []Params{
		CRC8, CRC8_CDMA2000, CRC8_DARC, CRC8_DVB_S2, CRC8_EBU,
		CRC8_I_CODE, CRC8_ITU, CRC8_MAXIM, CRC8_ROHC, CRC8_WCDMA,
	}
	for _, params := range cases {
		t.Run(params.Name, func(t *testing.T) {
			table := MakeTable(params)
			got := Checksum([]byte("123456789"), table)
			assert.Equal(t, params.Check, got)
		})
	}
}

func TestUpdateChunks(t *testing.T) {
	table := MakeTable(CRC8_MAXIM)
	data := []byte("123456789")
	for split := 0; split <= len(data); split++ {
		crc := Init(table)
		crc = Update(crc, data[:split], table)
		crc = Update(crc, data[split:], table)
		got := Complete(crc, table)
		assert.Equal(t, CRC8_MAXIM.Check, got, "split %d", split)
	}
}
