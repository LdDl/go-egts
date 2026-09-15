package configuration_test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/LdDl/go-egts/gateway/configuration"
	"github.com/stretchr/testify/assert"
)

func TestValkeyConfiguration(t *testing.T) {
	data := `[destinations_cfg]
stdout = false
[[destinations_cfg.valkey]]
id = "monitoring"
enabled = true
password = "first password"
[[destinations_cfg.valkey]]
id = "backup"
enabled = true
host = "localhost"
port = 6380
username = "second"
password = "second password"
database = 2
stream_name = "second.packets"
connect_timeout_seconds = 3
write_timeout_seconds = 7
`
	fname := filepath.Join(t.TempDir(), "valkey.toml")
	err := os.WriteFile(fname, []byte(data), 0600)
	assert.NoError(t, err)
	file, err := configuration.PrepareFileConfiguration(fname)
	assert.NoError(t, err)
	if err != nil {
		return
	}
	want := []configuration.ValkeyDestinationConf{
		{ID: "monitoring", Enabled: true, Host: "127.0.0.1", Port: 6379, Password: "first password", StreamName: "egts.packets", ConnectTimeoutSeconds: 5, WriteTimeoutSeconds: 10},
		{ID: "backup", Enabled: true, Host: "localhost", Port: 6380, UserName: "second", Password: "second password", Database: 2, StreamName: "second.packets", ConnectTimeoutSeconds: 3, WriteTimeoutSeconds: 7},
	}
	assert.Equal(t, want, file.DestinationsCfg.Valkey)
	values := map[string]string{
		"EGTS_PACKETS_STDOUT": "false", "EGTS_VALKEY_IDS": "monitoring,backup",
		"EGTS_VALKEY_ENABLED_MONITORING": "true", "EGTS_VALKEY_PASSWORD_MONITORING": "first password",
		"EGTS_VALKEY_ENABLED_BACKUP": "true", "EGTS_VALKEY_HOST_BACKUP": "localhost", "EGTS_VALKEY_PORT_BACKUP": "6380",
		"EGTS_VALKEY_USERNAME_BACKUP": "second", "EGTS_VALKEY_PASSWORD_BACKUP": "second password",
		"EGTS_VALKEY_DATABASE_BACKUP": "2", "EGTS_VALKEY_STREAM_NAME_BACKUP": "second.packets",
		"EGTS_VALKEY_CONNECT_TIMEOUT_SECONDS_BACKUP": "3", "EGTS_VALKEY_WRITE_TIMEOUT_SECONDS_BACKUP": "7",
	}
	for key, value := range values {
		t.Setenv(key, value)
	}
	env, err := configuration.PrepareEnvConfiguration()
	assert.NoError(t, err)
	assert.Equal(t, file, env)
	t.Setenv("EGTS_VALKEY_PASSWORD_MONITORING", "")
	env, err = configuration.PrepareEnvConfiguration()
	assert.NoError(t, err)
	if err != nil {
		return
	}
	assert.Empty(t, env.DestinationsCfg.Valkey[0].UserName)
	assert.Empty(t, env.DestinationsCfg.Valkey[0].Password)
}

func TestValkeyConfigurationErrors(t *testing.T) {
	cases := []envConfigurationTestCase{
		{name: "duplicate IDs", key: "EGTS_VALKEY_IDS", value: "primary,primary", errorText: "Duplicate"},
		{name: "empty ID", key: "EGTS_VALKEY_IDS", value: "primary,", errorText: "must not be empty"},
		{name: "invalid ID", key: "EGTS_VALKEY_IDS", value: "primary,UPPER", errorText: "lowercase"},
		{name: "empty host", key: "EGTS_VALKEY_HOST_PRIMARY", value: "", errorText: "host"},
		{name: "invalid port", key: "EGTS_VALKEY_PORT_PRIMARY", value: "server", errorText: "PORT_PRIMARY"},
		{name: "zero port", key: "EGTS_VALKEY_PORT_PRIMARY", value: "0", errorText: "port"},
		{name: "large port", key: "EGTS_VALKEY_PORT_PRIMARY", value: "65536", errorText: "port"},
		{name: "username without password", key: "EGTS_VALKEY_PASSWORD_PRIMARY", value: "", errorText: "password"},
		{name: "invalid database", key: "EGTS_VALKEY_DATABASE_PRIMARY", value: "db", errorText: "DATABASE_PRIMARY"},
		{name: "negative database", key: "EGTS_VALKEY_DATABASE_PRIMARY", value: "-1", errorText: "database"},
		{name: "empty stream", key: "EGTS_VALKEY_STREAM_NAME_PRIMARY", value: "", errorText: "stream_name"},
		{name: "invalid switch", key: "EGTS_VALKEY_ENABLED_PRIMARY", value: "yes", errorText: "ENABLED_PRIMARY"},
		{name: "invalid connect timeout", key: "EGTS_VALKEY_CONNECT_TIMEOUT_SECONDS_PRIMARY", value: "a", errorText: "CONNECT_TIMEOUT"},
		{name: "zero connect timeout", key: "EGTS_VALKEY_CONNECT_TIMEOUT_SECONDS_PRIMARY", value: "0", errorText: "connect_timeout"},
		{name: "large connect timeout", key: "EGTS_VALKEY_CONNECT_TIMEOUT_SECONDS_PRIMARY", value: "9223372037", errorText: "connect_timeout"},
		{name: "invalid write timeout", key: "EGTS_VALKEY_WRITE_TIMEOUT_SECONDS_PRIMARY", value: "a", errorText: "WRITE_TIMEOUT"},
		{name: "zero write timeout", key: "EGTS_VALKEY_WRITE_TIMEOUT_SECONDS_PRIMARY", value: "0", errorText: "write_timeout"},
		{name: "large write timeout", key: "EGTS_VALKEY_WRITE_TIMEOUT_SECONDS_PRIMARY", value: "9223372037", errorText: "write_timeout"},
		{name: "unknown field", key: "EGTS_VALKEY_POTR_PRIMARY", value: "6379", errorText: "Unknown Valkey variable"},
		{name: "unknown ID", key: "EGTS_VALKEY_PORT_BACKUP", value: "6380", errorText: "Unknown Valkey variable"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Setenv("EGTS_VALKEY_IDS", "primary")
			t.Setenv("EGTS_VALKEY_ENABLED_PRIMARY", "true")
			t.Setenv("EGTS_VALKEY_USERNAME_PRIMARY", "user")
			t.Setenv("EGTS_VALKEY_PASSWORD_PRIMARY", "secret")
			t.Setenv(tc.key, tc.value)
			cfg, err := configuration.PrepareEnvConfiguration()
			assert.ErrorContains(t, err, tc.errorText)
			assert.Nil(t, cfg)
			if err != nil {
				assert.NotContains(t, err.Error(), "secret")
			}
		})
	}
	for _, field := range []string{"connect_timeout_seconds = 0", "write_timeout_seconds = 0", "database = -1", "unknown = true"} {
		data := "[[destinations_cfg.valkey]]\nid = \"primary\"\nenabled = true\n" + field + "\n"
		fname := filepath.Join(t.TempDir(), "valkey.toml")
		err := os.WriteFile(fname, []byte(data), 0600)
		assert.NoError(t, err)
		cfg, err := configuration.PrepareFileConfiguration(fname)
		assert.Error(t, err)
		assert.Nil(t, cfg)
	}
}
