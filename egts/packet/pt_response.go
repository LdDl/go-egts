package packet

import (
	"bytes"
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
	buffer := bytes.NewBuffer(b)

	//  RPID Response Packet ID
	rpid := make([]byte, 2)
	if _, err = buffer.Read(rpid); err != nil {
		return fmt.Errorf("EGTS_PT_RESPONSE; Error reading RPID")
	}
	response.ResponsePacketID = binary.LittleEndian.Uint16(rpid)

	// PR Processing Result
	if response.ProcessingResult, err = buffer.ReadByte(); err != nil {
		return fmt.Errorf("EGTS_PT_RESPONSE; Error reading PR")
	}

	// SFRD (Services Frame Data)
	if buffer.Len() > 0 {
		response.SDR = &ServicesFrameData{}
		err := response.SDR.Decode(buffer.Bytes())
		if err != nil {
			return fmt.Errorf("EGTS_PT_RESPONSE;" + err.Error())
		}
	}
	return nil
}

// Encode Parse EGTS_PT_RESPONSE to slice of bytes
func (response *PTResponse) Encode() (b []byte, err error) {
	buffer := new(bytes.Buffer)
	if err = binary.Write(buffer, binary.LittleEndian, response.ResponsePacketID); err != nil {
		return nil, fmt.Errorf("EGTS_PT_RESPONSE; Error writing RPID")
	}

	if err = buffer.WriteByte(response.ProcessingResult); err != nil {
		return nil, fmt.Errorf("EGTS_PT_RESPONSE; Error writing PR")
	}
	if response.SDR != nil {
		sdr, err := response.SDR.Encode()
		if err != nil {
			return nil, fmt.Errorf("EGTS_PT_RESPONSE;" + err.Error())
		}
		buffer.Write(sdr)
	}
	return buffer.Bytes(), nil
}

// Len Returns length of bytes slice
func (response *PTResponse) Len() (l uint16) {
	encoded, _ := response.Encode()
	l = uint16(len(encoded))
	return l
}
