package packet

import (
	"encoding/binary"
	"fmt"
)

// PTResponse Subrecord of type EGTS_PT_RESPONSE
type PTResponse struct {
	ResponsePacketID uint16    `json:"RPID"` // RPID Response Packet ID
	ProcessingResult uint8     `json:"PR"`   // PR Processing Result
	SDR              BytesData `json:"SFRD"` // SFRD (Services Frame Data)
}

// Decode Parse slice of bytes to EGTS_PT_RESPONSE
func (response *PTResponse) Decode(b []byte) (err error) {
	//  RPID Response Packet ID
	response.ResponsePacketID = binary.LittleEndian.Uint16(b[0:2])
	// PR Processing Result
	response.ProcessingResult = uint8(b[2])
	// SFRD (Services Frame Data)
	if len(b[3:]) > 0 {
		response.SDR = &ServicesFrameData{}

		err := response.SDR.Decode(b[3:])
		if err != nil {
			return fmt.Errorf("EGTS_PT_RESPONSE;" + err.Error())
		}
	}
	return nil
}

// Encode Parse EGTS_PT_RESPONSE to slice of bytes
func (response *PTResponse) Encode() (b []byte, err error) {
	rpid := make([]byte, 2)
	binary.LittleEndian.PutUint16(rpid, response.ResponsePacketID)
	b = append(b, rpid...)
	b = append(b, response.ProcessingResult)
	if response.SDR != nil {
		sdr, _ := response.SDR.Encode()
		b = append(b, sdr...)
	}
	return b, nil
}

// Len Returns length of bytes slice
func (response *PTResponse) Len() (l uint16) {
	encoded, _ := response.Encode()
	l = uint16(len(encoded))
	return l
}
