package packet

import (
	"encoding/binary"
	"encoding/json"
	"fmt"
)

//PTResponse структура подзаписи типа EGTS_PT_RESPONSE
type PTResponse struct {
	ResponsePacketID uint16    `json:"RPID"` // RPID Response Packet ID
	ProcessingResult uint8     `json:"PR"`   // PR Processing Result
	SDR              BytesData `json:"SFRD"` // SFRD (Services Frame Data)
}

func (p *Packet) GetPTResponse() (ptResp PTResponse) {
	ptResp.ResponsePacketID = p.PacketID
	ptResp.ProcessingResult = p.Error

	str, _ := json.Marshal(ptResp)
	fmt.Println("resp:")
	fmt.Println(string(str))
	return ptResp
}

// Decode Parse array of bytes to EGTS_PT_RESPONSE
func (response *PTResponse) Decode(b []byte) {
	response.ResponsePacketID = binary.LittleEndian.Uint16(b[0:2])
	response.ProcessingResult = uint8(b[2])

	if len(b[3:]) > 0 {
		response.SDR = &ServicesFrameData{}
		response.SDR.Decode(b[3:])
	}
}

// Encode Parse EGTS_PT_RESPONSE to array of bytes
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
