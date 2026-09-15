package server

import (
	"bufio"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"time"

	"github.com/LdDl/go-egts/egts/subrecord"
	"github.com/LdDl/go-egts/gateway/configuration"
	"github.com/LdDl/go-egts/gateway/destination"
)

type dumpHeader struct {
	Version int `json:"version"`
}

type dumpRecord struct {
	ReceivedAt  time.Time          `json:"received_at"`
	Source      destination.Source `json:"source"`
	Raw         []byte             `json:"raw"`
	Stdout      bool               `json:"pending_stdout"`
	File        bool               `json:"pending_file"`
	EGTS        bool               `json:"pending_egts"`
	EGTSIDs     []string           `json:"pending_egts_ids"`
	RabbitMQIDs []string           `json:"pending_rabbitmq_ids"`
	SessionID   *string            `json:"session_id"`
	Identity    []byte             `json:"identity"`
}

func (s *server) restoreDump() (err error) {
	err = s.cfg.ValidateOutputs(os.Stdout, os.Stderr)
	if err != nil {
		return err
	}
	file, err := os.Open(filepath.Join(s.cfg.DeliveryCfg.DumpDirectory, configuration.DUMP_FILENAME))
	if errors.Is(err, os.ErrNotExist) {
		return nil
	}
	if err != nil {
		return fmt.Errorf("Can't open packet dump: %w", err)
	}
	defer func() {
		closeErr := file.Close()
		if closeErr != nil && err == nil {
			err = closeErr
		}
	}()
	scanner := bufio.NewScanner(file)
	scanner.Buffer(make([]byte, 4096), 131072)
	if !scanner.Scan() {
		return fmt.Errorf("Packet dump has no header")
	}
	var header dumpHeader
	err = json.Unmarshal(scanner.Bytes(), &header)
	if err != nil || (header.Version != 1 && header.Version != 2 && header.Version != 3 && header.Version != 4) {
		return fmt.Errorf("Invalid packet dump version")
	}
	enabled := make(map[string]bool)
	for _, cfg := range s.cfg.DestinationsCfg.EGTS {
		if cfg.Enabled {
			enabled[cfg.ID] = true
		}
	}
	enabledRabbits := make(map[string]bool)
	for _, cfg := range s.cfg.DestinationsCfg.RabbitMQ {
		if cfg.Enabled {
			enabledRabbits[cfg.ID] = true
		}
	}
	var pending []*pendingPacket
	for scanner.Scan() {
		if len(pending) >= s.cfg.DeliveryCfg.QueueCapacity {
			return fmt.Errorf("Packet dump exceeds queue_capacity; increase it to restore pending packets")
		}
		var saved dumpRecord
		err = json.Unmarshal(scanner.Bytes(), &saved)
		if err != nil {
			return fmt.Errorf("Invalid packet dump record: %w", err)
		}
		if header.Version < 3 && len(saved.EGTSIDs) > 0 {
			return fmt.Errorf("Invalid EGTS destinations in legacy packet dump")
		}
		if saved.EGTS {
			if header.Version != 2 || len(enabled) != 1 {
				return fmt.Errorf("Legacy relay dump requires exactly one enabled EGTS destination to restore")
			}
			for id := range enabled {
				saved.EGTSIDs = []string{id}
			}
		}
		if (!saved.Stdout && !saved.File && len(saved.EGTSIDs) == 0 && len(saved.RabbitMQIDs) == 0) || (saved.Stdout && !s.cfg.DestinationsCfg.Stdout) || (saved.File && !s.cfg.DestinationsCfg.File.Enabled) {
			return fmt.Errorf("Packet dump requires an unavailable destination")
		}
		var relayIDs map[string]bool
		if len(saved.EGTSIDs) > 0 {
			relayIDs = make(map[string]bool)
		}
		for _, id := range saved.EGTSIDs {
			if !enabled[id] {
				return fmt.Errorf("Packet dump requires an unavailable destination: EGTS %s", id)
			}
			if relayIDs[id] {
				return fmt.Errorf("Duplicate EGTS destination in packet dump: %s", id)
			}
			relayIDs[id] = true
		}
		if header.Version < 4 && len(saved.RabbitMQIDs) > 0 {
			return fmt.Errorf("Invalid RabbitMQ destinations in legacy packet dump")
		}
		var rabbitIDs map[string]bool
		if len(saved.RabbitMQIDs) > 0 {
			rabbitIDs = make(map[string]bool)
		}
		for _, id := range saved.RabbitMQIDs {
			if !enabledRabbits[id] {
				return fmt.Errorf("Packet dump requires an unavailable destination: RabbitMQ %s", id)
			}
			if rabbitIDs[id] {
				return fmt.Errorf("Duplicate RabbitMQ destination in packet dump: %s", id)
			}
			rabbitIDs[id] = true
		}
		record, err := destination.NewRecord(saved.ReceivedAt, saved.Source, saved.Raw)
		if err != nil {
			return fmt.Errorf("Invalid packet in dump: %w", err)
		}
		item := &pendingPacket{record: record, stdout: saved.Stdout, file: saved.File, egts: relayIDs, rabbitmq: rabbitIDs, done: make(chan struct{})}
		if len(relayIDs) > 0 {
			if saved.SessionID == nil || len(*saved.SessionID) != 32 {
				return fmt.Errorf("Invalid relay session in packet dump")
			}
			_, err = hex.DecodeString(*saved.SessionID)
			if err != nil {
				return fmt.Errorf("Invalid relay session identifier")
			}
			if len(saved.Identity) > 0 {
				identity := subrecord.SRTermIdentity{}
				err = identity.Decode(saved.Identity)
				if err != nil {
					return fmt.Errorf("Invalid relay identity in packet dump: %w", err)
				}
			}
			forward, err := shouldRelay(&record.Packet)
			if err != nil || !forward {
				return fmt.Errorf("Packet dump contains invalid relay data")
			}
			session := s.relays[*saved.SessionID]
			if session == nil {
				session, err = s.newRelaySession()
				if err != nil {
					return err
				}
				delete(s.relays, session.id)
				session.id = *saved.SessionID
				session.closed = true
				s.relays[session.id] = session
			}
			session.pending += len(relayIDs)
			item.relay = &relayPacket{session: session, identity: saved.Identity}
		}
		pending = append(pending, item)
	}
	err = scanner.Err()
	if err != nil {
		return fmt.Errorf("Can't read packet dump: %w", err)
	}
	s.queue = pending
	s.hasDump = true
	return nil
}

func (s *server) saveDump() (err error) {
	s.mu.Lock()
	var pending []dumpRecord
	for _, item := range s.queue {
		if item.stdout || item.file || len(item.egts) > 0 || len(item.rabbitmq) > 0 {
			saved := dumpRecord{ReceivedAt: item.record.ReceivedAt, Source: item.record.Source, Raw: item.record.Raw, Stdout: item.stdout, File: item.file}
			if len(item.egts) > 0 {
				for id := range item.egts {
					saved.EGTSIDs = append(saved.EGTSIDs, id)
				}
				sort.Strings(saved.EGTSIDs)
				saved.SessionID = &item.relay.session.id
				saved.Identity = item.relay.identity
			}
			for id := range item.rabbitmq {
				saved.RabbitMQIDs = append(saved.RabbitMQIDs, id)
			}
			sort.Strings(saved.RabbitMQIDs)
			pending = append(pending, saved)
		}
	}
	s.mu.Unlock()
	err = s.cfg.ValidateOutputs(os.Stdout, os.Stderr)
	if err != nil {
		return err
	}
	file, err := os.CreateTemp(s.cfg.DeliveryCfg.DumpDirectory, "queue-*.tmp")
	if err != nil {
		return fmt.Errorf("Can't create packet dump: %w", err)
	}
	name := file.Name()
	defer func() {
		if file != nil {
			closeErr := file.Close()
			if closeErr != nil && err == nil {
				err = closeErr
			}
		}
		removeErr := os.Remove(name)
		if removeErr != nil && !errors.Is(removeErr, os.ErrNotExist) && err == nil {
			err = removeErr
		}
	}()
	encoder := json.NewEncoder(file)
	version := 1
	for _, cfg := range s.cfg.DestinationsCfg.EGTS {
		if cfg.Enabled {
			version = 3
			break
		}
	}
	for _, cfg := range s.cfg.DestinationsCfg.RabbitMQ {
		if cfg.Enabled {
			version = 4
			break
		}
	}
	err = encoder.Encode(dumpHeader{Version: version})
	if err != nil {
		return fmt.Errorf("Can't write packet dump header: %w", err)
	}
	for _, record := range pending {
		err = encoder.Encode(record)
		if err != nil {
			return fmt.Errorf("Can't write packet dump: %w", err)
		}
	}
	err = file.Sync()
	if err != nil {
		return fmt.Errorf("Can't sync packet dump: %w", err)
	}
	err = file.Close()
	file = nil
	if err != nil {
		return err
	}
	err = s.cfg.ValidateOutputs(os.Stdout, os.Stderr)
	if err != nil {
		return err
	}
	err = os.Rename(name, filepath.Join(s.cfg.DeliveryCfg.DumpDirectory, configuration.DUMP_FILENAME))
	if err != nil {
		return fmt.Errorf("Can't replace packet dump: %w", err)
	}
	s.hasDump = true
	directory, err := os.Open(s.cfg.DeliveryCfg.DumpDirectory)
	if err != nil {
		return err
	}
	err = directory.Sync()
	closeErr := directory.Close()
	if err != nil {
		return fmt.Errorf("Can't sync packet dump directory: %w", err)
	}
	return closeErr
}
