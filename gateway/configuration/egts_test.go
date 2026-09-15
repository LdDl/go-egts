package configuration_test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/LdDl/go-egts/gateway/configuration"
	"github.com/stretchr/testify/assert"
)

func TestEGTSConfiguration(t *testing.T) {
	fname := filepath.Join(t.TempDir(), "relay.toml")
	data := `[destinations_cfg]
stdout = false
[[destinations_cfg.egts]]
id = "primary"
enabled = true
host = "localhost"
port = 9000
connect_timeout_seconds = 2
ack_timeout_seconds = 3
[destinations_cfg.egts.auth]
enabled = true
username = "relay"
password = "remote password"
`
	err := os.WriteFile(fname, []byte(data), 0600)
	assert.NoError(t, err)
	cfg, err := configuration.PrepareFileConfiguration(fname)
	assert.NoError(t, err)
	if err != nil {
		return
	}
	want := configuration.EGTSDestinationConf{
		ID: "primary", Enabled: true, Host: "localhost", Port: 9000, ConnectTimeoutSeconds: 2, AckTimeoutSeconds: 3,
		Auth: configuration.EGTSAuthConf{Enabled: true, UserName: "relay", Password: "remote password"},
	}
	assert.Equal(t, []configuration.EGTSDestinationConf{want}, cfg.DestinationsCfg.EGTS)
	values := map[string]string{
		"EGTS_RELAY_IDS":             "primary",
		"EGTS_RELAY_ENABLED_PRIMARY": "true", "EGTS_RELAY_HOST_PRIMARY": "localhost", "EGTS_RELAY_PORT_PRIMARY": "9000",
		"EGTS_RELAY_CONNECT_TIMEOUT_SECONDS_PRIMARY": "2", "EGTS_RELAY_ACK_TIMEOUT_SECONDS_PRIMARY": "3",
		"EGTS_RELAY_AUTH_ENABLED_PRIMARY": "true", "EGTS_RELAY_AUTH_USERNAME_PRIMARY": "relay", "EGTS_RELAY_AUTH_PASSWORD_PRIMARY": "remote password",
	}
	for key, value := range values {
		t.Setenv(key, value)
	}
	cfg, err = configuration.PrepareEnvConfiguration()
	assert.NoError(t, err)
	if err != nil {
		return
	}
	assert.Equal(t, []configuration.EGTSDestinationConf{want}, cfg.DestinationsCfg.EGTS)
}

func TestEGTSConfigurationErrors(t *testing.T) {
	cases := []envConfigurationTestCase{
		{name: "invalid switch", key: "EGTS_RELAY_ENABLED_PRIMARY", value: "maybe", errorText: "EGTS_RELAY_ENABLED_PRIMARY"},
		{name: "empty host", key: "EGTS_RELAY_HOST_PRIMARY", value: "", errorText: "egts.host"},
		{name: "invalid port", key: "EGTS_RELAY_PORT_PRIMARY", value: "tcp", errorText: "EGTS_RELAY_PORT_PRIMARY"},
		{name: "large port", key: "EGTS_RELAY_PORT_PRIMARY", value: "65536", errorText: "egts.port"},
		{name: "zero port", key: "EGTS_RELAY_PORT_PRIMARY", value: "0", errorText: "egts.port"},
		{name: "invalid connect timeout", key: "EGTS_RELAY_CONNECT_TIMEOUT_SECONDS_PRIMARY", value: "1s", errorText: "EGTS_RELAY_CONNECT_TIMEOUT_SECONDS_PRIMARY"},
		{name: "negative connect timeout", key: "EGTS_RELAY_CONNECT_TIMEOUT_SECONDS_PRIMARY", value: "-1", errorText: "egts.connect_timeout_seconds"},
		{name: "invalid ACK timeout", key: "EGTS_RELAY_ACK_TIMEOUT_SECONDS_PRIMARY", value: "many", errorText: "EGTS_RELAY_ACK_TIMEOUT_SECONDS_PRIMARY"},
		{name: "zero ACK timeout", key: "EGTS_RELAY_ACK_TIMEOUT_SECONDS_PRIMARY", value: "0", errorText: "egts.ack_timeout_seconds"},
		{name: "invalid auth switch", key: "EGTS_RELAY_AUTH_ENABLED_PRIMARY", value: "maybe", errorText: "EGTS_RELAY_AUTH_ENABLED_PRIMARY"},
		{name: "missing password", key: "EGTS_RELAY_AUTH_ENABLED_PRIMARY", value: "true", errorText: "egts.auth.password"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Setenv("EGTS_RELAY_IDS", "primary")
			t.Setenv("EGTS_RELAY_ENABLED_PRIMARY", "true")
			t.Setenv("EGTS_RELAY_AUTH_PASSWORD_PRIMARY", "")
			t.Setenv(tc.key, tc.value)
			cfg, err := configuration.PrepareEnvConfiguration()
			assert.ErrorContains(t, err, tc.errorText)
			assert.Nil(t, cfg)
		})
	}
}

type relayListTestCase struct {
	name      string
	ids       string
	key       string
	value     string
	errorText string
}

func TestEGTSListDefaultsAndEnvironment(t *testing.T) {
	fname := filepath.Join(t.TempDir(), "relay.toml")
	data := `[[destinations_cfg.egts]]
id = "monitoring"
enabled = true
host = "192.0.2.1"
port = 9001
[destinations_cfg.egts.auth]
enabled = true
username = "first"
password = "first password"
[[destinations_cfg.egts]]
id = "backup"
enabled = true
ack_timeout_seconds = 7
`
	err := os.WriteFile(fname, []byte(data), 0600)
	assert.NoError(t, err)
	file, err := configuration.PrepareFileConfiguration(fname)
	assert.NoError(t, err)
	if err != nil {
		return
	}
	assert.Len(t, file.DestinationsCfg.EGTS, 2)
	assert.Equal(t, 5, file.DestinationsCfg.EGTS[0].ConnectTimeoutSeconds)
	assert.Equal(t, 10, file.DestinationsCfg.EGTS[0].AckTimeoutSeconds)
	assert.Equal(t, "127.0.0.1", file.DestinationsCfg.EGTS[1].Host)
	assert.Equal(t, 8082, file.DestinationsCfg.EGTS[1].Port)
	assert.False(t, file.DestinationsCfg.EGTS[1].Auth.Enabled)
	values := map[string]string{
		"EGTS_RELAY_IDS":                        "monitoring, backup",
		"EGTS_RELAY_ENABLED_MONITORING":         "true",
		"EGTS_RELAY_HOST_MONITORING":            "192.0.2.1",
		"EGTS_RELAY_PORT_MONITORING":            "9001",
		"EGTS_RELAY_AUTH_ENABLED_MONITORING":    "true",
		"EGTS_RELAY_AUTH_USERNAME_MONITORING":   "first",
		"EGTS_RELAY_AUTH_PASSWORD_MONITORING":   "first password",
		"EGTS_RELAY_ENABLED_BACKUP":             "true",
		"EGTS_RELAY_ACK_TIMEOUT_SECONDS_BACKUP": "7",
	}
	for key, value := range values {
		t.Setenv(key, value)
	}
	env, err := configuration.PrepareEnvConfiguration()
	assert.NoError(t, err)
	assert.Equal(t, file, env)
}

func TestEGTSListErrors(t *testing.T) {
	cases := []relayListTestCase{
		{name: "duplicate ID", ids: "primary,primary", errorText: "Duplicate"},
		{name: "empty ID", ids: "primary,", errorText: "id must not be empty"},
		{name: "uppercase ID", ids: "Primary", errorText: "lowercase"},
		{name: "invalid ID", ids: "primary-backup", errorText: "underscores"},
		{name: "unknown ID", ids: "primary", key: "EGTS_RELAY_HOST_BACKUP", value: "localhost", errorText: "Unknown relay variable"},
		{name: "unknown field", ids: "primary", key: "EGTS_RELAY_POTR_PRIMARY", value: "8082", errorText: "Unknown relay variable"},
		{name: "legacy key", ids: "primary", key: "EGTS_RELAY_ENABLED", value: "true", errorText: "ID suffix"},
		{name: "no ID list", key: "EGTS_RELAY_ENABLED_PRIMARY", value: "true", errorText: "EGTS_RELAY_IDS"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Setenv("EGTS_RELAY_IDS", tc.ids)
			if tc.key != "" {
				t.Setenv(tc.key, tc.value)
			}
			cfg, err := configuration.PrepareEnvConfiguration()
			assert.ErrorContains(t, err, tc.errorText)
			assert.Nil(t, cfg)
		})
	}
	for _, data := range []string{
		"[[destinations_cfg.egts]]\nid = \"one\"\n[[destinations_cfg.egts]]\nid = \"one\"\n",
		"[[destinations_cfg.egts]]\nenabled = false\n",
		"[[destinations_cfg.egts]]\nid = \"one\"\nenabled = true\nport = 0\n",
		"[[destinations_cfg.egts]]\nid = \"one\"\n[[destinations_cfg.egts]]\nid = \"two\"\nenabled = true\nack_timeout_seconds = 0\n",
		"[[destinations_cfg.egts]]\nid = \"one\"\n[[destinations_cfg.egts]]\nid = \"two\"\nunknown = true\n",
		"[destinations_cfg.egts]\nenabled = true\n",
	} {
		fname := filepath.Join(t.TempDir(), "relay.toml")
		err := os.WriteFile(fname, []byte(data), 0600)
		assert.NoError(t, err)
		cfg, err := configuration.PrepareFileConfiguration(fname)
		assert.Error(t, err)
		assert.Nil(t, cfg)
	}
}
