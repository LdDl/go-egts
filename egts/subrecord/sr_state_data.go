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
	decoded := SRStateData{}
	decoded.StateByte, err = buffer.ReadByte()
	if err != nil {
		return fmt.Errorf("EGTS_SR_STATE_DATA; Error reading ST: %w", err)
	}
	if decoded.StateByte >= 8 {
		return fmt.Errorf("EGTS_SR_STATE_DATA; Such ST does not exists")
	}
	decoded.State = states[decoded.StateByte]

	decoded.MainPowerSourceVoltageByte, err = buffer.ReadByte()
	if err != nil {
		return fmt.Errorf("EGTS_SR_STATE_DATA; Error reading MPSV: %w", err)
	}

	decoded.BackupBatteryVoltageByte, err = buffer.ReadByte()
	if err != nil {
		return fmt.Errorf("EGTS_SR_STATE_DATA; Error reading BBV: %w", err)
	}

	decoded.InternalBatteryVoltageByte, err = buffer.ReadByte()
	if err != nil {
		return fmt.Errorf("EGTS_SR_STATE_DATA; Error reading IBV: %w", err)
	}

	decoded.MainPowerSourceVoltage = float32(decoded.MainPowerSourceVoltageByte) * 0.1
	decoded.BackupBatteryVoltage = float32(decoded.BackupBatteryVoltageByte) * 0.1
	decoded.InternalBatteryVoltage = float32(decoded.InternalBatteryVoltageByte) * 0.1

	flagByte := byte(0)
	flagByte, err = buffer.ReadByte()
	if err != nil {
		return fmt.Errorf("EGTS_SR_STATE_DATA; Error reading flags: %w", err)
	}
	flagByteAsBits := fmt.Sprintf("%08b", flagByte)
	decoded.NavigationModuleEnable = flagByteAsBits[5:6]
	decoded.InternalBatteryEnable = flagByteAsBits[6:7]
	decoded.BackupBatteryEnable = flagByteAsBits[7:]

	if buffer.Len() != 0 {
		return fmt.Errorf("EGTS_SR_STATE_DATA; Unexpected trailing data")
	}
	*subr = decoded
	return nil
}

// Encode Parse EGTS_SR_STATE_DATA to array of bytes
func (subr *SRStateData) Encode() (b []byte, err error) {
	if subr.StateByte >= 8 {
		return nil, fmt.Errorf("EGTS_SR_STATE_DATA; Such ST does not exists")
	}
	if subr.NavigationModuleEnable != "0" && subr.NavigationModuleEnable != "1" {
		return nil, fmt.Errorf("EGTS_SR_STATE_DATA; Invalid NMS flag")
	}
	if subr.InternalBatteryEnable != "0" && subr.InternalBatteryEnable != "1" {
		return nil, fmt.Errorf("EGTS_SR_STATE_DATA; Invalid IBU flag")
	}
	if subr.BackupBatteryEnable != "0" && subr.BackupBatteryEnable != "1" {
		return nil, fmt.Errorf("EGTS_SR_STATE_DATA; Invalid BBU flag")
	}

	buffer := new(bytes.Buffer)
	err = buffer.WriteByte(subr.StateByte)
	if err != nil {
		return nil, fmt.Errorf("EGTS_SR_STATE_DATA; Error writing ST: %w", err)
	}
	err = buffer.WriteByte(subr.MainPowerSourceVoltageByte)
	if err != nil {
		return nil, fmt.Errorf("EGTS_SR_STATE_DATA; Error writing MPSV: %w", err)
	}
	err = buffer.WriteByte(subr.BackupBatteryVoltageByte)
	if err != nil {
		return nil, fmt.Errorf("EGTS_SR_STATE_DATA; Error writing BBV: %w", err)
	}
	err = buffer.WriteByte(subr.InternalBatteryVoltageByte)
	if err != nil {
		return nil, fmt.Errorf("EGTS_SR_STATE_DATA; Error writing IBV: %w", err)
	}

	flagsBits := strings.Repeat("0", 5) + subr.NavigationModuleEnable + subr.InternalBatteryEnable + subr.BackupBatteryEnable
	flags := uint64(0)
	flags, err = strconv.ParseUint(flagsBits, 2, 8)
	if err != nil {
		return nil, fmt.Errorf("EGTS_SR_STATE_DATA; Error parsing flags: %w", err)
	}
	err = buffer.WriteByte(uint8(flags))
	if err != nil {
		return nil, fmt.Errorf("EGTS_SR_STATE_DATA; Error writing flags byte: %w", err)
	}

	return buffer.Bytes(), nil
}

// Len Returns length of bytes slice
func (subr *SRStateData) Len() (l uint16) {
	encoded, _ := subr.Encode()
	l = uint16(len(encoded))
	return l
}
