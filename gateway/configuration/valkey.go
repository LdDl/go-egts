package configuration

import (
	"fmt"
	"os"
	"strconv"
	"strings"
)

type ValkeyDestinationConf struct {
	ID                    string `toml:"id" json:"id"`
	Enabled               bool   `toml:"enabled" json:"enabled"`
	Host                  string `toml:"host" json:"host"`
	Port                  int    `toml:"port" json:"port"`
	UserName              string `toml:"username" json:"username"`
	Password              string `toml:"password" json:"password"`
	Database              int    `toml:"database" json:"database"`
	StreamName            string `toml:"stream_name" json:"stream_name"`
	ConnectTimeoutSeconds int    `toml:"connect_timeout_seconds" json:"connect_timeout_seconds"`
	WriteTimeoutSeconds   int    `toml:"write_timeout_seconds" json:"write_timeout_seconds"`
}

func DefaultValkeyDestination() ValkeyDestinationConf {
	return ValkeyDestinationConf{
		Host: "127.0.0.1", Port: 6379, StreamName: "egts.packets",
		ConnectTimeoutSeconds: 5, WriteTimeoutSeconds: 10,
	}
}

func (cfg ValkeyDestinationConf) Validate() error {
	if cfg.ID == "" {
		return fmt.Errorf("destinations_cfg.valkey.id must not be empty")
	}
	for _, character := range cfg.ID {
		if (character < 'a' || character > 'z') && (character < '0' || character > '9') && character != '_' {
			return fmt.Errorf("destinations_cfg.valkey.id may contain only lowercase Latin letters, digits and underscores")
		}
	}
	if !cfg.Enabled {
		return nil
	}
	if strings.TrimSpace(cfg.Host) == "" {
		return fmt.Errorf("destinations_cfg.valkey.host must not be empty")
	}
	if cfg.Port < 1 || cfg.Port > 65535 {
		return fmt.Errorf("destinations_cfg.valkey.port must be between 1 and 65535")
	}
	if cfg.UserName != "" && cfg.Password == "" {
		return fmt.Errorf("destinations_cfg.valkey.password is required when username is set")
	}
	if cfg.Database < 0 {
		return fmt.Errorf("destinations_cfg.valkey.database must not be negative")
	}
	if strings.TrimSpace(cfg.StreamName) == "" {
		return fmt.Errorf("destinations_cfg.valkey.stream_name must not be empty")
	}
	if cfg.ConnectTimeoutSeconds < 1 || int64(cfg.ConnectTimeoutSeconds) > 9223372036 {
		return fmt.Errorf("destinations_cfg.valkey.connect_timeout_seconds must be between 1 and 9223372036")
	}
	if cfg.WriteTimeoutSeconds < 1 || int64(cfg.WriteTimeoutSeconds) > 9223372036 {
		return fmt.Errorf("destinations_cfg.valkey.write_timeout_seconds must be between 1 and 9223372036")
	}
	return nil
}

func prepareValkeyEnvConfiguration() ([]ValkeyDestinationConf, error) {
	var destinations []ValkeyDestinationConf
	ids, exists := os.LookupEnv("EGTS_VALKEY_IDS")
	known := map[string]bool{"EGTS_VALKEY_IDS": true}
	if exists && ids != "" {
		for _, id := range strings.Split(ids, ",") {
			cfg := DefaultValkeyDestination()
			cfg.ID = strings.TrimSpace(id)
			suffix := "_" + strings.ToUpper(cfg.ID)
			key := "EGTS_VALKEY_ENABLED" + suffix
			known[key] = true
			value, found := os.LookupEnv(key)
			if found {
				parsed, err := strconv.ParseBool(value)
				if err != nil {
					return nil, fmt.Errorf("%s: %w", key, err)
				}
				cfg.Enabled = parsed
			}
			key = "EGTS_VALKEY_HOST" + suffix
			known[key] = true
			value, found = os.LookupEnv(key)
			if found {
				cfg.Host = value
			}
			key = "EGTS_VALKEY_PORT" + suffix
			known[key] = true
			value, found = os.LookupEnv(key)
			if found {
				parsed, err := strconv.Atoi(value)
				if err != nil {
					return nil, fmt.Errorf("%s: %w", key, err)
				}
				cfg.Port = parsed
			}
			key = "EGTS_VALKEY_USERNAME" + suffix
			known[key] = true
			value, found = os.LookupEnv(key)
			if found {
				cfg.UserName = value
			}
			key = "EGTS_VALKEY_PASSWORD" + suffix
			known[key] = true
			value, found = os.LookupEnv(key)
			if found {
				cfg.Password = value
			}
			key = "EGTS_VALKEY_DATABASE" + suffix
			known[key] = true
			value, found = os.LookupEnv(key)
			if found {
				parsed, err := strconv.Atoi(value)
				if err != nil {
					return nil, fmt.Errorf("%s: %w", key, err)
				}
				cfg.Database = parsed
			}
			key = "EGTS_VALKEY_STREAM_NAME" + suffix
			known[key] = true
			value, found = os.LookupEnv(key)
			if found {
				cfg.StreamName = value
			}
			key = "EGTS_VALKEY_CONNECT_TIMEOUT_SECONDS" + suffix
			known[key] = true
			value, found = os.LookupEnv(key)
			if found {
				parsed, err := strconv.Atoi(value)
				if err != nil {
					return nil, fmt.Errorf("%s: %w", key, err)
				}
				cfg.ConnectTimeoutSeconds = parsed
			}
			key = "EGTS_VALKEY_WRITE_TIMEOUT_SECONDS" + suffix
			known[key] = true
			value, found = os.LookupEnv(key)
			if found {
				parsed, err := strconv.Atoi(value)
				if err != nil {
					return nil, fmt.Errorf("%s: %w", key, err)
				}
				cfg.WriteTimeoutSeconds = parsed
			}
			destinations = append(destinations, cfg)
		}
	}
	for _, variable := range os.Environ() {
		key, _, _ := strings.Cut(variable, "=")
		if strings.HasPrefix(key, "EGTS_VALKEY_") && !known[key] {
			return nil, fmt.Errorf("Unknown Valkey variable %s; use EGTS_VALKEY_IDS and a destination ID suffix", key)
		}
	}
	return destinations, nil
}
