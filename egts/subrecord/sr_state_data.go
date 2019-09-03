package subrecord

import (
	"fmt"
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
	MainPowerSourceVoltage float32 `json:"MPSV"` /* Main Power Source Voltage, in 0.1V */
	BackupBatteryVoltage   float32 `json:"BBV"`  /* Back Up Battery Voltage, in 0.1V */
	InternalBatteryVoltage float32 `json:"IBV"`  /* Internal Battery Voltage, in 0.1V */
}

// Decode Parse array of bytes to EGTS_SR_STATE_DATA
func (subr *SRStateData) Decode(b []byte) {
	states := []string{"Idle", "EraGlonass", "Active", "EmergencyCall", "EmergencyMonitor", "Testing", "Service", "FirmwareUpdate"}
	subr.State = states[int(b[0])]
	subr.StateByte = b[0]
	subr.MainPowerSourceVoltage = float32(b[1]) * 0.1
	subr.BackupBatteryVoltage = float32(b[2]) * 0.1
	subr.InternalBatteryVoltage = float32(b[3]) * 0.1

	flagByte := uint16(b[4])
	flagByteAsBits := fmt.Sprintf("%08b", flagByte)
	subr.NavigationModuleEnable = flagByteAsBits[5:6]
	subr.InternalBatteryEnable = flagByteAsBits[6:7]
	subr.BackupBatteryEnable = flagByteAsBits[7:]
}

// Encode Parse EGTS_SR_STATE_DATA to array of bytes
func (subr *SRStateData) Encode() (b []byte) {
	return b
}

// Len Returns length of bytes slice
func (subr *SRStateData) Len() (l uint16) {
	l = uint16(len(subr.Encode()))
	return l
}
