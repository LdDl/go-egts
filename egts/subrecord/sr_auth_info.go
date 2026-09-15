package subrecord

import (
	"bytes"
	"fmt"
	"strings"
)

type SRAuthInfo struct {
	UserName       string  `json:"UNM"`
	Password       string  `json:"UPSW"`
	ServerSequence *string `json:"SS"`
}

func (subr *SRAuthInfo) Decode(b []byte) error {
	fields := bytes.Split(b, []byte{0})
	if (len(fields) != 3 && len(fields) != 4) || len(fields[len(fields)-1]) != 0 {
		return fmt.Errorf("EGTS_SR_AUTH_INFO; Invalid string delimiters")
	}
	if len(fields[0]) > 32 || len(fields[1]) > 32 {
		return fmt.Errorf("EGTS_SR_AUTH_INFO; UNM and UPSW must not exceed 32 bytes")
	}
	decoded := SRAuthInfo{UserName: string(fields[0]), Password: string(fields[1])}
	if len(fields) == 4 {
		if len(fields[2]) > 255 {
			return fmt.Errorf("EGTS_SR_AUTH_INFO; SS must not exceed 255 bytes")
		}
		sequence := string(fields[2])
		decoded.ServerSequence = &sequence
	}
	*subr = decoded
	return nil
}

func (subr *SRAuthInfo) Encode() ([]byte, error) {
	if subr == nil {
		return nil, fmt.Errorf("EGTS_SR_AUTH_INFO; Subrecord is nil")
	}
	if len(subr.UserName) > 32 || len(subr.Password) > 32 {
		return nil, fmt.Errorf("EGTS_SR_AUTH_INFO; UNM and UPSW must not exceed 32 bytes")
	}
	if strings.ContainsRune(subr.UserName, 0) || strings.ContainsRune(subr.Password, 0) {
		return nil, fmt.Errorf("EGTS_SR_AUTH_INFO; Embedded string delimiter")
	}
	data := append([]byte(subr.UserName), 0)
	data = append(data, []byte(subr.Password)...)
	data = append(data, 0)
	if subr.ServerSequence != nil {
		if strings.ContainsRune(*subr.ServerSequence, 0) || len(*subr.ServerSequence) > 255 {
			return nil, fmt.Errorf("EGTS_SR_AUTH_INFO; Invalid SS")
		}
		data = append(data, []byte(*subr.ServerSequence)...)
		data = append(data, 0)
	}
	return data, nil
}

func (subr *SRAuthInfo) Len() uint16 {
	data, err := subr.Encode()
	if err != nil {
		return 0
	}
	return uint16(len(data))
}
