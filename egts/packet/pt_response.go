package packet

import (
	"encoding/binary"
)

// PTResponse Subrecord of type EGTS_PT_RESPONSE
type PTResponse struct {
	ResponsePacketID uint16    `json:"RPID"` // RPID Response Packet ID
	ProcessingResult uint8     `json:"PR"`   // PR Processing Result
	SDR              BytesData `json:"SFRD"` // SFRD (Services Frame Data)
}

// Decode Parse slice of bytes to EGTS_PT_RESPONSE
func (response *PTResponse) Decode(b []byte) {
	//  RPID Response Packet ID
	response.ResponsePacketID = binary.LittleEndian.Uint16(b[0:2])
	// PR Processing Result
	response.ProcessingResult = uint8(b[2])
	// SFRD (Services Frame Data)
	if len(b[3:]) > 0 {
		response.SDR = &ServicesFrameData{}
		response.SDR.Decode(b[3:])
	}
}

// Encode Parse EGTS_PT_RESPONSE to slice of bytes
func (response *PTResponse) Encode() (b []byte) {
	rpid := make([]byte, 2)
	binary.LittleEndian.PutUint16(rpid, response.ResponsePacketID)
	b = append(b, rpid...)
	b = append(b, response.ProcessingResult)
	if response.SDR != nil {
		sdr := response.SDR.Encode()
		b = append(b, sdr...)
	}
	return b
}

// Len Returns length of bytes slice
func (response *PTResponse) Len() (l uint16) {
	l = uint16(len(response.Encode()))
	return l
}
