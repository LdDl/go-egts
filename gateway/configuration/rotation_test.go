package configuration

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
	"github.com/stretchr/testify/assert"
)

type rotationRecordsTestCase struct {
	name      string
	backups   int
	total     int64
	restart   bool
	expected  string
	fileCount int
}

type rotationValidationTestCase struct {
	name     string
	settings RotationConf
	valid    bool
}

func TestLogRotationRecords(t *testing.T) {
	previousLogger := log.Logger
	previousHandler := zerolog.ErrorHandler
	t.Cleanup(func() {
		log.Logger = previousLogger
		zerolog.ErrorHandler = previousHandler
	})
	cases := []rotationRecordsTestCase{
		{name: "whole records at size boundary", backups: 5, total: 60, expected: "aaaa\nbbbb\ncccc\ndddd\neeee\n", fileCount: 3},
		{name: "archive count", backups: 1, total: 60, expected: "cccc\ndddd\neeee\n", fileCount: 2},
		{name: "no archives", backups: 0, total: 60, expected: "eeee\n", fileCount: 1},
		{name: "total at boundary", backups: 5, total: 15, expected: "cccc\ndddd\neeee\n", fileCount: 2},
		{name: "total below next archive size", backups: 5, total: 14, expected: "eeee\n", fileCount: 1},
		{name: "restart appends", backups: 5, total: 60, restart: true, expected: "aaaa\nbbbb\ncccc\ndddd\neeee\n", fileCount: 3},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			root := t.TempDir()
			cfg := DefaultConfiguration()
			cfg.LogsCfg.Output = "file"
			cfg.LogsCfg.Directory = filepath.Join(root, "logs")
			cfg.DeliveryCfg.DumpDirectory = filepath.Join(root, "queue")
			cfg.LogsCfg.Rotation = RotationConf{MaxFileSizeBytes: 10, MaxBackups: tc.backups, MaxTotalSizeBytes: tc.total}
			file, err := PrepareLogger(cfg)
			assert.NoError(t, err)
			if err != nil {
				return
			}
			for i, record := range []string{"aaaa\n", "bbbb\n", "cccc\n", "dddd\n", "eeee\n"} {
				n, err := file.Write([]byte(record))
				assert.NoError(t, err)
				assert.Equal(t, len(record), n)
				if tc.restart && i == 2 {
					err = file.Close()
					assert.NoError(t, err)
					file, err = PrepareLogger(cfg)
					assert.NoError(t, err)
					if err != nil {
						return
					}
				}
			}
			err = file.Close()
			assert.NoError(t, err)
			err = file.Close()
			assert.NoError(t, err)
			entries, err := os.ReadDir(cfg.LogsCfg.Directory)
			assert.NoError(t, err)
			assert.Len(t, entries, tc.fileCount)
			var content strings.Builder
			var total int64
			for _, entry := range entries {
				data, err := os.ReadFile(filepath.Join(cfg.LogsCfg.Directory, entry.Name()))
				assert.NoError(t, err)
				assert.LessOrEqual(t, len(data), 10)
				assert.True(t, strings.HasSuffix(string(data), "\n"))
				content.Write(data)
				total += int64(len(data))
			}
			assert.LessOrEqual(t, total, tc.total)
			assert.Equal(t, tc.expected, content.String())
			n, err := file.Write([]byte("after close\n"))
			assert.Zero(t, n)
			assert.ErrorIs(t, err, os.ErrClosed)
		})
	}
}

func TestLogRotationRetention(t *testing.T) {
	previousLogger := log.Logger
	previousHandler := zerolog.ErrorHandler
	t.Cleanup(func() {
		log.Logger = previousLogger
		zerolog.ErrorHandler = previousHandler
	})
	for _, stage := range []string{"startup", "write", "close"} {
		t.Run(stage, func(t *testing.T) {
			root := t.TempDir()
			cfg := DefaultConfiguration()
			cfg.LogsCfg.Output = "file"
			cfg.LogsCfg.Directory = filepath.Join(root, "logs")
			cfg.DeliveryCfg.DumpDirectory = filepath.Join(root, "queue")
			cfg.LogsCfg.Rotation = RotationConf{MaxFileSizeBytes: 32, MaxBackups: 5, MaxAgeDays: 1, MaxTotalSizeBytes: 64}
			err := os.Mkdir(cfg.LogsCfg.Directory, 0750)
			assert.NoError(t, err)
			if err != nil {
				return
			}
			var file *RotatingFile
			if stage != "startup" {
				file, err = PrepareLogger(cfg)
				assert.NoError(t, err)
				if err != nil {
					return
				}
			}
			old := filepath.Join(cfg.LogsCfg.Directory, "application-"+time.Now().UTC().Add(-48*time.Hour).Format("20060102T150405.000000000Z")+".log")
			recent := filepath.Join(cfg.LogsCfg.Directory, "application-"+time.Now().UTC().Add(-time.Hour).Format("20060102T150405.000000000Z")+".log")
			for _, name := range []string{old, recent, filepath.Join(cfg.LogsCfg.Directory, "application-not-an-archive.log"), filepath.Join(cfg.LogsCfg.Directory, PACKETS_FILENAME), filepath.Join(cfg.LogsCfg.Directory, DUMP_FILENAME)} {
				err = os.WriteFile(name, []byte("keep\n"), 0640)
				assert.NoError(t, err)
			}
			if stage == "startup" {
				file, err = PrepareLogger(cfg)
				assert.NoError(t, err)
				if err != nil {
					return
				}
			}
			if stage == "write" {
				_, err = file.Write([]byte("new\n"))
				assert.NoError(t, err)
			}
			if stage == "close" {
				err = file.Close()
				assert.NoError(t, err)
			}
			_, err = os.Stat(old)
			assert.True(t, errors.Is(err, os.ErrNotExist))
			for _, name := range []string{recent, filepath.Join(cfg.LogsCfg.Directory, "application-not-an-archive.log"), filepath.Join(cfg.LogsCfg.Directory, PACKETS_FILENAME), filepath.Join(cfg.LogsCfg.Directory, DUMP_FILENAME)} {
				data, err := os.ReadFile(name)
				assert.NoError(t, err)
				assert.Equal(t, "keep\n", string(data))
			}
			err = file.Close()
			assert.NoError(t, err)
		})
	}
}

func TestLogRotationConcurrentWrites(t *testing.T) {
	previousLogger := log.Logger
	previousHandler := zerolog.ErrorHandler
	t.Cleanup(func() {
		log.Logger = previousLogger
		zerolog.ErrorHandler = previousHandler
	})
	root := t.TempDir()
	cfg := DefaultConfiguration()
	cfg.LogsCfg.Output = "file"
	cfg.LogsCfg.Directory = filepath.Join(root, "logs")
	cfg.DeliveryCfg.DumpDirectory = filepath.Join(root, "queue")
	cfg.LogsCfg.Rotation = RotationConf{MaxFileSizeBytes: 16, MaxBackups: 128, MaxTotalSizeBytes: 4096}
	file, err := PrepareLogger(cfg)
	assert.NoError(t, err)
	if err != nil {
		return
	}
	var wait sync.WaitGroup
	var expected []string
	for i := 0; i < 128; i++ {
		record := fmt.Sprintf("%03d", i)
		expected = append(expected, record)
		wait.Add(1)
		go func(record string) {
			defer wait.Done()
			n, err := file.Write([]byte(record + "\n"))
			assert.NoError(t, err)
			assert.Equal(t, 4, n)
		}(record)
	}
	wait.Wait()
	err = file.Close()
	assert.NoError(t, err)
	entries, err := os.ReadDir(cfg.LogsCfg.Directory)
	assert.NoError(t, err)
	assert.Len(t, entries, 32)
	var actual []string
	for _, entry := range entries {
		data, err := os.ReadFile(filepath.Join(cfg.LogsCfg.Directory, entry.Name()))
		assert.NoError(t, err)
		assert.Len(t, data, 16)
		actual = append(actual, strings.Split(strings.TrimSuffix(string(data), "\n"), "\n")...)
	}
	assert.ElementsMatch(t, expected, actual)
}

func TestLogRotationWriteErrors(t *testing.T) {
	previousLogger := log.Logger
	previousHandler := zerolog.ErrorHandler
	t.Cleanup(func() {
		log.Logger = previousLogger
		zerolog.ErrorHandler = previousHandler
	})
	for _, cause := range []string{"oversized record", "read-only descriptor", "replaced active file"} {
		t.Run(cause, func(t *testing.T) {
			root := t.TempDir()
			cfg := DefaultConfiguration()
			cfg.LogsCfg.Output = "file"
			cfg.LogsCfg.Directory = filepath.Join(root, "logs")
			cfg.DeliveryCfg.DumpDirectory = filepath.Join(root, "queue")
			cfg.LogsCfg.Rotation = RotationConf{MaxFileSizeBytes: 5, MaxBackups: 5, MaxTotalSizeBytes: 30}
			file, err := PrepareLogger(cfg)
			assert.NoError(t, err)
			if err != nil {
				return
			}
			name := filepath.Join(cfg.LogsCfg.Directory, LOG_FILENAME)
			record := []byte("longer than the file limit\n")
			if cause == "read-only descriptor" {
				err = file.file.Close()
				assert.NoError(t, err)
				file.file, err = os.Open(name)
				assert.NoError(t, err)
				record = []byte("new\n")
			}
			if cause == "replaced active file" {
				_, err = file.Write([]byte("full\n"))
				assert.NoError(t, err)
				err = os.Rename(name, name+".saved")
				assert.NoError(t, err)
				err = os.WriteFile(name, []byte("other"), 0640)
				assert.NoError(t, err)
				record = []byte("new\n")
			}
			n, writeErr := file.Write(record)
			assert.Zero(t, n)
			assert.Error(t, writeErr)
			err = file.Close()
			assert.Error(t, err)
			assert.ErrorIs(t, err, writeErr)
			data, err := os.ReadFile(name)
			assert.NoError(t, err)
			if cause == "replaced active file" {
				assert.Equal(t, "other", string(data))
				data, err = os.ReadFile(name + ".saved")
				assert.NoError(t, err)
				assert.Equal(t, "full\n", string(data))
			} else {
				assert.Empty(t, data)
			}
		})
	}
}

func TestRotationConfiguration(t *testing.T) {
	cases := []rotationValidationTestCase{
		{name: "minimum", settings: RotationConf{MaxFileSizeBytes: 1, MaxTotalSizeBytes: 1}, valid: true},
		{name: "zero file size", settings: RotationConf{MaxTotalSizeBytes: 100}},
		{name: "negative file size", settings: RotationConf{MaxFileSizeBytes: -1, MaxTotalSizeBytes: 100}},
		{name: "negative archive count", settings: RotationConf{MaxFileSizeBytes: 10, MaxTotalSizeBytes: 100, MaxBackups: -1}},
		{name: "negative age", settings: RotationConf{MaxFileSizeBytes: 10, MaxTotalSizeBytes: 100, MaxAgeDays: -1}},
		{name: "age overflow", settings: RotationConf{MaxFileSizeBytes: 10, MaxTotalSizeBytes: 100, MaxAgeDays: 106752}},
		{name: "zero total", settings: RotationConf{MaxFileSizeBytes: 10}},
		{name: "total below file size", settings: RotationConf{MaxFileSizeBytes: 10, MaxTotalSizeBytes: 9}},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			err := tc.settings.Validate()
			cfg := DefaultConfiguration()
			cfg.LogsCfg.Output = "file"
			cfg.LogsCfg.Directory = filepath.Join(t.TempDir(), "logs")
			cfg.LogsCfg.Rotation = tc.settings
			configErr := cfg.Validate()
			if tc.valid {
				assert.NoError(t, err)
				assert.NoError(t, configErr)
			} else {
				assert.Error(t, err)
				assert.Error(t, configErr)
			}
		})
	}
}

func TestLogRotationInterruptedArchive(t *testing.T) {
	previousLogger := log.Logger
	previousHandler := zerolog.ErrorHandler
	t.Cleanup(func() {
		log.Logger = previousLogger
		zerolog.ErrorHandler = previousHandler
	})
	for _, maxFileSize := range []int64{5, 20} {
		root := t.TempDir()
		cfg := DefaultConfiguration()
		cfg.LogsCfg.Output = "file"
		cfg.LogsCfg.Directory = filepath.Join(root, "logs")
		cfg.DeliveryCfg.DumpDirectory = filepath.Join(root, "queue")
		cfg.LogsCfg.Rotation = RotationConf{MaxFileSizeBytes: maxFileSize, MaxBackups: 5, MaxTotalSizeBytes: 30}
		err := os.Mkdir(cfg.LogsCfg.Directory, 0750)
		assert.NoError(t, err)
		if err != nil {
			return
		}
		active := filepath.Join(cfg.LogsCfg.Directory, LOG_FILENAME)
		existing := "existing log data\n"
		err = os.WriteFile(active, []byte(existing), 0640)
		assert.NoError(t, err)
		unfinished := filepath.Join(cfg.LogsCfg.Directory, "application-20200102T000000.000000000Z.log")
		err = os.Link(active, unfinished)
		assert.NoError(t, err)
		previous := filepath.Join(cfg.LogsCfg.Directory, "application-20200101T000000.000000000Z.log")
		err = os.WriteFile(previous, []byte("old\n"), 0640)
		assert.NoError(t, err)
		file, err := PrepareLogger(cfg)
		assert.NoError(t, err)
		if err != nil {
			return
		}
		_, err = os.Stat(unfinished)
		assert.True(t, errors.Is(err, os.ErrNotExist))
		_, err = file.Write([]byte("new\n"))
		assert.NoError(t, err)
		err = file.Close()
		assert.NoError(t, err)
		entries, err := os.ReadDir(cfg.LogsCfg.Directory)
		assert.NoError(t, err)
		var content strings.Builder
		for _, entry := range entries {
			data, err := os.ReadFile(filepath.Join(cfg.LogsCfg.Directory, entry.Name()))
			assert.NoError(t, err)
			content.Write(data)
		}
		assert.Equal(t, "old\n"+existing+"new\n", content.String())
	}
}

func TestLogRotationRejectsPacketArchiveAlias(t *testing.T) {
	previousLogger := log.Logger
	previousHandler := zerolog.ErrorHandler
	t.Cleanup(func() {
		log.Logger = previousLogger
		zerolog.ErrorHandler = previousHandler
	})
	root := t.TempDir()
	cfg := DefaultConfiguration()
	cfg.LogsCfg.Output = "file"
	cfg.LogsCfg.Directory = filepath.Join(root, "logs")
	cfg.LogsCfg.Rotation = RotationConf{MaxFileSizeBytes: 5, MaxBackups: 0, MaxAgeDays: 1, MaxTotalSizeBytes: 5}
	cfg.DeliveryCfg.DumpDirectory = filepath.Join(root, "queue")
	cfg.DestinationsCfg.File.Enabled = true
	cfg.DestinationsCfg.File.Directory = filepath.Join(root, "packets")
	file, err := PrepareLogger(cfg)
	assert.NoError(t, err)
	if err != nil {
		return
	}
	_, err = file.Write([]byte("full\n"))
	assert.NoError(t, err)
	err = os.Mkdir(cfg.DestinationsCfg.File.Directory, 0750)
	assert.NoError(t, err)
	packet := filepath.Join(cfg.DestinationsCfg.File.Directory, PACKETS_FILENAME)
	err = os.WriteFile(packet, []byte("packet data\n"), 0640)
	assert.NoError(t, err)
	archive := filepath.Join(cfg.LogsCfg.Directory, "application-20200101T000000.000000000Z.log")
	err = os.Link(packet, archive)
	assert.NoError(t, err)
	n, err := file.Write([]byte("next\n"))
	assert.Zero(t, n)
	var conflict *OutputConflictError
	assert.ErrorAs(t, err, &conflict)
	err = file.Close()
	assert.ErrorAs(t, err, &conflict)
	for _, name := range []string{packet, archive} {
		data, err := os.ReadFile(name)
		assert.NoError(t, err)
		assert.Equal(t, "packet data\n", string(data))
	}
	data, err := os.ReadFile(filepath.Join(cfg.LogsCfg.Directory, LOG_FILENAME))
	assert.NoError(t, err)
	assert.Equal(t, "full\n", string(data))
}
