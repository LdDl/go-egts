package configuration

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestPacketRotationConfiguration(t *testing.T) {
	cases := []rotationValidationTestCase{
		{name: "valid", settings: RotationConf{MaxFileSizeBytes: 1, MaxTotalSizeBytes: 1}, valid: true},
		{name: "zero file limit", settings: RotationConf{MaxFileSizeBytes: 0, MaxTotalSizeBytes: 1}},
		{name: "negative file limit", settings: RotationConf{MaxFileSizeBytes: -1, MaxTotalSizeBytes: 1}},
		{name: "negative backups", settings: RotationConf{MaxFileSizeBytes: 1, MaxBackups: -1, MaxTotalSizeBytes: 1}},
		{name: "negative age", settings: RotationConf{MaxFileSizeBytes: 1, MaxAgeDays: -1, MaxTotalSizeBytes: 1}},
		{name: "age overflow", settings: RotationConf{MaxFileSizeBytes: 1, MaxAgeDays: 106752, MaxTotalSizeBytes: 1}},
		{name: "total below file limit", settings: RotationConf{MaxFileSizeBytes: 2, MaxTotalSizeBytes: 1}},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			root := t.TempDir()
			cfg := DefaultConfiguration()
			cfg.DeliveryCfg.DumpDirectory = filepath.Join(root, "queue")
			cfg.DestinationsCfg.File.Enabled = true
			cfg.DestinationsCfg.File.Directory = filepath.Join(root, "packets")
			cfg.DestinationsCfg.File.Rotation = tc.settings
			err := cfg.Validate()
			if tc.valid {
				assert.NoError(t, err)
			} else {
				assert.ErrorContains(t, err, "destinations_cfg.file.rotation")
				file, err := PreparePacketFile(cfg)
				assert.Error(t, err)
				assert.Nil(t, file)
				_, err = os.Stat(cfg.DestinationsCfg.File.Directory)
				assert.ErrorIs(t, err, os.ErrNotExist)
			}
			cfg.DestinationsCfg.File.Enabled = false
			err = cfg.Validate()
			assert.NoError(t, err)
		})
	}
}

func TestPacketFileRetention(t *testing.T) {
	root := t.TempDir()
	cfg := DefaultConfiguration()
	cfg.DeliveryCfg.DumpDirectory = filepath.Join(root, "queue")
	cfg.LogsCfg.Output = "file"
	cfg.LogsCfg.Directory = filepath.Join(root, "logs")
	cfg.DestinationsCfg.File.Enabled = true
	cfg.DestinationsCfg.File.Directory = filepath.Join(root, "packets")
	cfg.DestinationsCfg.File.Rotation = RotationConf{MaxFileSizeBytes: 10, MaxBackups: 5, MaxAgeDays: 1, MaxTotalSizeBytes: 60}
	err := os.MkdirAll(cfg.LogsCfg.Directory, 0750)
	assert.NoError(t, err)
	err = os.MkdirAll(cfg.DestinationsCfg.File.Directory, 0750)
	assert.NoError(t, err)
	old := "packets-" + time.Now().UTC().Add(-48*time.Hour).Format(archiveTimeFormat) + ".ndjson"
	recent := "packets-" + time.Now().UTC().Add(-time.Hour).Format(archiveTimeFormat) + ".ndjson"
	logArchive := filepath.Join(cfg.LogsCfg.Directory, "application-"+time.Now().UTC().Add(-48*time.Hour).Format(archiveTimeFormat)+".log")
	foreign := filepath.Join(cfg.DestinationsCfg.File.Directory, "packets-invalid.ndjson")
	for _, name := range []string{filepath.Join(cfg.DestinationsCfg.File.Directory, old), filepath.Join(cfg.DestinationsCfg.File.Directory, recent), logArchive, foreign} {
		err = os.WriteFile(name, []byte("keep\n"), 0640)
		assert.NoError(t, err)
	}
	file, err := PreparePacketFile(cfg)
	assert.NoError(t, err)
	if err != nil {
		return
	}
	_, err = os.Stat(filepath.Join(cfg.DestinationsCfg.File.Directory, old))
	assert.ErrorIs(t, err, os.ErrNotExist)
	for _, name := range []string{filepath.Join(cfg.DestinationsCfg.File.Directory, recent), logArchive, foreign} {
		data, err := os.ReadFile(name)
		assert.NoError(t, err)
		assert.Equal(t, "keep\n", string(data))
	}
	err = file.Close()
	assert.NoError(t, err)
}

func TestPacketFileInterruptedRotation(t *testing.T) {
	root := t.TempDir()
	cfg := DefaultConfiguration()
	cfg.DeliveryCfg.DumpDirectory = filepath.Join(root, "queue")
	cfg.DestinationsCfg.File.Enabled = true
	cfg.DestinationsCfg.File.Directory = filepath.Join(root, "packets")
	cfg.DestinationsCfg.File.Rotation = RotationConf{MaxFileSizeBytes: 5, MaxBackups: 2, MaxTotalSizeBytes: 20}
	err := os.MkdirAll(cfg.DestinationsCfg.File.Directory, 0750)
	assert.NoError(t, err)
	active := filepath.Join(cfg.DestinationsCfg.File.Directory, PACKETS_FILENAME)
	archive := filepath.Join(cfg.DestinationsCfg.File.Directory, "packets-"+time.Now().UTC().Format(archiveTimeFormat)+".ndjson")
	err = os.WriteFile(active, []byte("aaaa\nbbbb\n"), 0640)
	assert.NoError(t, err)
	err = os.Link(active, archive)
	assert.NoError(t, err)
	file, err := PreparePacketFile(cfg)
	assert.NoError(t, err)
	if err != nil {
		return
	}
	n, err := file.Write([]byte("cccc\n"))
	assert.NoError(t, err)
	assert.Equal(t, 5, n)
	err = file.Close()
	assert.NoError(t, err)
	entries, err := os.ReadDir(cfg.DestinationsCfg.File.Directory)
	assert.NoError(t, err)
	assert.Len(t, entries, 2)
	var data []byte
	for _, entry := range entries {
		content, err := os.ReadFile(filepath.Join(cfg.DestinationsCfg.File.Directory, entry.Name()))
		assert.NoError(t, err)
		data = append(data, content...)
	}
	assert.Equal(t, "aaaa\nbbbb\ncccc\n", string(data))
}
