package subrecord

import (
	"bytes"
	"fmt"
	"strconv"
	"strings"
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
	MainPowerSourceVoltageByte uint8   `json:"MPSV"`
	BackupBatteryVoltageByte   uint8   `json:"BBV"`
	InternalBatteryVoltageByte uint8   `json:"IBV"`
}

var (
	// Possible states
	states = [8]string{"Idle", "EraGlonass", "Active", "EmergencyCall", "EmergencyMonitor", "Testing", "Service", "FirmwareUpdate"}
)

// Decode Parse array of bytes to EGTS_SR_STATE_DATA
func (subr *SRStateData) Decode(b []byte) (err error) {
	buffer := bytes.NewReader(b)
	if subr.StateByte, err = buffer.ReadByte(); err != nil {
		return fmt.Errorf("EGTS_SR_STATE_DATA; Error reading ST")
	}
	if subr.StateByte < 0 || subr.StateByte >= 8 {
		return fmt.Errorf("EGTS_SR_STATE_DATA; Such ST does not exists")
	}
	subr.State = states[subr.StateByte]

	if subr.MainPowerSourceVoltageByte, err = buffer.ReadByte(); err != nil {
		return fmt.Errorf("EGTS_SR_STATE_DATA; Error reading MPSV")
	}

	if subr.BackupBatteryVoltageByte, err = buffer.ReadByte(); err != nil {
		return fmt.Errorf("EGTS_SR_STATE_DATA; Error reading BBV")
	}

	if subr.InternalBatteryVoltageByte, err = buffer.ReadByte(); err != nil {
		return fmt.Errorf("EGTS_SR_STATE_DATA; Error reading IBV")
	}

	subr.MainPowerSourceVoltage = float32(subr.MainPowerSourceVoltageByte) * 0.1
	subr.BackupBatteryVoltage = float32(subr.BackupBatteryVoltageByte) * 0.1
	subr.InternalBatteryVoltage = float32(subr.InternalBatteryVoltageByte) * 0.1

	flagByte := byte(0)
	if flagByte, err = buffer.ReadByte(); err != nil {
		return fmt.Errorf("EGTS_SR_STATE_DATA; Error reading flags")
	}
	flagByteAsBits := fmt.Sprintf("%08b", flagByte)
	// log.Println("parsed", flagByteAsBits)
	subr.NavigationModuleEnable = flagByteAsBits[5:6]
	subr.InternalBatteryEnable = flagByteAsBits[6:7]
	subr.BackupBatteryEnable = flagByteAsBits[7:]

	return nil
}

// Encode Parse EGTS_SR_STATE_DATA to array of bytes
func (subr *SRStateData) Encode() (b []byte, err error) {
	buffer := new(bytes.Buffer)
	if err = buffer.WriteByte(subr.StateByte); err != nil {
		return nil, fmt.Errorf("EGTS_SR_STATE_DATA; Error writing ST")
	}
	if err = buffer.WriteByte(subr.MainPowerSourceVoltageByte); err != nil {
		return nil, fmt.Errorf("EGTS_SR_STATE_DATA; Error writing MPSV")
	}
	if err = buffer.WriteByte(subr.BackupBatteryVoltageByte); err != nil {
		return nil, fmt.Errorf("EGTS_SR_STATE_DATA; Error writing BBV")
	}
	if err = buffer.WriteByte(subr.InternalBatteryVoltageByte); err != nil {
		return nil, fmt.Errorf("EGTS_SR_STATE_DATA; Error writing IBV")
	}

	flagsBits := strings.Repeat("0", 5) + subr.NavigationModuleEnable + subr.InternalBatteryEnable + subr.BackupBatteryEnable
	flags := uint64(0)
	flags, _ = strconv.ParseUint(flagsBits, 2, 8)
	if err = buffer.WriteByte(uint8(flags)); err != nil {
		return nil, fmt.Errorf("EGTS_SR_STATE_DATA; Error writing flags byte")
	}

	return buffer.Bytes(), nil
}

// Len Returns length of bytes slice
func (subr *SRStateData) Len() (l uint16) {
	encoded, _ := subr.Encode()
	l = uint16(len(encoded))
	return l
}
