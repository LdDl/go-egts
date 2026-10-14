package configuration_test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/LdDl/go-egts/gateway/configuration"
	"github.com/stretchr/testify/assert"
)

func TestServiceVersionConfiguration(t *testing.T) {
	name := filepath.Join(t.TempDir(), "v2.toml")
	if !assert.NoError(t, os.WriteFile(name, []byte("[server_cfg]\nservice_version = 2\n"), 0600)) {
		return
	}
	cfg, err := configuration.PrepareFileConfiguration(name)
	if !assert.NoError(t, err) {
		return
	}
	assert.Equal(t, uint8(2), cfg.ServerCfg.ServiceVersion)
	cfg.ServerCfg.ServiceVersion = 3
	assert.ErrorContains(t, cfg.Validate(), "service_version")
	t.Setenv("EGTS_SERVICE_VERSION", "2")
	cfg, err = configuration.PrepareEnvConfiguration()
	if !assert.NoError(t, err) {
		return
	}
	assert.Equal(t, uint8(2), cfg.ServerCfg.ServiceVersion)
	t.Setenv("EGTS_SERVICE_VERSION", "256")
	_, err = configuration.PrepareEnvConfiguration()
	assert.Error(t, err)
	t.Setenv("EGTS_SERVICE_VERSION", "3")
	_, err = configuration.PrepareEnvConfiguration()
	assert.Error(t, err)
}
