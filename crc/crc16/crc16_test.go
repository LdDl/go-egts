package crc16

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestKnownChecks(t *testing.T) {
	cases := []Params{
		CRC16_ARC, CRC16_AUG_CCITT, CRC16_BUYPASS, CRC16_CCITT_FALSE,
		CRC16_CDMA2000, CRC16_DDS_110, CRC16_DECT_R, CRC16_DECT_X,
		CRC16_DNP, CRC16_EN_13757, CRC16_GENIBUS, CRC16_MAXIM,
		CRC16_MCRF4XX, CRC16_RIELLO, CRC16_T10_DIF, CRC16_TELEDISK,
		CRC16_TMS37157, CRC16_USB, CRC16_CRC_A, CRC16_KERMIT,
		CRC16_MODBUS, CRC16_X_25, CRC16_XMODEM,
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
	table := MakeTable(CRC16_USB)
	data := []byte("123456789")
	for split := 0; split <= len(data); split++ {
		crc := Init(table)
		crc = Update(crc, data[:split], table)
		crc = Update(crc, data[split:], table)
		got := Complete(crc, table)
		assert.Equal(t, CRC16_USB.Check, got, "split %d", split)
	}
}
