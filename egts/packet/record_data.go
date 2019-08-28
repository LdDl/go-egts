package packet

import "fmt"

// RecordData - формат отдельной подзаписи Протокола Уровня Поддержки Услуг.
type RecordData struct {
	SubrecordType   uint8       // SRT (Subrecord Туре)
	SubrecordLength uint16      // SRL(Subrecord Length)
	SubrecordData   interface{} // SRD (Subrecord Data)
}

func (rd RecordData) String() string {
	return fmt.Sprintf("\n\t\tSRT (Subrecord Туре): %v\n\t\tSRL(Subrecord Length): %v\n\t\tSRD (Subrecord Data):%v\n",
		rd.SubrecordType,
		rd.SubrecordLength,
		rd.SubrecordData,
	)
}
