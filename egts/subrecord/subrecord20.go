package subrecord

import "github.com/LdDl/go-egts/egts/utils"

// EgtsSrStateData - Используется для передачи на аппаратно-программный
// комплекс информации о состоянии абонентского терминала
type EgtsSrStateData struct {
	State                  string  /* State */
	NavigationModuleEnable bool    /* Navigation Module State */
	InternalBatteryEnable  bool    /* Internal Battery Used */
	BackupBatteryEnable    bool    /* Back Up Battery Used */
	MainPowerSourceVoltage float32 /* Main Power Source Voltage, in 0.1V */
	BackupBatteryVoltage   float32 /* Back Up Battery Voltage, in 0.1V */
	InternalBatteryVoltage float32 /* Internal Battery Voltage, in 0.1V */
}

//ParseEgtsSrStateData - EGTS_SR_STATE_DATA
func ParseEgtsSrStateData(b []byte) interface{} {
	var d EgtsSrStateData
	states := []string{"Idle", "EraGlonass", "Active", "EmergencyCall", "EmergencyMonitor", "Testing", "Service", "FirmwareUpdate"}
	d.State = states[int(b[0])]
	d.MainPowerSourceVoltage = float32(b[1]) * 0.1
	d.BackupBatteryVoltage = float32(b[2]) * 0.1
	d.InternalBatteryVoltage = float32(b[3]) * 0.1
	d.NavigationModuleEnable = utils.BitField(uint16(b[4]), 0).(bool)
	d.InternalBatteryEnable = utils.BitField(uint16(b[4]), 1).(bool)
	d.BackupBatteryEnable = utils.BitField(uint16(b[4]), 2).(bool)
	return d
}
