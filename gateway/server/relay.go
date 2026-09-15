package server

import (
	"crypto/rand"
	"encoding/hex"
	"errors"
	"fmt"

	"github.com/LdDl/go-egts/egts/packet"
	"github.com/LdDl/go-egts/gateway/destination"
)

var ErrRelayCredentials = errors.New("A relayed data packet must not contain incoming authentication credentials")

type relaySession struct {
	id      string
	writer  *destination.EGTS
	pending int
	closed  bool
}

type relayPacket struct {
	session  *relaySession
	identity []byte
}

func (s *server) newRelaySession() (*relaySession, error) {
	random := make([]byte, 16)
	_, err := rand.Read(random)
	if err != nil {
		return nil, err
	}
	writer, err := destination.PrepareEGTS(&s.cfg)
	if err != nil {
		return nil, err
	}
	session := &relaySession{id: hex.EncodeToString(random), writer: writer}
	s.mu.Lock()
	s.relays[session.id] = session
	s.mu.Unlock()
	return session, nil
}

func (s *server) endRelaySession(session *relaySession) error {
	s.mu.Lock()
	session.closed = true
	finished := session.pending == 0
	if finished {
		delete(s.relays, session.id)
	}
	s.mu.Unlock()
	if finished {
		return session.writer.Close()
	}
	return nil
}

func shouldRelay(pkg *packet.Packet) (bool, error) {
	if pkg.PacketType != packet.EGTS_PT_APPDATA {
		return false, fmt.Errorf("EGTS relay requires an APPDATA packet")
	}
	services := pkg.ServicesFrameData.(*packet.ServicesFrameData)
	hasData := len(*services) == 0
	hasCredentials := false
	for _, service := range *services {
		if len(service.RecordsData) == 0 && (service.SourceServiceType != packet.SERVICE_AUTH || service.RecipientServiceType != packet.SERVICE_AUTH) {
			hasData = true
		}
		for _, record := range service.RecordsData {
			if record.SubrecordType == packet.AuthInfo {
				hasCredentials = true
			}
			if (service.SourceServiceType != packet.SERVICE_AUTH || service.RecipientServiceType != packet.SERVICE_AUTH) && record.SubrecordType != packet.RecordResponse {
				hasData = true
			}
		}
	}
	if hasData && hasCredentials {
		return false, ErrRelayCredentials
	}
	return hasData, nil
}
