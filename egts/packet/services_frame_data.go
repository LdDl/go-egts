package packet

// ServicesFrameData SFRD (Services Frame Data)
type ServicesFrameData []*ServiceDataRecord

// ServiceDataRecord - формат отдельной записи Протокола Уровня Поддержки Услуг.
type ServiceDataRecord struct {
	RecordLength uint16 `json:"RL"` // RL (Record Length)
	RecordNumber uint16 `json:"RN"` // RN (Record Number)
	/* RecordFlags (RFL): SSOD, RSOD, GRP, RPP, TMFE, EVFE, OBFE */
	SSOD bool `json:"SSOD"` // SSOD Source Service On Device
	RSOD bool `json:"RSOD"` // RSOD Recipient Service On Device
	GRP  bool `json:"GRP"`  // GRP Group
	RPP  int  `json:"RPP"`  // RPP Record Processing Priority
	TMFE bool `json:"TMFE"` // TMFE Time Field Exists
	EVFE bool `json:"EVFE"` // EVFE Event ID Field Exists
	OBFE bool `json:"OBFE"` // OBFE Object ID FieldExists
	/*                                                          */
	ObjectIdentifier     uint32     `json:"OID"`  // OID (Object Identifier)
	EventIdentifier      uint32     `json:"EVID"` // EVID (Event Identifier)
	Time                 uint32     `json:"TM"`   // TM (Time)
	SourceServiceType    uint8      `json:"SST"`  // SST (Source Service Type)
	RecipientServiceType uint8      `json:"RST"`  // RST (Recipient Service Type)
	RecordData           RecordData `json:"RD"`   // RD (Record Data)
}

// ReadServicesFrameData - считывает данные поля SFRD - структура данных, зависящая от типа пакета и содержащая информацию
// Протокола уровня поддержки услуг
func (p *Packet) ReadServicesFrameData(b []byte) (sfrd ServicesFrameData, err uint8) {
	switch p.PacketType {
	case EGTS_PT_RESPONSE:
		ptResp := BytesToPTResponse(b)
		sfrd, err = ptResp.SDR, ptResp.ProcessingResult
		break
	case EGTS_PT_APPDATA:
		sfrd, err = BytesToPTAppData(b)
		break
	case EGTS_PT_SIGNED_APPDATA:
		// @todo
		break
	default:
		// nothing
		break
	}

	return
}
