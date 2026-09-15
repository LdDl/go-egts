package configuration_test

import (
	"errors"
	"os"
	"path/filepath"
	"testing"

	"github.com/LdDl/go-egts/gateway/configuration"
	"github.com/stretchr/testify/assert"
)

type fileConfigurationTestCase struct {
	name      string
	data      string
	errorText string
}

type envConfigurationTestCase struct {
	name      string
	key       string
	value     string
	errorText string
}

func TestPrepareFileConfiguration(t *testing.T) {
	fname := filepath.Join(t.TempDir(), "gateway.toml")
	data := `[server_cfg]
host = "127.0.0.1"
port = 9000
[auth_cfg]
enabled = true
password = "test password # with spaces"
[delivery_cfg]
ack_mode = "delivered"
queue_capacity = 32
dump_after_seconds = 15
dump_directory = "/tmp/gateway/queue"
[logs_cfg]
output = "stderr"
directory = "/tmp/gateway/logs"
[destinations_cfg]
stdout = false
[destinations_cfg.file]
enabled = true
directory = "/tmp/gateway/packets"
`
	err := os.WriteFile(fname, []byte(data), 0600)
	assert.NoError(t, err)
	if err != nil {
		return
	}
	cfg, err := configuration.PrepareFileConfiguration(fname)
	assert.NoError(t, err)
	want := &configuration.Configuration{
		ServerCfg: configuration.ServerConf{Host: "127.0.0.1", Port: 9000},
		AuthCfg: configuration.AuthConf{
			Enabled:  true,
			Password: "test password # with spaces",
		},
		DeliveryCfg: configuration.DeliveryConf{
			AckMode:          "delivered",
			QueueCapacity:    32,
			DumpAfterSeconds: 15,
			DumpDirectory:    "/tmp/gateway/queue",
		},
		LogsCfg: configuration.LogsConf{Output: "stderr", Directory: "/tmp/gateway/logs"},
		DestinationsCfg: configuration.DestinationsConf{
			Stdout: false,
			File:   configuration.FileDestinationConf{Enabled: true, Directory: "/tmp/gateway/packets"},
		},
	}
	assert.Equal(t, want, cfg)
}

func TestPrepareFileConfigurationDefaults(t *testing.T) {
	for _, data := range []string{"", "[server_cfg]\nport = 8081\n", "[auth_cfg]\nenabled = false\npassword = \"\"\n"} {
		fname := filepath.Join(t.TempDir(), "gateway.toml")
		err := os.WriteFile(fname, []byte(data), 0600)
		assert.NoError(t, err)
		if err != nil {
			return
		}
		cfg, err := configuration.PrepareFileConfiguration(fname)
		assert.NoError(t, err)
		assert.Equal(t, configuration.DefaultConfiguration(), cfg)
	}
}

func TestPrepareFileConfigurationErrors(t *testing.T) {
	cases := []fileConfigurationTestCase{
		{name: "invalid TOML", data: "[server_cfg", errorText: "Can't decode TOML"},
		{name: "wrong type", data: "[server_cfg]\nport = \"8081\"", errorText: "Can't decode TOML"},
		{name: "duplicate field", data: "[server_cfg]\nport = 8081\nport = 9000", errorText: "Can't decode TOML"},
		{name: "unknown section", data: "[server]\nport = 8081", errorText: "Unknown configuration key"},
		{name: "unknown field", data: "[server_cfg]\nprot = 8081", errorText: "server_cfg.prot"},
		{name: "empty host", data: "[server_cfg]\nhost = \" \"", errorText: "server_cfg.host"},
		{name: "zero port", data: "[server_cfg]\nport = 0", errorText: "server_cfg.port"},
		{name: "negative port", data: "[server_cfg]\nport = -1", errorText: "server_cfg.port"},
		{name: "large port", data: "[server_cfg]\nport = 65536", errorText: "server_cfg.port"},
		{name: "missing password", data: "[auth_cfg]\nenabled = true", errorText: "auth_cfg.password"},
		{name: "invalid mode", data: "[delivery_cfg]\nack_mode = \"received\"", errorText: "delivery_cfg.ack_mode"},
		{name: "empty mode", data: "[delivery_cfg]\nack_mode = \"\"", errorText: "delivery_cfg.ack_mode"},
		{name: "zero capacity", data: "[delivery_cfg]\nqueue_capacity = 0", errorText: "delivery_cfg.queue_capacity"},
		{name: "negative capacity", data: "[delivery_cfg]\nqueue_capacity = -1", errorText: "delivery_cfg.queue_capacity"},
		{name: "zero dump delay", data: "[delivery_cfg]\ndump_after_seconds = 0", errorText: "delivery_cfg.dump_after_seconds"},
		{name: "negative dump delay", data: "[delivery_cfg]\ndump_after_seconds = -1", errorText: "delivery_cfg.dump_after_seconds"},
		{name: "empty dump path", data: "[delivery_cfg]\ndump_directory = \" \"", errorText: "delivery_cfg.dump_directory"},
		{name: "removed log setting", data: "[logs_cfg]\nlevel = \"info\"", errorText: "Unknown configuration key: logs_cfg.level"},
		{name: "unsupported log output", data: "[logs_cfg]\noutput = \"unknown\"", errorText: "logs_cfg.output"},
		{name: "empty log directory", data: "[logs_cfg]\noutput = \"file\"\ndirectory = \"\"", errorText: "logs_cfg.directory"},
		{name: "empty packet directory", data: "[destinations_cfg.file]\nenabled = true\ndirectory = \"\"", errorText: "destinations_cfg.file.directory"},
		{name: "no packet destination", data: "[destinations_cfg]\nstdout = false", errorText: "At least one packet destination"},
		{name: "shared stdout", data: "[logs_cfg]\noutput = \"stdout\"", errorText: "Output destinations overlap"},
		{name: "unquoted password", data: "[auth_cfg]\npassword = test-secret-do-not-log", errorText: "Can't decode TOML"},
		{name: "password with wrong type", data: "[auth_cfg]\npassword = [\"test-secret-do-not-log\"]", errorText: "Can't decode TOML"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			fname := filepath.Join(t.TempDir(), "gateway.toml")
			err := os.WriteFile(fname, []byte(tc.data), 0600)
			assert.NoError(t, err)
			if err != nil {
				return
			}
			cfg, err := configuration.PrepareFileConfiguration(fname)
			assert.Error(t, err)
			assert.Nil(t, cfg)
			if err == nil {
				return
			}
			assert.Contains(t, err.Error(), tc.errorText)
			assert.NotContains(t, err.Error(), "test-secret-do-not-log")
		})
	}
	fname := filepath.Join(t.TempDir(), "missing.toml")
	cfg, err := configuration.PrepareFileConfiguration(fname)
	assert.Error(t, err)
	assert.Nil(t, cfg)
	assert.True(t, errors.Is(err, os.ErrNotExist))
}

func TestPrepareEnvConfiguration(t *testing.T) {
	values := map[string]string{
		"EGTS_SERVER_HOST":            "::1",
		"EGTS_SERVER_PORT":            "65535",
		"EGTS_AUTH_ENABLED":           "true",
		"EGTS_AUTH_PASSWORD":          "test password # with spaces",
		"EGTS_ACK_MODE":               "delivered",
		"EGTS_QUEUE_CAPACITY":         "32",
		"EGTS_DUMP_AFTER_SECONDS":     "15",
		"EGTS_DUMP_DIRECTORY":         "/tmp/gateway/queue",
		"EGTS_LOG_OUTPUT":             "stderr",
		"EGTS_LOG_DIRECTORY":          "/tmp/gateway/logs",
		"EGTS_PACKETS_STDOUT":         "false",
		"EGTS_PACKETS_FILE_ENABLED":   "true",
		"EGTS_PACKETS_FILE_DIRECTORY": "/tmp/gateway/packets",
	}
	for key, value := range values {
		t.Setenv(key, value)
	}
	cfg, err := configuration.PrepareEnvConfiguration()
	assert.NoError(t, err)
	want := &configuration.Configuration{
		ServerCfg: configuration.ServerConf{Host: "::1", Port: 65535},
		AuthCfg: configuration.AuthConf{
			Enabled:  true,
			Password: "test password # with spaces",
		},
		DeliveryCfg: configuration.DeliveryConf{
			AckMode:          "delivered",
			QueueCapacity:    32,
			DumpAfterSeconds: 15,
			DumpDirectory:    "/tmp/gateway/queue",
		},
		LogsCfg: configuration.LogsConf{Output: "stderr", Directory: "/tmp/gateway/logs"},
		DestinationsCfg: configuration.DestinationsConf{
			Stdout: false,
			File:   configuration.FileDestinationConf{Enabled: true, Directory: "/tmp/gateway/packets"},
		},
	}
	assert.Equal(t, want, cfg)

	t.Setenv("EGTS_AUTH_ENABLED", "false")
	t.Setenv("EGTS_AUTH_PASSWORD", "")
	cfg, err = configuration.PrepareEnvConfiguration()
	assert.NoError(t, err)
	if err != nil {
		return
	}
	assert.False(t, cfg.AuthCfg.Enabled)
	assert.Empty(t, cfg.AuthCfg.Password)

	t.Setenv("EGTS_AUTH_ENABLED", "true")
	cfg, err = configuration.PrepareEnvConfiguration()
	assert.Error(t, err)
	assert.Nil(t, cfg)
}

func TestPrepareEnvConfigurationErrors(t *testing.T) {
	values := map[string]string{
		"EGTS_SERVER_HOST":            "127.0.0.1",
		"EGTS_SERVER_PORT":            "8081",
		"EGTS_AUTH_ENABLED":           "false",
		"EGTS_AUTH_PASSWORD":          "",
		"EGTS_ACK_MODE":               "queued",
		"EGTS_QUEUE_CAPACITY":         "1024",
		"EGTS_DUMP_AFTER_SECONDS":     "60",
		"EGTS_DUMP_DIRECTORY":         "./data/queue",
		"EGTS_LOG_OUTPUT":             "stderr",
		"EGTS_LOG_DIRECTORY":          "./data/logs",
		"EGTS_PACKETS_STDOUT":         "true",
		"EGTS_PACKETS_FILE_ENABLED":   "false",
		"EGTS_PACKETS_FILE_DIRECTORY": "./data/packets",
	}
	for key, value := range values {
		t.Setenv(key, value)
	}
	cases := []envConfigurationTestCase{
		{name: "empty host", key: "EGTS_SERVER_HOST", value: "", errorText: "server_cfg.host"},
		{name: "empty port", key: "EGTS_SERVER_PORT", value: "", errorText: "EGTS_SERVER_PORT"},
		{name: "invalid port", key: "EGTS_SERVER_PORT", value: "tcp", errorText: "EGTS_SERVER_PORT"},
		{name: "port overflow", key: "EGTS_SERVER_PORT", value: "9999999999999999999999", errorText: "EGTS_SERVER_PORT"},
		{name: "zero port", key: "EGTS_SERVER_PORT", value: "0", errorText: "server_cfg.port"},
		{name: "invalid bool", key: "EGTS_AUTH_ENABLED", value: "maybe", errorText: "EGTS_AUTH_ENABLED"},
		{name: "empty bool", key: "EGTS_AUTH_ENABLED", value: "", errorText: "EGTS_AUTH_ENABLED"},
		{name: "invalid mode", key: "EGTS_ACK_MODE", value: "received", errorText: "delivery_cfg.ack_mode"},
		{name: "invalid capacity", key: "EGTS_QUEUE_CAPACITY", value: "many", errorText: "EGTS_QUEUE_CAPACITY"},
		{name: "negative capacity", key: "EGTS_QUEUE_CAPACITY", value: "-1", errorText: "delivery_cfg.queue_capacity"},
		{name: "invalid delay", key: "EGTS_DUMP_AFTER_SECONDS", value: "1s", errorText: "EGTS_DUMP_AFTER_SECONDS"},
		{name: "zero delay", key: "EGTS_DUMP_AFTER_SECONDS", value: "0", errorText: "delivery_cfg.dump_after_seconds"},
		{name: "empty path", key: "EGTS_DUMP_DIRECTORY", value: "", errorText: "delivery_cfg.dump_directory"},
		{name: "empty output", key: "EGTS_LOG_OUTPUT", value: "", errorText: "logs_cfg.output"},
		{name: "invalid packet stdout", key: "EGTS_PACKETS_STDOUT", value: "maybe", errorText: "EGTS_PACKETS_STDOUT"},
		{name: "invalid packet file flag", key: "EGTS_PACKETS_FILE_ENABLED", value: "maybe", errorText: "EGTS_PACKETS_FILE_ENABLED"},
		{name: "no packet destination", key: "EGTS_PACKETS_STDOUT", value: "false", errorText: "At least one packet destination"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Setenv(tc.key, tc.value)
			cfg, err := configuration.PrepareEnvConfiguration()
			assert.Error(t, err)
			assert.Nil(t, cfg)
			if err == nil {
				return
			}
			assert.Contains(t, err.Error(), tc.errorText)
		})
	}
}

func TestExampleConfigurations(t *testing.T) {
	cfg, err := configuration.PrepareFileConfiguration("../../egts_gateway.toml")
	assert.NoError(t, err)
	assert.Equal(t, configuration.DefaultConfiguration(), cfg)
}
