package configuration

import (
	"fmt"
	"os"
	"strconv"
	"strings"
)

type RabbitMQDestinationConf struct {
	ID                    string `toml:"id" json:"id"`
	Enabled               bool   `toml:"enabled" json:"enabled"`
	Host                  string `toml:"host" json:"host"`
	Port                  int    `toml:"port" json:"port"`
	UserName              string `toml:"username" json:"username"`
	Password              string `toml:"password" json:"password"`
	Vhost                 string `toml:"vhost" json:"vhost"`
	QueueName             string `toml:"queue_name" json:"queue_name"`
	QueueType             string `toml:"queue_type" json:"queue_type"`
	ConnectTimeoutSeconds int    `toml:"connect_timeout_seconds" json:"connect_timeout_seconds"`
	PublishTimeoutSeconds int    `toml:"publish_timeout_seconds" json:"publish_timeout_seconds"`
}

func DefaultRabbitMQDestination() RabbitMQDestinationConf {
	return RabbitMQDestinationConf{
		Host: "127.0.0.1", Port: 5672, Vhost: "/", QueueName: "egts.packets", QueueType: "classic",
		ConnectTimeoutSeconds: 5, PublishTimeoutSeconds: 10,
	}
}

func (cfg RabbitMQDestinationConf) Validate() error {
	if cfg.ID == "" {
		return fmt.Errorf("destinations_cfg.rabbitmq.id must not be empty")
	}
	for _, character := range cfg.ID {
		if (character < 'a' || character > 'z') && (character < '0' || character > '9') && character != '_' {
			return fmt.Errorf("destinations_cfg.rabbitmq.id may contain only lowercase Latin letters, digits and underscores")
		}
	}
	if !cfg.Enabled {
		return nil
	}
	if strings.TrimSpace(cfg.Host) == "" {
		return fmt.Errorf("destinations_cfg.rabbitmq.host must not be empty")
	}
	if cfg.Port < 1 || cfg.Port > 65535 {
		return fmt.Errorf("destinations_cfg.rabbitmq.port must be between 1 and 65535")
	}
	if cfg.UserName == "" || strings.ContainsRune(cfg.UserName, 0) {
		return fmt.Errorf("destinations_cfg.rabbitmq.username must not be empty or contain a null byte")
	}
	if cfg.Password == "" || strings.ContainsRune(cfg.Password, 0) {
		return fmt.Errorf("destinations_cfg.rabbitmq.password must not be empty or contain a null byte")
	}
	if cfg.Vhost == "" || len(cfg.Vhost) > 255 || strings.ContainsRune(cfg.Vhost, 0) {
		return fmt.Errorf("destinations_cfg.rabbitmq.vhost must contain 1 to 255 bytes without a null byte")
	}
	if cfg.QueueName == "" || len(cfg.QueueName) > 255 || strings.ContainsRune(cfg.QueueName, 0) || strings.HasPrefix(cfg.QueueName, "amq.") {
		return fmt.Errorf("destinations_cfg.rabbitmq.queue_name must contain 1 to 255 bytes without a null byte and must not start with amq.")
	}
	if cfg.QueueType != "classic" && cfg.QueueType != "quorum" {
		return fmt.Errorf("destinations_cfg.rabbitmq.queue_type must be classic or quorum")
	}
	if cfg.ConnectTimeoutSeconds < 1 || int64(cfg.ConnectTimeoutSeconds) > 9223372036 {
		return fmt.Errorf("destinations_cfg.rabbitmq.connect_timeout_seconds must be between 1 and 9223372036")
	}
	if cfg.PublishTimeoutSeconds < 1 || int64(cfg.PublishTimeoutSeconds) > 9223372036 {
		return fmt.Errorf("destinations_cfg.rabbitmq.publish_timeout_seconds must be between 1 and 9223372036")
	}
	return nil
}

func prepareRabbitMQEnvConfiguration() ([]RabbitMQDestinationConf, error) {
	var destinations []RabbitMQDestinationConf
	ids, exists := os.LookupEnv("EGTS_RABBITMQ_IDS")
	known := map[string]bool{"EGTS_RABBITMQ_IDS": true}
	if exists && ids != "" {
		for _, id := range strings.Split(ids, ",") {
			cfg := DefaultRabbitMQDestination()
			cfg.ID = strings.TrimSpace(id)
			suffix := "_" + strings.ToUpper(cfg.ID)
			key := "EGTS_RABBITMQ_ENABLED" + suffix
			known[key] = true
			value, found := os.LookupEnv(key)
			if found {
				parsed, err := strconv.ParseBool(value)
				if err != nil {
					return nil, fmt.Errorf("%s: %w", key, err)
				}
				cfg.Enabled = parsed
			}
			key = "EGTS_RABBITMQ_HOST" + suffix
			known[key] = true
			value, found = os.LookupEnv(key)
			if found {
				cfg.Host = value
			}
			key = "EGTS_RABBITMQ_PORT" + suffix
			known[key] = true
			value, found = os.LookupEnv(key)
			if found {
				parsed, err := strconv.Atoi(value)
				if err != nil {
					return nil, fmt.Errorf("%s: %w", key, err)
				}
				cfg.Port = parsed
			}
			key = "EGTS_RABBITMQ_USERNAME" + suffix
			known[key] = true
			value, found = os.LookupEnv(key)
			if found {
				cfg.UserName = value
			}
			key = "EGTS_RABBITMQ_PASSWORD" + suffix
			known[key] = true
			value, found = os.LookupEnv(key)
			if found {
				cfg.Password = value
			}
			key = "EGTS_RABBITMQ_VHOST" + suffix
			known[key] = true
			value, found = os.LookupEnv(key)
			if found {
				cfg.Vhost = value
			}
			key = "EGTS_RABBITMQ_QUEUE_NAME" + suffix
			known[key] = true
			value, found = os.LookupEnv(key)
			if found {
				cfg.QueueName = value
			}
			key = "EGTS_RABBITMQ_QUEUE_TYPE" + suffix
			known[key] = true
			value, found = os.LookupEnv(key)
			if found {
				cfg.QueueType = value
			}
			key = "EGTS_RABBITMQ_CONNECT_TIMEOUT_SECONDS" + suffix
			known[key] = true
			value, found = os.LookupEnv(key)
			if found {
				parsed, err := strconv.Atoi(value)
				if err != nil {
					return nil, fmt.Errorf("%s: %w", key, err)
				}
				cfg.ConnectTimeoutSeconds = parsed
			}
			key = "EGTS_RABBITMQ_PUBLISH_TIMEOUT_SECONDS" + suffix
			known[key] = true
			value, found = os.LookupEnv(key)
			if found {
				parsed, err := strconv.Atoi(value)
				if err != nil {
					return nil, fmt.Errorf("%s: %w", key, err)
				}
				cfg.PublishTimeoutSeconds = parsed
			}
			destinations = append(destinations, cfg)
		}
	}
	for _, variable := range os.Environ() {
		key, _, _ := strings.Cut(variable, "=")
		if strings.HasPrefix(key, "EGTS_RABBITMQ_") && !known[key] {
			return nil, fmt.Errorf("Unknown RabbitMQ variable %s; use EGTS_RABBITMQ_IDS and a destination ID suffix", key)
		}
	}
	return destinations, nil
}
