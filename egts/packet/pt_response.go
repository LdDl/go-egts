package packet

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"io"
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
	decoded := PTResponse{}

	//  RPID Response Packet ID
	rpid := make([]byte, 2)
	_, err = io.ReadFull(buffer, rpid)
	if err != nil {
		return fmt.Errorf("EGTS_PT_RESPONSE; Error reading RPID: %w", err)
	}
	decoded.ResponsePacketID = binary.LittleEndian.Uint16(rpid)

	// PR Processing Result
	decoded.ProcessingResult, err = buffer.ReadByte()
	if err != nil {
		return fmt.Errorf("EGTS_PT_RESPONSE; Error reading PR: %w", err)
	}

	// SFRD (Services Frame Data)
	if buffer.Len() > 0 {
		decoded.SDR = &ServicesFrameData{}
		err = decoded.SDR.Decode(buffer.Bytes())
		if err != nil {
			return fmt.Errorf("EGTS_PT_RESPONSE; %w", err)
		}
	}
	*response = decoded
	return nil
}

// Encode Parse EGTS_PT_RESPONSE to slice of bytes
func (response *PTResponse) Encode() (b []byte, err error) {
	if response == nil {
		return nil, fmt.Errorf("EGTS_PT_RESPONSE; Response is nil")
	}
	buffer := new(bytes.Buffer)
	err = binary.Write(buffer, binary.LittleEndian, response.ResponsePacketID)
	if err != nil {
		return nil, fmt.Errorf("EGTS_PT_RESPONSE; Error writing RPID: %w", err)
	}

	err = buffer.WriteByte(response.ProcessingResult)
	if err != nil {
		return nil, fmt.Errorf("EGTS_PT_RESPONSE; Error writing PR: %w", err)
	}
	if response.SDR != nil {
		data, ok := response.SDR.(*ServicesFrameData)
		if !ok || data == nil {
			return nil, fmt.Errorf("EGTS_PT_RESPONSE; SDR requires ServicesFrameData")
		}
		sdr, err := response.SDR.Encode()
		if err != nil {
			return nil, fmt.Errorf("EGTS_PT_RESPONSE; %w", err)
		}
		if len(sdr)+3 > 65535 {
			return nil, fmt.Errorf("EGTS_PT_RESPONSE; Response data exceeds 65535 bytes")
		}
		_, err = buffer.Write(sdr)
		if err != nil {
			return nil, fmt.Errorf("EGTS_PT_RESPONSE; Error writing SFRD: %w", err)
		}
	}
	return buffer.Bytes(), nil
}

// Len Returns length of bytes slice
func (response *PTResponse) Len() (l uint16) {
	encoded, _ := response.Encode()
	l = uint16(len(encoded))
	return l
}
