package server

import (
	"bufio"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/LdDl/go-egts/gateway/configuration"
	"github.com/LdDl/go-egts/gateway/destination"
)

type dumpHeader struct {
	Version int `json:"version"`
}

type dumpRecord struct {
	ReceivedAt time.Time          `json:"received_at"`
	Source     destination.Source `json:"source"`
	Raw        []byte             `json:"raw"`
	Stdout     bool               `json:"pending_stdout"`
	File       bool               `json:"pending_file"`
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
	if err != nil || header.Version != 1 {
		return fmt.Errorf("Invalid packet dump version")
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
		if (!saved.Stdout && !saved.File) || (saved.Stdout && !s.cfg.DestinationsCfg.Stdout) || (saved.File && !s.cfg.DestinationsCfg.File.Enabled) {
			return fmt.Errorf("Packet dump requires an unavailable destination")
		}
		record, err := destination.NewRecord(saved.ReceivedAt, saved.Source, saved.Raw)
		if err != nil {
			return fmt.Errorf("Invalid packet in dump: %w", err)
		}
		pending = append(pending, &pendingPacket{record: record, stdout: saved.Stdout, file: saved.File, done: make(chan struct{})})
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
		if item.stdout || item.file {
			pending = append(pending, dumpRecord{ReceivedAt: item.record.ReceivedAt, Source: item.record.Source, Raw: item.record.Raw, Stdout: item.stdout, File: item.file})
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
	err = encoder.Encode(dumpHeader{Version: 1})
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
