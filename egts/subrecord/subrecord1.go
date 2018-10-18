package subrecord

import (
	"encoding/binary"

	"github.com/LdDl/go-egts/egts/utils"
)

//EgtsSrTermIdentity --
type EgtsSrTermIdentity struct {
	TerminalIdentifier                                  uint16 // TID
	HomeDispatcherIdentifier                            uint16 // HDID
	InternationalMobileEquipmentIdentity                string // IMEI
	InternationalMobileSubscriberIdentity               string // IMSI
	LanguageCode                                        string // LNGC
	NetworkIdentifier                                   uint16 // NID
	BufferSize                                          uint16 // BS
	MobileStationIntegratedServicesDigitalNetworkNumber string // MSISDN
	//Flags
	MNE   bool
	BSE   bool
	NIDE  bool
	SSRA  bool
	LNGCE bool
	IMSIE bool
	IMEIE bool
	HDIDE bool
}

//ParseEgtsSrTermIdentity - EGTS_SR_TERM_IDENTITY
//Подзапись EGTS_SR_TERM_IDENTITY сервиса EGTS_AUTH_SERVICE
//Code subrecord 1
func ParseEgtsSrTermIdentity(b []byte) interface{} {
	var d EgtsSrTermIdentity
	i := 0
	d.TerminalIdentifier = binary.LittleEndian.Uint16(b[i : i+4])
	i += 4
	// Flags
	flagBytes := uint16(b[i])
	i++
	d.HDIDE = utils.BitField(flagBytes, 0).(bool)
	d.IMEIE = utils.BitField(flagBytes, 1).(bool)
	d.IMSIE = utils.BitField(flagBytes, 2).(bool)
	d.LNGCE = utils.BitField(flagBytes, 3).(bool)
	d.SSRA = utils.BitField(flagBytes, 4).(bool)
	d.NIDE = utils.BitField(flagBytes, 5).(bool)
	d.BSE = utils.BitField(flagBytes, 6).(bool)
	d.MNE = utils.BitField(flagBytes, 7).(bool)
	if d.HDIDE {
		d.HomeDispatcherIdentifier = binary.LittleEndian.Uint16(b[i : i+2])
		i += 2
	}
	if d.IMEIE {
		d.InternationalMobileEquipmentIdentity = string(b[i : i+15])
		i += 15
	}
	if d.IMSIE {
		d.InternationalMobileSubscriberIdentity = string(b[i : i+16])
		i += 16
	}
	if d.LNGCE {
		d.LanguageCode = string(b[i : i+3])
		i += 3
	}
	if d.NIDE {
		d.NetworkIdentifier = binary.LittleEndian.Uint16(b[i : i+3])
		i += 3
	}
	if d.BSE {
		d.BufferSize = binary.LittleEndian.Uint16(b[i : i+2])
		i += 2
	}
	if d.MNE {
		d.MobileStationIntegratedServicesDigitalNetworkNumber = string(b[i : i+15])
		i += 15
	}
	return d
}
