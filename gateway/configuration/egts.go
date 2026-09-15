package configuration

import (
	"fmt"
	"strings"
)

func (cfg EGTSDestinationConf) Validate() error {
	if !cfg.Enabled {
		return nil
	}
	if strings.TrimSpace(cfg.Host) == "" {
		return fmt.Errorf("destinations_cfg.egts.host must not be empty")
	}
	if cfg.Port < 1 || cfg.Port > 65535 {
		return fmt.Errorf("destinations_cfg.egts.port must be between 1 and 65535")
	}
	if cfg.ConnectTimeoutSeconds < 1 || int64(cfg.ConnectTimeoutSeconds) > 9223372036 {
		return fmt.Errorf("destinations_cfg.egts.connect_timeout_seconds must be between 1 and 9223372036")
	}
	if cfg.AckTimeoutSeconds < 1 || int64(cfg.AckTimeoutSeconds) > 9223372036 {
		return fmt.Errorf("destinations_cfg.egts.ack_timeout_seconds must be between 1 and 9223372036")
	}
	if cfg.Auth.Enabled {
		if len(cfg.Auth.UserName) > 32 || strings.ContainsRune(cfg.Auth.UserName, 0) {
			return fmt.Errorf("destinations_cfg.egts.auth.username must not exceed 32 bytes or contain a null byte")
		}
		if cfg.Auth.Password == "" || len(cfg.Auth.Password) > 32 || strings.ContainsRune(cfg.Auth.Password, 0) {
			return fmt.Errorf("destinations_cfg.egts.auth.password must contain 1 to 32 bytes without a null byte")
		}
	}
	return nil
}
