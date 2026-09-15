package configuration_test

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/LdDl/go-egts/gateway/configuration"
	"github.com/LdDl/go-egts/gateway/logger"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
	"github.com/stretchr/testify/assert"
)

func TestApplicationFileLogger(t *testing.T) {
	previousLogger := log.Logger
	previousTimeFormat := zerolog.TimeFieldFormat
	t.Cleanup(func() {
		log.Logger = previousLogger
		zerolog.TimeFieldFormat = previousTimeFormat
	})
	root := t.TempDir()
	cfg := configuration.DefaultConfiguration()
	cfg.LogsCfg.Output = "file"
	cfg.LogsCfg.Directory = filepath.Join(root, "logs")
	cfg.DestinationsCfg.File.Enabled = true
	cfg.DestinationsCfg.File.Directory = filepath.Join(root, "packets")
	cfg.DeliveryCfg.DumpDirectory = filepath.Join(root, "queue")
	for _, directory := range []string{cfg.LogsCfg.Directory, cfg.DestinationsCfg.File.Directory, cfg.DeliveryCfg.DumpDirectory} {
		err := os.Mkdir(directory, 0750)
		assert.NoError(t, err)
		if err != nil {
			return
		}
	}
	packetFile := filepath.Join(cfg.DestinationsCfg.File.Directory, configuration.PACKETS_FILENAME)
	dumpFile := filepath.Join(cfg.DeliveryCfg.DumpDirectory, configuration.DUMP_FILENAME)
	data := []byte("existing packet data\n")
	for _, name := range []string{packetFile, dumpFile} {
		err := os.WriteFile(name, data, 0640)
		assert.NoError(t, err)
		if err != nil {
			return
		}
	}
	for i := 0; i < 2; i++ {
		file, err := configuration.PrepareLogger(cfg)
		assert.NoError(t, err)
		if err != nil {
			return
		}
		assert.NotNil(t, file)
		if file == nil {
			return
		}
		log.Log().Str("scope", logger.SCOPE_STARTUP).Str("event", logger.EVENT_STARTUP).Msg("Application event")
		err = file.Close()
		assert.NoError(t, err)
	}
	encoded, err := os.ReadFile(filepath.Join(cfg.LogsCfg.Directory, configuration.LOG_FILENAME))
	assert.NoError(t, err)
	lines := strings.Split(strings.TrimSuffix(string(encoded), "\n"), "\n")
	assert.Len(t, lines, 2)
	for _, line := range lines {
		var entry map[string]interface{}
		err = json.Unmarshal([]byte(line), &entry)
		assert.NoError(t, err)
		assert.Equal(t, "startup", entry["scope"])
		assert.Equal(t, "startup", entry["event"])
		assert.Equal(t, "Application event", entry["message"])
		assert.NotContains(t, entry, "level")
	}
	for _, name := range []string{packetFile, dumpFile} {
		encoded, err := os.ReadFile(name)
		assert.NoError(t, err)
		assert.Equal(t, data, encoded)
	}
}
