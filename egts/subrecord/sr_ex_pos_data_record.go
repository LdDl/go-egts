package subrecord

import (
	"encoding/binary"

	"github.com/LdDl/go-egts/egts/utils"
)

//EgtsSrExPosDataRecord - Используется абонентским терминалом
//при передаче дополнительных данных определения местоположения
type EgtsSrExPosDataRecord struct {
	VerticalDiluptionOfPrecision   uint16 /* Vertical Dilution of Precision */
	HorizontalDiluptionOfPrecision uint16 /* Horizontal Dilution of Precision */
	PositionDiluptionOfPrecision   uint16 /* Position Dilution of Precision */
	Satellites                     uint8  /* Satellites */
	NavigationSystem               uint16 /* Navigation System */
}

//ParseEgtsSrExPosData - EGTS_SR_EXT_POS_DATA
func ParseEgtsSrExPosData(b []byte) interface{} {
	var d EgtsSrExPosDataRecord
	// Flags
	flagBytes := uint16(b[0])
	VDOP := utils.BitField(flagBytes, 0).(bool)
	HDOP := utils.BitField(flagBytes, 1).(bool)
	PDOP := utils.BitField(flagBytes, 2).(bool)
	SFE := utils.BitField(flagBytes, 3).(bool)
	NSFE := utils.BitField(flagBytes, 4).(bool)
	n := 1
	if VDOP {
		d.VerticalDiluptionOfPrecision = binary.LittleEndian.Uint16(b[n : n+2])
		n += 2
	}
	if HDOP {
		d.HorizontalDiluptionOfPrecision = binary.LittleEndian.Uint16(b[n : n+2])
		n += 2
	}
	if PDOP {
		d.PositionDiluptionOfPrecision = binary.LittleEndian.Uint16(b[n : n+2])
		n += 2
	}
	if SFE {
		d.Satellites = uint8(b[n])
		n++
	}
	if NSFE {
		d.NavigationSystem = binary.LittleEndian.Uint16(b[n : n+2])
		n += 2
	}
	/*
		NS:
		0	- система не определена;
		1 - ГЛОНАСС;
		2 - GPS;
		4 - Galileo;
		8 - Compass;
		16 - Beidou;
		32 - DORIS;
		64 - IRNSS;
		128 - QZSS.
		Остальные значения зарезервированы.
	*/
	return d
}
