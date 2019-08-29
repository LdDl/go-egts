package subrecord

import "github.com/LdDl/go-egts/egts/utils"

// SRStateData EGTS_SR_STATE_DATA
/*
	Используется для передачи на аппаратно-программный комплекс
	информации о состоянии абонентского терминала
*/
type SRStateData struct {
	State                  string  `json:"ST"`   /* State */
	NavigationModuleEnable bool    `json:"NMS"`  /* Navigation Module State */
	InternalBatteryEnable  bool    `json:"IBU"`  /* Internal Battery Used */
	BackupBatteryEnable    bool    `json:"BBU"`  /* Back Up Battery Used */
	MainPowerSourceVoltage float32 `json:"MPSV"` /* Main Power Source Voltage, in 0.1V */
	BackupBatteryVoltage   float32 `json:"BBV"`  /* Back Up Battery Voltage, in 0.1V */
	InternalBatteryVoltage float32 `json:"IBV"`  /* Internal Battery Voltage, in 0.1V */
}

//BytesToSRStateData Parse array of bytes to EGTS_SR_STATE_DATA
func BytesToSRStateData(b []byte) (subr SRStateData) {
	states := []string{"Idle", "EraGlonass", "Active", "EmergencyCall", "EmergencyMonitor", "Testing", "Service", "FirmwareUpdate"}
	subr.State = states[int(b[0])]
	subr.MainPowerSourceVoltage = float32(b[1]) * 0.1
	subr.BackupBatteryVoltage = float32(b[2]) * 0.1
	subr.InternalBatteryVoltage = float32(b[3]) * 0.1
	subr.NavigationModuleEnable = utils.BitField(uint16(b[4]), 0).(bool)
	subr.InternalBatteryEnable = utils.BitField(uint16(b[4]), 1).(bool)
	subr.BackupBatteryEnable = utils.BitField(uint16(b[4]), 2).(bool)
	return subr
}
