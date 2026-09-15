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

type RotatingFile struct {
	mu        sync.Mutex
	cfg       Configuration
	directory string
	filename  string
	rotation  RotationConf
	file      *os.File
	size      int64
	closed    bool
	writeErr  error
}

type fileArchive struct {
	path string
	time time.Time
	info os.FileInfo
}

func PreparePacketFile(cfg *Configuration) (*RotatingFile, error) {
	if !cfg.DestinationsCfg.File.Enabled {
		return nil, fmt.Errorf("Packet file destination is disabled")
	}
	file, err := prepareRotatingFile(cfg, cfg.DestinationsCfg.File.Directory, PACKETS_FILENAME, cfg.DestinationsCfg.File.Rotation)
	if err != nil {
		return nil, fmt.Errorf("Can't prepare packet file: %w", err)
	}
	return file, nil
}

func prepareRotatingFile(cfg *Configuration, directory, filename string, rotation RotationConf) (*RotatingFile, error) {
	err := rotation.Validate()
	if err != nil {
		return nil, err
	}
	err = cfg.ValidateOutputs(os.Stdout, os.Stderr)
	if err != nil {
		return nil, err
	}
	if strings.TrimSpace(directory) == "" {
		return nil, fmt.Errorf("Output directory must not be empty")
	}
	directory, err = resolveDirectory(directory)
	if err != nil {
		return nil, fmt.Errorf("Can't resolve output directory: %w", err)
	}
	err = os.MkdirAll(directory, 0750)
	if err != nil {
		return nil, fmt.Errorf("Can't create output directory: %w", err)
	}
	file := &RotatingFile{cfg: *cfg, directory: directory, filename: filename, rotation: rotation}
	if filename == LOG_FILENAME {
		file.cfg.LogsCfg.Directory = directory
	} else {
		file.cfg.DestinationsCfg.File.Directory = directory
	}
	err = file.open()
	if err == nil && file.size > rotation.MaxFileSizeBytes {
		err = file.rotate()
	}
	if err == nil {
		err = file.cleanup(0)
	}
	if err != nil {
		file.writeErr = err
		closeErr := file.Close()
		return nil, closeErr
	}
	return file, nil
}

func (cfg RotationConf) Validate() error {
	if cfg.MaxFileSizeBytes < 1 {
		return fmt.Errorf("max_file_size_bytes must be positive")
	}
	if cfg.MaxBackups < 0 {
		return fmt.Errorf("max_backups must not be negative")
	}
	if cfg.MaxAgeDays < 0 || cfg.MaxAgeDays > 106751 {
		return fmt.Errorf("max_age_days must be between 0 and 106751")
	}
	if cfg.MaxTotalSizeBytes < cfg.MaxFileSizeBytes {
		return fmt.Errorf("max_total_size_bytes must be at least max_file_size_bytes")
	}
	return nil
}

func (l *RotatingFile) Write(data []byte) (int, error) {
	l.mu.Lock()
	defer l.mu.Unlock()
	if l.closed {
		return 0, os.ErrClosed
	}
	if len(data) == 0 {
		return 0, nil
	}
	length := int64(len(data))
	if length > l.rotation.MaxFileSizeBytes {
		err := fmt.Errorf("Record size %d exceeds max_file_size_bytes", length)
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
	if l.size > l.rotation.MaxFileSizeBytes-length {
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

func (l *RotatingFile) Close() error {
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
			l.writeErr = fmt.Errorf("%w; Can't close output file: %v", l.writeErr, closeErr)
		} else {
			l.writeErr = closeErr
		}
	}
	return l.writeErr
}

func (l *RotatingFile) open() error {
	err := l.cfg.ValidateOutputs(os.Stdout, os.Stderr)
	if err != nil {
		return err
	}
	name := filepath.Join(l.directory, l.filename)
	file, err := os.OpenFile(name, os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0640)
	if err != nil {
		return fmt.Errorf("Can't open output file: %w", err)
	}
	info, err := file.Stat()
	if err == nil {
		err = l.cfg.ValidateOutputs(os.Stdout, os.Stderr)
	}
	var archives []fileArchive
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
			return fmt.Errorf("%w; Can't close output file: %v", err, closeErr)
		}
		return err
	}
	l.file = file
	l.size = info.Size()
	return nil
}

func (l *RotatingFile) rotate() error {
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
	name := filepath.Join(l.directory, l.filename)
	info, err := os.Stat(name)
	if err != nil {
		return fmt.Errorf("Can't inspect output file before rotation: %w", err)
	}
	openedInfo, err := l.file.Stat()
	if err != nil {
		return err
	}
	if !os.SameFile(info, openedInfo) {
		return fmt.Errorf("Output file was replaced while open")
	}
	err = l.file.Sync()
	if err != nil {
		return fmt.Errorf("Can't sync output file before rotation: %w", err)
	}
	var archive string
	for {
		archive = filepath.Join(l.directory, strings.TrimSuffix(l.filename, filepath.Ext(l.filename))+"-"+timestamp.Format(archiveTimeFormat)+filepath.Ext(l.filename))
		// Link fails if the archive already exists, preserving existing data.
		err = os.Link(name, archive)
		if errors.Is(err, os.ErrExist) {
			timestamp = timestamp.Add(time.Nanosecond)
			continue
		}
		if err != nil {
			return fmt.Errorf("Can't archive output file: %w", err)
		}
		break
	}
	err = os.Remove(name)
	if err != nil {
		removeErr := os.Remove(archive)
		if removeErr != nil {
			return fmt.Errorf("%w; Can't remove unfinished archive: %v", err, removeErr)
		}
		return fmt.Errorf("Can't remove active output file after archiving: %w", err)
	}
	err = l.file.Close()
	l.file = nil
	l.size = 0
	if err != nil {
		return fmt.Errorf("Can't close archived output file: %w", err)
	}
	return l.open()
}

func (l *RotatingFile) archives() ([]fileArchive, error) {
	entries, err := os.ReadDir(l.directory)
	if err != nil {
		return nil, fmt.Errorf("Can't read output file directory: %w", err)
	}
	var archives []fileArchive
	extension := filepath.Ext(l.filename)
	prefix := strings.TrimSuffix(l.filename, extension) + "-"
	for _, entry := range entries {
		name := entry.Name()
		if !strings.HasPrefix(name, prefix) || !strings.HasSuffix(name, extension) {
			continue
		}
		timestamp := strings.TrimSuffix(strings.TrimPrefix(name, prefix), extension)
		created, err := time.Parse(archiveTimeFormat, timestamp)
		if err != nil {
			continue
		}
		path := filepath.Join(l.directory, name)
		info, err := os.Stat(path)
		if err != nil {
			return nil, fmt.Errorf("Can't inspect output file archive: %w", err)
		}
		if !info.Mode().IsRegular() {
			return nil, fmt.Errorf("Output file archive is not a regular file: %s", path)
		}
		archives = append(archives, fileArchive{path: path, time: created, info: info})
	}
	sort.Slice(archives, func(i, j int) bool {
		return archives[i].time.After(archives[j].time)
	})
	return archives, nil
}

func (l *RotatingFile) cleanup(pendingBytes int64) error {
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
	rotation := l.rotation
	remaining := rotation.MaxTotalSizeBytes - pendingBytes - l.size
	if remaining < 0 {
		return fmt.Errorf("Output file exceeds max_total_size_bytes")
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
				return fmt.Errorf("Can't remove output file archive: %w", err)
			}
			continue
		}
		kept++
		remaining -= archive.info.Size()
	}
	return nil
}
