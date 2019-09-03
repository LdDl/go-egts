package subrecord

import (
	"encoding/hex"
	"fmt"
	"log"
	"strconv"
)

// SRStateData EGTS_SR_STATE_DATA
/*
	Используется для передачи на аппаратно-программный комплекс
	информации о состоянии абонентского терминала
*/
type SRStateData struct {
	/* Header section */
	State     string `json:"ST_value"` /* State (string representation) */
	StateByte uint8  `json:"ST"`       /* State */
	/* Flags section */
	NavigationModuleEnable string `json:"NMS"` /* Navigation Module State */
	InternalBatteryEnable  string `json:"IBU"` /* Internal Battery Used */
	BackupBatteryEnable    string `json:"BBU"` /* Back Up Battery Used */
	/* Data section */
	MainPowerSourceVoltage     float32 `json:"MPSV_value"` /* Main Power Source Voltage, in 0.1V */
	BackupBatteryVoltage       float32 `json:"BBV_value"`  /* Back Up Battery Voltage, in 0.1V */
	InternalBatteryVoltage     float32 `json:"IBV_value"`  /* Internal Battery Voltage, in 0.1V */
	MainPowerSourceVoltageByte uint8   `json:"MPSV"`       /* Main Power Source Voltage, in 0.1V */
	BackupBatteryVoltageByte   uint8   `json:"BBV"`        /* Back Up Battery Voltage, in 0.1V */
	InternalBatteryVoltageByte uint8   `json:"IBV"`        /* Internal Battery Voltage, in 0.1V */
}

// Decode Parse array of bytes to EGTS_SR_STATE_DATA
func (subr *SRStateData) Decode(b []byte) {

	states := []string{"Idle", "EraGlonass", "Active", "EmergencyCall", "EmergencyMonitor", "Testing", "Service", "FirmwareUpdate"}
	subr.State = states[int(b[0])]
	subr.StateByte = b[0]

	subr.MainPowerSourceVoltage = float32(b[1]) * 0.1
	subr.BackupBatteryVoltage = float32(b[2]) * 0.1
	subr.InternalBatteryVoltage = float32(b[3]) * 0.1
	subr.MainPowerSourceVoltageByte = b[1]
	subr.BackupBatteryVoltageByte = b[2]
	subr.InternalBatteryVoltageByte = b[3]

	flagByte := uint16(b[4])
	flagByteAsBits := fmt.Sprintf("%08b", flagByte)
	subr.NavigationModuleEnable = flagByteAsBits[5:6]
	subr.InternalBatteryEnable = flagByteAsBits[6:7]
	subr.BackupBatteryEnable = flagByteAsBits[7:]

	log.Println(b, hex.EncodeToString(b))
	log.Println(subr.Encode())
}

// Encode Parse EGTS_SR_STATE_DATA to array of bytes
func (subr *SRStateData) Encode() (b []byte) {
	b = append(b, subr.StateByte)
	b = append(b, subr.MainPowerSourceVoltageByte)
	b = append(b, subr.BackupBatteryVoltageByte)
	b = append(b, subr.InternalBatteryVoltageByte)

	flagsBits := subr.NavigationModuleEnable + subr.InternalBatteryEnable + subr.BackupBatteryEnable
	flags := uint64(0)
	flags, _ = strconv.ParseUint(flagsBits, 2, 8)
	b = append(b, uint8(flags))
	return b
}

// Len Returns length of bytes slice
func (subr *SRStateData) Len() (l uint16) {
	l = uint16(len(subr.Encode()))
	return l
}
