package configuration_test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/LdDl/go-egts/gateway/configuration"
	"github.com/stretchr/testify/assert"
)

type outputDirectoryTestCase struct {
	name    string
	logs    string
	packets string
	queue   string
	valid   bool
}

type outputLinkTestCase struct {
	name    string
	source  string
	target  string
	symlink bool
}

func TestOutputDirectories(t *testing.T) {
	root := t.TempDir()
	cases := []outputDirectoryTestCase{
		{name: "separate", logs: "logs", packets: "packets", queue: "queue", valid: true},
		{name: "same", logs: "logs", packets: "logs", queue: "queue"},
		{name: "same after cleaning", logs: "logs", packets: "packets/../logs", queue: "queue"},
		{name: "packets inside logs", logs: "logs", packets: "logs/packets", queue: "queue"},
		{name: "logs inside packets", logs: "packets/logs", packets: "packets", queue: "queue"},
		{name: "queue equals logs", logs: "logs", packets: "packets", queue: "logs"},
		{name: "queue equals packets", logs: "logs", packets: "packets", queue: "packets"},
		{name: "queue inside logs", logs: "logs", packets: "packets", queue: "logs/queue"},
		{name: "queue contains packets", logs: "logs", packets: "queue/packets", queue: "queue"},
		{name: "similar prefixes", logs: "out", packets: "output", queue: "out_queue", valid: true},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			cfg := configuration.DefaultConfiguration()
			cfg.LogsCfg.Output = "file"
			cfg.LogsCfg.Directory = filepath.Join(root, tc.logs)
			cfg.DestinationsCfg.Stdout = false
			cfg.DestinationsCfg.File.Enabled = true
			cfg.DestinationsCfg.File.Directory = filepath.Join(root, tc.packets)
			cfg.DeliveryCfg.DumpDirectory = filepath.Join(root, tc.queue)
			err := cfg.ValidateOutputs(nil, nil)
			if tc.valid {
				assert.NoError(t, err)
			} else {
				assert.Error(t, err)
				var conflict *configuration.OutputConflictError
				assert.ErrorAs(t, err, &conflict)
			}
		})
	}
	entries, err := os.ReadDir(root)
	assert.NoError(t, err)
	assert.Empty(t, entries)
}

func TestOutputDirectoryAliases(t *testing.T) {
	root := t.TempDir()
	real := filepath.Join(root, "real")
	alias := filepath.Join(root, "alias")
	err := os.Mkdir(real, 0750)
	assert.NoError(t, err)
	if err != nil {
		return
	}
	err = os.Symlink(real, alias)
	assert.NoError(t, err)
	if err != nil {
		return
	}
	for _, child := range []string{"", "missing", "missing/child"} {
		cfg := configuration.DefaultConfiguration()
		cfg.LogsCfg.Output = "file"
		cfg.LogsCfg.Directory = filepath.Join(real, child)
		cfg.DestinationsCfg.File.Enabled = true
		cfg.DestinationsCfg.File.Directory = filepath.Join(alias, child)
		cfg.DeliveryCfg.DumpDirectory = filepath.Join(root, "queue")
		err = cfg.ValidateOutputs(nil, nil)
		assert.Error(t, err)
		var conflict *configuration.OutputConflictError
		assert.ErrorAs(t, err, &conflict)
	}
	cfg := configuration.DefaultConfiguration()
	cfg.LogsCfg.Output = "file"
	cfg.LogsCfg.Directory = real
	cfg.DestinationsCfg.File.Enabled = true
	cwd, err := os.Getwd()
	assert.NoError(t, err)
	if err != nil {
		return
	}
	cfg.DestinationsCfg.File.Directory, err = filepath.Rel(cwd, alias)
	assert.NoError(t, err)
	cfg.DeliveryCfg.DumpDirectory = filepath.Join(root, "queue")
	err = cfg.ValidateOutputs(nil, nil)
	assert.Error(t, err)
	var conflict *configuration.OutputConflictError
	assert.ErrorAs(t, err, &conflict)
}

func TestOutputFileAliases(t *testing.T) {
	cases := []outputLinkTestCase{
		{name: "packets linked to logs", source: "logs/application.log", target: "packets/packets.ndjson"},
		{name: "logs linked to packets", source: "packets/packets.ndjson", target: "logs/application.log"},
		{name: "log archive linked to packets", source: "packets/packets.ndjson", target: "logs/application-old.log"},
		{name: "packet archive linked to logs", source: "logs/application.log", target: "packets/packets-old.ndjson"},
		{name: "dump linked to packets", source: "packets/packets.ndjson", target: "queue/queue.dump"},
		{name: "logs linked to dump", source: "queue/queue.dump", target: "logs/application.log"},
		{name: "symbolic packet link", source: "logs/application.log", target: "packets/packets.ndjson", symlink: true},
		{name: "symbolic log link", source: "packets/packets.ndjson", target: "logs/application.log", symlink: true},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			root := t.TempDir()
			for _, directory := range []string{"logs", "packets", "queue"} {
				err := os.Mkdir(filepath.Join(root, directory), 0750)
				assert.NoError(t, err)
				if err != nil {
					return
				}
			}
			source := filepath.Join(root, tc.source)
			target := filepath.Join(root, tc.target)
			data := []byte("existing data\n")
			err := os.WriteFile(source, data, 0640)
			assert.NoError(t, err)
			if err != nil {
				return
			}
			if tc.symlink {
				err = os.Symlink(source, target)
			} else {
				err = os.Link(source, target)
			}
			assert.NoError(t, err)
			if err != nil {
				return
			}
			cfg := configuration.DefaultConfiguration()
			cfg.LogsCfg.Output = "file"
			cfg.LogsCfg.Directory = filepath.Join(root, "logs")
			cfg.DestinationsCfg.File.Enabled = true
			cfg.DestinationsCfg.File.Directory = filepath.Join(root, "packets")
			cfg.DeliveryCfg.DumpDirectory = filepath.Join(root, "queue")
			err = cfg.ValidateOutputs(nil, nil)
			var conflict *configuration.OutputConflictError
			assert.ErrorAs(t, err, &conflict)
			logFile, err := configuration.PrepareLogger(cfg)
			assert.Error(t, err)
			assert.Nil(t, logFile)
			result, err := os.ReadFile(source)
			assert.NoError(t, err)
			assert.Equal(t, data, result)
		})
	}
}

func TestOutputStreams(t *testing.T) {
	root := t.TempDir()
	stdout, err := os.Create(filepath.Join(root, "stdout"))
	assert.NoError(t, err)
	if err != nil {
		return
	}
	defer stdout.Close()
	stderr, err := os.Create(filepath.Join(root, "stderr"))
	assert.NoError(t, err)
	if err != nil {
		return
	}
	defer stderr.Close()
	cfg := configuration.DefaultConfiguration()
	cfg.DeliveryCfg.DumpDirectory = filepath.Join(root, "queue")
	err = cfg.ValidateOutputs(stdout, stderr)
	assert.NoError(t, err)
	err = cfg.ValidateOutputs(stdout, stdout)
	var conflict *configuration.OutputConflictError
	isConflict := assert.ErrorAs(t, err, &conflict)
	if isConflict {
		assert.True(t, conflict.StderrUnsafe)
	}
	cfg.LogsCfg.Output = "stdout"
	err = cfg.ValidateOutputs(stdout, stderr)
	assert.Error(t, err)
	cfg.DestinationsCfg.Stdout = false
	cfg.DestinationsCfg.File.Enabled = true
	cfg.DestinationsCfg.File.Directory = filepath.Join(root, "packets")
	err = cfg.ValidateOutputs(stdout, stderr)
	assert.NoError(t, err)

	err = os.Mkdir(cfg.DestinationsCfg.File.Directory, 0750)
	assert.NoError(t, err)
	if err != nil {
		return
	}
	err = os.Link(stderr.Name(), filepath.Join(cfg.DestinationsCfg.File.Directory, configuration.PACKETS_FILENAME))
	assert.NoError(t, err)
	if err != nil {
		return
	}
	err = cfg.ValidateOutputs(stdout, stderr)
	isConflict = assert.ErrorAs(t, err, &conflict)
	if isConflict {
		assert.True(t, conflict.StderrUnsafe)
	}
}

func TestInvalidOutputDirectory(t *testing.T) {
	root := t.TempDir()
	file := filepath.Join(root, "file")
	err := os.WriteFile(file, []byte("existing data"), 0640)
	assert.NoError(t, err)
	if err != nil {
		return
	}
	alias := filepath.Join(root, "broken")
	err = os.Symlink(filepath.Join(root, "missing"), alias)
	assert.NoError(t, err)
	if err != nil {
		return
	}
	for _, directory := range []string{file, filepath.Join(file, "child"), alias, filepath.Join(alias, "child")} {
		cfg := configuration.DefaultConfiguration()
		cfg.LogsCfg.Output = "file"
		cfg.LogsCfg.Directory = directory
		cfg.DeliveryCfg.DumpDirectory = filepath.Join(root, "queue")
		err = cfg.ValidateOutputs(nil, nil)
		assert.Error(t, err)
	}
	cfg := configuration.DefaultConfiguration()
	cfg.LogsCfg.Directory = file
	cfg.DestinationsCfg.File.Directory = alias
	cfg.DeliveryCfg.DumpDirectory = filepath.Join(root, "queue")
	err = cfg.ValidateOutputs(nil, nil)
	assert.NoError(t, err)
}
