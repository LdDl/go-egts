package packet

import "fmt"

// Print - prints packet's header and data
func (p *Packet) Print() {
	fmt.Println("\t\tPacket")
	fmt.Println("\tHeader")
	fmt.Println("Protocol Version: ", p.ProtocolVersion)
	fmt.Println("Security Key ID: ", p.SecurityKeyID)
	fmt.Println("Flag: PR ", p.PR, " CMP ", p.CMP, " ENA ", p.ENA, " RTE ", p.RTE, " PRF ", p.PRF)
	fmt.Println("Header Length: ", p.HeaderLength)
	fmt.Println("Header Encoding: ", p.HeaderEncoding)
	fmt.Println("Frame Data Length: ", p.FrameDataLength)
	fmt.Println("Packet ID: ", p.PacketID)
	fmt.Println("Packet Type: ", p.PacketType)
	// fmt.Println("Peer Address: ", p.PeerAddress)
	// fmt.Println("Recipient Address: ", p.RecipientAddress)
	// fmt.Println("Time To Live: ", p.TimeToLive)
	fmt.Println("Header Check Sum: ", p.HeaderCheckSum)
	fmt.Println("\tServices Frame Data Struct")
	for i := 0; i < len(p.ServicesFrameData); i++ {
		fmt.Println("Record Length: ", p.ServicesFrameData[i].RecordLength)
		fmt.Println("Record Number: ", p.ServicesFrameData[i].RecordNumber)
		fmt.Println("Object Identifier: ", p.ServicesFrameData[i].ObjectIdentifier)
		fmt.Println("Event Identifier: ", p.ServicesFrameData[i].EventIdentifier)
		fmt.Println("Time: ", p.ServicesFrameData[i].Time)
		fmt.Println("Source Service Type: ", p.ServicesFrameData[i].SourceServiceType)
		fmt.Println("Recipient Service Type: ", p.ServicesFrameData[i].RecipientServiceType)
		fmt.Println("Record Data: ", p.ServicesFrameData[i].RecordData)

		fmt.Println("\tRecordData Struct")
		fmt.Println("Subrecord Type: ", p.ServicesFrameData[i].RecordData.SubrecordType)
		fmt.Println("Subrecord Length: ", p.ServicesFrameData[i].RecordData.SubrecordLength)
		fmt.Println("Subrecord Data: ", p.ServicesFrameData[i].RecordData.SubrecordData)

		fmt.Println("\tSubrecord Data Struct: ", p.ServicesFrameData[i].RecordData.SubrecordData)
	}

	fmt.Println("Services Frame Data Check Sum: ", p.ServicesFrameDataCheckSum)
}
