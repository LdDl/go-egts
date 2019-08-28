package packet

import "fmt"

// type ServicesFrameData []*ServiceDataRecord

// ServiceDataRecord - формат отдельной записи Протокола Уровня Поддержки Услуг.
type ServiceDataRecord struct {
	RecordLength uint16 // RL (Record Length)
	RecordNumber uint16 // RN (Record Number)
	/* RecordFlags (RFL): SSOD, RSOD, GRP, RPP, TMFE, EVFE, OBFE */
	SSOD bool // SSOD
	RSOD bool // RSOD
	GRP  bool // GRP
	RPP  int  // RPP
	TMFE bool // TMFE
	EVFE bool // EVFE
	OBFE bool // OBFE
	/*                                                          */
	ObjectIdentifier     uint32     // OID (Object Identifier)
	EventIdentifier      uint32     // EVID (Event Identifier)
	Time                 uint32     // TM (Time)
	SourceServiceType    uint8      // SST (Source Service Type)
	RecipientServiceType uint8      // RST (Recipient Service Type)
	RecordData           RecordData // RD (Record Data)
}

func (sdr ServiceDataRecord) String() string {
	return fmt.Sprintf("\n\tSDR:\n\tRL (Record Length): %v\n\tRN (Record Number): %v\n\tRFL (Record Flags):\n\t\tSSOD: %v\n\t\tRSOD: %v\n\t\tGRP: %v\n\t\tRPP: %v\n\t\tTMFE: %v\n\t\tEVFE: %v\n\t\tOBFE: %v\n\tOID (Object Identifier): %v\n\tEVID (Event Identifier): %v\n\tTM (Time): %v\n\tSST (Source Service Type): %v\n\tRST (Recipient Service Type): %v\n\tRD (Record Data): %v\n\t",
		sdr.RecordLength,
		sdr.RecordNumber,
		sdr.SSOD,
		sdr.RSOD,
		sdr.GRP,
		sdr.RPP,
		sdr.TMFE,
		sdr.EVFE,
		sdr.OBFE,
		sdr.ObjectIdentifier,
		sdr.EventIdentifier,
		sdr.Time,
		sdr.SourceServiceType,
		sdr.RecipientServiceType,
		sdr.RecordData,
	)
}
