package configuration_test

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/LdDl/go-egts/gateway/configuration"
	"github.com/stretchr/testify/assert"
)

func TestRabbitMQConfiguration(t *testing.T) {
	data := `[destinations_cfg]
stdout = false
[[destinations_cfg.rabbitmq]]
id = "monitoring"
enabled = true
username = "first"
password = "first password"
vhost = "egts"
queue_name = "first.packets"
queue_type = "quorum"
[[destinations_cfg.rabbitmq]]
id = "backup"
enabled = true
host = "localhost"
port = 5673
username = "second"
password = "second password"
connect_timeout_seconds = 3
publish_timeout_seconds = 7
`
	fname := filepath.Join(t.TempDir(), "rabbitmq.toml")
	err := os.WriteFile(fname, []byte(data), 0600)
	assert.NoError(t, err)
	file, err := configuration.PrepareFileConfiguration(fname)
	assert.NoError(t, err)
	if err != nil {
		return
	}
	want := []configuration.RabbitMQDestinationConf{
		{ID: "monitoring", Enabled: true, Host: "127.0.0.1", Port: 5672, UserName: "first", Password: "first password", Vhost: "egts", QueueName: "first.packets", QueueType: "quorum", ConnectTimeoutSeconds: 5, PublishTimeoutSeconds: 10},
		{ID: "backup", Enabled: true, Host: "localhost", Port: 5673, UserName: "second", Password: "second password", Vhost: "/", QueueName: "egts.packets", QueueType: "classic", ConnectTimeoutSeconds: 3, PublishTimeoutSeconds: 7},
	}
	assert.Equal(t, want, file.DestinationsCfg.RabbitMQ)
	values := map[string]string{
		"EGTS_PACKETS_STDOUT": "false", "EGTS_RABBITMQ_IDS": "monitoring,backup",
		"EGTS_RABBITMQ_ENABLED_MONITORING": "true", "EGTS_RABBITMQ_USERNAME_MONITORING": "first", "EGTS_RABBITMQ_PASSWORD_MONITORING": "first password",
		"EGTS_RABBITMQ_VHOST_MONITORING": "egts", "EGTS_RABBITMQ_QUEUE_NAME_MONITORING": "first.packets", "EGTS_RABBITMQ_QUEUE_TYPE_MONITORING": "quorum",
		"EGTS_RABBITMQ_ENABLED_BACKUP": "true", "EGTS_RABBITMQ_HOST_BACKUP": "localhost", "EGTS_RABBITMQ_PORT_BACKUP": "5673",
		"EGTS_RABBITMQ_USERNAME_BACKUP": "second", "EGTS_RABBITMQ_PASSWORD_BACKUP": "second password",
		"EGTS_RABBITMQ_CONNECT_TIMEOUT_SECONDS_BACKUP": "3", "EGTS_RABBITMQ_PUBLISH_TIMEOUT_SECONDS_BACKUP": "7",
	}
	for key, value := range values {
		t.Setenv(key, value)
	}
	env, err := configuration.PrepareEnvConfiguration()
	assert.NoError(t, err)
	assert.Equal(t, file, env)
}

func TestRabbitMQConfigurationErrors(t *testing.T) {
	cases := []envConfigurationTestCase{
		{name: "duplicate IDs", key: "EGTS_RABBITMQ_IDS", value: "primary,primary", errorText: "Duplicate"},
		{name: "empty ID", key: "EGTS_RABBITMQ_IDS", value: "primary,", errorText: "must not be empty"},
		{name: "invalid ID", key: "EGTS_RABBITMQ_IDS", value: "primary,UPPER", errorText: "lowercase"},
		{name: "empty host", key: "EGTS_RABBITMQ_HOST_PRIMARY", value: "", errorText: "host"},
		{name: "invalid port", key: "EGTS_RABBITMQ_PORT_PRIMARY", value: "rabbit", errorText: "PORT_PRIMARY"},
		{name: "zero port", key: "EGTS_RABBITMQ_PORT_PRIMARY", value: "0", errorText: "port"},
		{name: "empty user", key: "EGTS_RABBITMQ_USERNAME_PRIMARY", value: "", errorText: "username"},
		{name: "empty password", key: "EGTS_RABBITMQ_PASSWORD_PRIMARY", value: "", errorText: "password"},
		{name: "empty vhost", key: "EGTS_RABBITMQ_VHOST_PRIMARY", value: "", errorText: "vhost"},
		{name: "long queue", key: "EGTS_RABBITMQ_QUEUE_NAME_PRIMARY", value: strings.Repeat("q", 256), errorText: "queue_name"},
		{name: "reserved queue", key: "EGTS_RABBITMQ_QUEUE_NAME_PRIMARY", value: "amq.test", errorText: "amq."},
		{name: "invalid queue type", key: "EGTS_RABBITMQ_QUEUE_TYPE_PRIMARY", value: "unknown", errorText: "queue_type"},
		{name: "invalid switch", key: "EGTS_RABBITMQ_ENABLED_PRIMARY", value: "yes", errorText: "ENABLED_PRIMARY"},
		{name: "invalid connect timeout", key: "EGTS_RABBITMQ_CONNECT_TIMEOUT_SECONDS_PRIMARY", value: "a", errorText: "CONNECT_TIMEOUT"},
		{name: "zero connect timeout", key: "EGTS_RABBITMQ_CONNECT_TIMEOUT_SECONDS_PRIMARY", value: "0", errorText: "connect_timeout"},
		{name: "invalid publish timeout", key: "EGTS_RABBITMQ_PUBLISH_TIMEOUT_SECONDS_PRIMARY", value: "a", errorText: "PUBLISH_TIMEOUT"},
		{name: "zero publish timeout", key: "EGTS_RABBITMQ_PUBLISH_TIMEOUT_SECONDS_PRIMARY", value: "0", errorText: "publish_timeout"},
		{name: "unknown field", key: "EGTS_RABBITMQ_POTR_PRIMARY", value: "5672", errorText: "Unknown RabbitMQ variable"},
		{name: "unknown ID", key: "EGTS_RABBITMQ_PORT_BACKUP", value: "5673", errorText: "Unknown RabbitMQ variable"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Setenv("EGTS_RABBITMQ_IDS", "primary")
			t.Setenv("EGTS_RABBITMQ_ENABLED_PRIMARY", "true")
			t.Setenv("EGTS_RABBITMQ_USERNAME_PRIMARY", "user")
			t.Setenv("EGTS_RABBITMQ_PASSWORD_PRIMARY", "secret")
			t.Setenv(tc.key, tc.value)
			cfg, err := configuration.PrepareEnvConfiguration()
			assert.ErrorContains(t, err, tc.errorText)
			assert.NotContains(t, err.Error(), "secret")
			assert.Nil(t, cfg)
		})
	}
	for _, field := range []string{"connect_timeout_seconds = 0", "publish_timeout_seconds = 0", "unknown = true"} {
		data := "[[destinations_cfg.rabbitmq]]\nid = \"primary\"\nenabled = true\nusername = \"user\"\npassword = \"secret\"\n" + field + "\n"
		fname := filepath.Join(t.TempDir(), "rabbit.toml")
		err := os.WriteFile(fname, []byte(data), 0600)
		assert.NoError(t, err)
		cfg, err := configuration.PrepareFileConfiguration(fname)
		assert.Error(t, err)
		assert.Nil(t, cfg)
	}
}
