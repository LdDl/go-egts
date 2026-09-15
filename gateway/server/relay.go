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
	writers map[string]*destination.EGTS
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
	session := &relaySession{id: hex.EncodeToString(random), writers: make(map[string]*destination.EGTS)}
	for _, cfg := range s.cfg.DestinationsCfg.EGTS {
		if !cfg.Enabled {
			continue
		}
		writer, err := destination.PrepareEGTS(cfg, s.cfg.ServerCfg)
		if err != nil {
			return nil, err
		}
		session.writers[cfg.ID] = writer
	}
	if len(session.writers) == 0 {
		return nil, nil
	}
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
		return session.close()
	}
	return nil
}

func (session *relaySession) close() error {
	var result error
	for id, writer := range session.writers {
		err := writer.Close()
		if err != nil {
			if result == nil {
				result = fmt.Errorf("Can't close EGTS destination %s: %w", id, err)
			} else {
				result = fmt.Errorf("%w; Can't close EGTS destination %s: %v", result, id, err)
			}
		}
	}
	return result
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
