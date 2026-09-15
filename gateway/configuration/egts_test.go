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
[destinations_cfg.egts]
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
		Enabled: true, Host: "localhost", Port: 9000, ConnectTimeoutSeconds: 2, AckTimeoutSeconds: 3,
		Auth: configuration.EGTSAuthConf{Enabled: true, UserName: "relay", Password: "remote password"},
	}
	assert.Equal(t, want, cfg.DestinationsCfg.EGTS)
	values := map[string]string{
		"EGTS_RELAY_ENABLED": "true", "EGTS_RELAY_HOST": "localhost", "EGTS_RELAY_PORT": "9000",
		"EGTS_RELAY_CONNECT_TIMEOUT_SECONDS": "2", "EGTS_RELAY_ACK_TIMEOUT_SECONDS": "3",
		"EGTS_RELAY_AUTH_ENABLED": "true", "EGTS_RELAY_AUTH_USERNAME": "relay", "EGTS_RELAY_AUTH_PASSWORD": "remote password",
	}
	for key, value := range values {
		t.Setenv(key, value)
	}
	cfg, err = configuration.PrepareEnvConfiguration()
	assert.NoError(t, err)
	if err != nil {
		return
	}
	assert.Equal(t, want, cfg.DestinationsCfg.EGTS)
}

func TestEGTSConfigurationErrors(t *testing.T) {
	cases := []envConfigurationTestCase{
		{name: "invalid switch", key: "EGTS_RELAY_ENABLED", value: "maybe", errorText: "EGTS_RELAY_ENABLED"},
		{name: "empty host", key: "EGTS_RELAY_HOST", value: "", errorText: "egts.host"},
		{name: "invalid port", key: "EGTS_RELAY_PORT", value: "tcp", errorText: "EGTS_RELAY_PORT"},
		{name: "large port", key: "EGTS_RELAY_PORT", value: "65536", errorText: "egts.port"},
		{name: "zero port", key: "EGTS_RELAY_PORT", value: "0", errorText: "egts.port"},
		{name: "invalid connect timeout", key: "EGTS_RELAY_CONNECT_TIMEOUT_SECONDS", value: "1s", errorText: "EGTS_RELAY_CONNECT_TIMEOUT_SECONDS"},
		{name: "negative connect timeout", key: "EGTS_RELAY_CONNECT_TIMEOUT_SECONDS", value: "-1", errorText: "egts.connect_timeout_seconds"},
		{name: "invalid ACK timeout", key: "EGTS_RELAY_ACK_TIMEOUT_SECONDS", value: "many", errorText: "EGTS_RELAY_ACK_TIMEOUT_SECONDS"},
		{name: "zero ACK timeout", key: "EGTS_RELAY_ACK_TIMEOUT_SECONDS", value: "0", errorText: "egts.ack_timeout_seconds"},
		{name: "invalid auth switch", key: "EGTS_RELAY_AUTH_ENABLED", value: "maybe", errorText: "EGTS_RELAY_AUTH_ENABLED"},
		{name: "missing password", key: "EGTS_RELAY_AUTH_ENABLED", value: "true", errorText: "egts.auth.password"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Setenv("EGTS_RELAY_ENABLED", "true")
			t.Setenv("EGTS_RELAY_AUTH_PASSWORD", "")
			t.Setenv(tc.key, tc.value)
			cfg, err := configuration.PrepareEnvConfiguration()
			assert.ErrorContains(t, err, tc.errorText)
			assert.Nil(t, cfg)
		})
	}
}
