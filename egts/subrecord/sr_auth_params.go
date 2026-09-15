package subrecord

import "fmt"

// SRAuthParams currently supports authentication without encryption.
type SRAuthParams struct {
	Flags uint8 `json:"FLG"`
}

func (subr *SRAuthParams) Decode(b []byte) error {
	if len(b) != 1 || b[0] != 0 {
		return fmt.Errorf("EGTS_SR_AUTH_PARAMS; Only authentication without encryption is supported")
	}
	subr.Flags = 0
	return nil
}

func (subr *SRAuthParams) Encode() ([]byte, error) {
	if subr == nil || subr.Flags != 0 {
		return nil, fmt.Errorf("EGTS_SR_AUTH_PARAMS; Only authentication without encryption is supported")
	}
	return []byte{0}, nil
}

func (subr *SRAuthParams) Len() uint16 {
	data, err := subr.Encode()
	if err != nil {
		return 0
	}
	return uint16(len(data))
}
