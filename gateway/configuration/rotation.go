package configuration

import (
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"sync"
	"time"
)

const archiveTimeFormat = "20060102T150405.000000000Z"

type LogFile struct {
	mu       sync.Mutex
	cfg      Configuration
	file     *os.File
	size     int64
	closed   bool
	writeErr error
}

type logArchive struct {
	path string
	time time.Time
	info os.FileInfo
}

func (cfg RotationConf) Validate() error {
	if cfg.MaxFileSizeBytes < 1 {
		return fmt.Errorf("logs_cfg.rotation.max_file_size_bytes must be positive")
	}
	if cfg.MaxBackups < 0 {
		return fmt.Errorf("logs_cfg.rotation.max_backups must not be negative")
	}
	if cfg.MaxAgeDays < 0 || cfg.MaxAgeDays > 106751 {
		return fmt.Errorf("logs_cfg.rotation.max_age_days must be between 0 and 106751")
	}
	if cfg.MaxTotalSizeBytes < cfg.MaxFileSizeBytes {
		return fmt.Errorf("logs_cfg.rotation.max_total_size_bytes must be at least max_file_size_bytes")
	}
	return nil
}

func (l *LogFile) Write(data []byte) (int, error) {
	l.mu.Lock()
	defer l.mu.Unlock()
	if l.closed {
		return 0, os.ErrClosed
	}
	if len(data) == 0 {
		return 0, nil
	}
	length := int64(len(data))
	if length > l.cfg.LogsCfg.Rotation.MaxFileSizeBytes {
		err := fmt.Errorf("Log record size %d exceeds max_file_size_bytes", length)
		l.writeErr = err
		return 0, err
	}
	if l.file == nil {
		err := l.open()
		if err != nil {
			l.writeErr = err
			return 0, err
		}
	}
	info, err := l.file.Stat()
	if err != nil {
		l.writeErr = err
		return 0, err
	}
	l.size = info.Size()
	if l.size > l.cfg.LogsCfg.Rotation.MaxFileSizeBytes-length {
		err := l.rotate()
		if err != nil {
			l.writeErr = err
			return 0, err
		}
	}
	err = l.cleanup(length)
	if err != nil {
		l.writeErr = err
		return 0, err
	}
	n, err := l.file.Write(data)
	l.size += int64(n)
	if err == nil && n != len(data) {
		err = io.ErrShortWrite
	}
	if err != nil {
		l.writeErr = err
	}
	return n, err
}

func (l *LogFile) Close() error {
	l.mu.Lock()
	defer l.mu.Unlock()
	if l.closed {
		return l.writeErr
	}
	l.closed = true
	if l.file == nil {
		return l.writeErr
	}
	err := l.cleanup(0)
	closeErr := l.file.Close()
	l.file = nil
	if err != nil {
		l.writeErr = err
	}
	if closeErr != nil {
		if l.writeErr != nil {
			l.writeErr = fmt.Errorf("%w; Can't close application log: %v", l.writeErr, closeErr)
		} else {
			l.writeErr = closeErr
		}
	}
	return l.writeErr
}

func (l *LogFile) open() error {
	err := l.cfg.ValidateOutputs(os.Stdout, os.Stderr)
	if err != nil {
		return err
	}
	name := filepath.Join(l.cfg.LogsCfg.Directory, LOG_FILENAME)
	file, err := os.OpenFile(name, os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0640)
	if err != nil {
		return fmt.Errorf("Can't open application log: %w", err)
	}
	info, err := file.Stat()
	if err == nil {
		err = l.cfg.ValidateOutputs(os.Stdout, os.Stderr)
	}
	var archives []logArchive
	if err == nil {
		archives, err = l.archives()
	}
	if err == nil {
		for _, archive := range archives {
			if !os.SameFile(info, archive.info) {
				continue
			}
			err = os.Remove(archive.path)
			if err != nil {
				break
			}
		}
	}
	if err != nil {
		closeErr := file.Close()
		if closeErr != nil {
			return fmt.Errorf("%w; Can't close application log: %v", err, closeErr)
		}
		return err
	}
	l.file = file
	l.size = info.Size()
	return nil
}

func (l *LogFile) rotate() error {
	err := l.cfg.ValidateOutputs(os.Stdout, os.Stderr)
	if err != nil {
		return err
	}
	archives, err := l.archives()
	if err != nil {
		return err
	}
	timestamp := time.Now().UTC()
	if len(archives) > 0 && !timestamp.After(archives[0].time) {
		timestamp = archives[0].time.Add(time.Nanosecond)
	}
	name := filepath.Join(l.cfg.LogsCfg.Directory, LOG_FILENAME)
	info, err := os.Stat(name)
	if err != nil {
		return fmt.Errorf("Can't inspect application log before rotation: %w", err)
	}
	openedInfo, err := l.file.Stat()
	if err != nil {
		return err
	}
	if !os.SameFile(info, openedInfo) {
		return fmt.Errorf("Application log was replaced while open")
	}
	err = l.file.Sync()
	if err != nil {
		return fmt.Errorf("Can't sync application log before rotation: %w", err)
	}
	var archive string
	for {
		archive = filepath.Join(l.cfg.LogsCfg.Directory, "application-"+timestamp.Format(archiveTimeFormat)+".log")
		// Link fails if the archive already exists, preserving existing data.
		err = os.Link(name, archive)
		if errors.Is(err, os.ErrExist) {
			timestamp = timestamp.Add(time.Nanosecond)
			continue
		}
		if err != nil {
			return fmt.Errorf("Can't archive application log: %w", err)
		}
		break
	}
	err = os.Remove(name)
	if err != nil {
		removeErr := os.Remove(archive)
		if removeErr != nil {
			return fmt.Errorf("%w; Can't remove unfinished archive: %v", err, removeErr)
		}
		return fmt.Errorf("Can't remove active application log after archiving: %w", err)
	}
	err = l.file.Close()
	l.file = nil
	l.size = 0
	if err != nil {
		return fmt.Errorf("Can't close archived application log: %w", err)
	}
	return l.open()
}

func (l *LogFile) archives() ([]logArchive, error) {
	entries, err := os.ReadDir(l.cfg.LogsCfg.Directory)
	if err != nil {
		return nil, fmt.Errorf("Can't read application log directory: %w", err)
	}
	var archives []logArchive
	for _, entry := range entries {
		name := entry.Name()
		if !strings.HasPrefix(name, "application-") || !strings.HasSuffix(name, ".log") {
			continue
		}
		timestamp := strings.TrimSuffix(strings.TrimPrefix(name, "application-"), ".log")
		created, err := time.Parse(archiveTimeFormat, timestamp)
		if err != nil {
			continue
		}
		path := filepath.Join(l.cfg.LogsCfg.Directory, name)
		info, err := os.Stat(path)
		if err != nil {
			return nil, fmt.Errorf("Can't inspect application log archive: %w", err)
		}
		if !info.Mode().IsRegular() {
			return nil, fmt.Errorf("Application log archive is not a regular file: %s", path)
		}
		archives = append(archives, logArchive{path: path, time: created, info: info})
	}
	sort.Slice(archives, func(i, j int) bool {
		return archives[i].time.After(archives[j].time)
	})
	return archives, nil
}

func (l *LogFile) cleanup(pendingBytes int64) error {
	err := l.cfg.ValidateOutputs(os.Stdout, os.Stderr)
	if err != nil {
		return err
	}
	archives, err := l.archives()
	if err != nil {
		return err
	}
	info, err := l.file.Stat()
	if err != nil {
		return err
	}
	l.size = info.Size()
	rotation := l.cfg.LogsCfg.Rotation
	remaining := rotation.MaxTotalSizeBytes - pendingBytes - l.size
	if remaining < 0 {
		return fmt.Errorf("Application log exceeds max_total_size_bytes")
	}
	cutoff := time.Now().UTC().Add(-time.Duration(rotation.MaxAgeDays) * 24 * time.Hour)
	kept := 0
	removeOlder := false
	for _, archive := range archives {
		// Recover an interruption between linking the archive and removing the active name.
		unfinished := os.SameFile(info, archive.info)
		expired := rotation.MaxAgeDays > 0 && archive.time.Before(cutoff)
		if !unfinished && (kept >= rotation.MaxBackups || archive.info.Size() > remaining || expired) {
			removeOlder = true
		}
		if unfinished || removeOlder {
			err = os.Remove(archive.path)
			if err != nil {
				return fmt.Errorf("Can't remove application log archive: %w", err)
			}
			continue
		}
		kept++
		remaining -= archive.info.Size()
	}
	return nil
}
