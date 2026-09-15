package configuration

import (
	"errors"
	"flag"
	"fmt"
	"os"
	"strconv"
	"strings"

	"github.com/BurntSushi/toml"
	"github.com/joho/godotenv"
)

type Configuration struct {
	ServerCfg       ServerConf       `toml:"server_cfg" json:"server_cfg"`
	AuthCfg         AuthConf         `toml:"auth_cfg" json:"auth_cfg"`
	DeliveryCfg     DeliveryConf     `toml:"delivery_cfg" json:"delivery_cfg"`
	LogsCfg         LogsConf         `toml:"logs_cfg" json:"logs_cfg"`
	DestinationsCfg DestinationsConf `toml:"destinations_cfg" json:"destinations_cfg"`
}

type ServerConf struct {
	Host string `toml:"host" json:"host"`
	Port int    `toml:"port" json:"port"`
}

type AuthConf struct {
	Enabled  bool   `toml:"enabled" json:"enabled"`
	Password string `toml:"password" json:"password"`
}

type DeliveryConf struct {
	AckMode          string `toml:"ack_mode" json:"ack_mode"`
	QueueCapacity    int    `toml:"queue_capacity" json:"queue_capacity"`
	DumpAfterSeconds int    `toml:"dump_after_seconds" json:"dump_after_seconds"`
	DumpDirectory    string `toml:"dump_directory" json:"dump_directory"`
}

type LogsConf struct {
	Output    string       `toml:"output" json:"output"`
	Directory string       `toml:"directory" json:"directory"`
	Rotation  RotationConf `toml:"rotation" json:"rotation"`
}

type RotationConf struct {
	MaxFileSizeBytes  int64 `toml:"max_file_size_bytes" json:"max_file_size_bytes"`
	MaxBackups        int   `toml:"max_backups" json:"max_backups"`
	MaxAgeDays        int   `toml:"max_age_days" json:"max_age_days"`
	MaxTotalSizeBytes int64 `toml:"max_total_size_bytes" json:"max_total_size_bytes"`
}

type DestinationsConf struct {
	Stdout bool                  `toml:"stdout" json:"stdout"`
	File   FileDestinationConf   `toml:"file" json:"file"`
	EGTS   []EGTSDestinationConf `toml:"egts" json:"egts"`
}

type EGTSDestinationConf struct {
	ID                    string       `toml:"id" json:"id"`
	Enabled               bool         `toml:"enabled" json:"enabled"`
	Host                  string       `toml:"host" json:"host"`
	Port                  int          `toml:"port" json:"port"`
	ConnectTimeoutSeconds int          `toml:"connect_timeout_seconds" json:"connect_timeout_seconds"`
	AckTimeoutSeconds     int          `toml:"ack_timeout_seconds" json:"ack_timeout_seconds"`
	Auth                  EGTSAuthConf `toml:"auth" json:"auth"`
}

type EGTSAuthConf struct {
	Enabled  bool   `toml:"enabled" json:"enabled"`
	UserName string `toml:"username" json:"username"`
	Password string `toml:"password" json:"password"`
}

type FileDestinationConf struct {
	Enabled   bool         `toml:"enabled" json:"enabled"`
	Directory string       `toml:"directory" json:"directory"`
	Rotation  RotationConf `toml:"rotation" json:"rotation"`
}

func DefaultConfiguration() *Configuration {
	return &Configuration{
		ServerCfg: ServerConf{
			Host: "0.0.0.0",
			Port: 8081,
		},
		AuthCfg: AuthConf{
			Enabled:  false,
			Password: "",
		},
		DeliveryCfg: DeliveryConf{
			AckMode:          "queued",
			QueueCapacity:    1024,
			DumpAfterSeconds: 60,
			DumpDirectory:    "./data/queue",
		},
		LogsCfg: LogsConf{
			Output:    "stderr",
			Directory: "./data/logs",
			Rotation: RotationConf{
				MaxFileSizeBytes:  10485760,
				MaxBackups:        5,
				MaxAgeDays:        7,
				MaxTotalSizeBytes: 62914560,
			},
		},
		DestinationsCfg: DestinationsConf{
			Stdout: true,
			File: FileDestinationConf{
				Enabled:   false,
				Directory: "./data/packets",
				Rotation: RotationConf{
					MaxFileSizeBytes:  10485760,
					MaxBackups:        5,
					MaxAgeDays:        7,
					MaxTotalSizeBytes: 62914560,
				},
			},
		},
	}
}

func PrepareConfiguration() (*Configuration, error) {
	confName := flag.String("conf", "", "TOML configuration file path; otherwise use ENV and .env")
	flag.Parse()
	if flag.NArg() != 0 {
		return nil, fmt.Errorf("Unexpected positional arguments; use -conf for a TOML file")
	}
	if *confName != "" {
		return PrepareFileConfiguration(*confName)
	}

	_, err := os.Stat(".env")
	if err != nil && !errors.Is(err, os.ErrNotExist) {
		return nil, fmt.Errorf("Can't access .env: %w", err)
	}
	if err == nil {
		err = godotenv.Overload(".env")
		if err != nil {
			return nil, fmt.Errorf("Can't load .env")
		}
	}
	return PrepareEnvConfiguration()
}

type egtsFileConfiguration struct {
	DestinationsCfg map[string]toml.Primitive `toml:"destinations_cfg"`
}

func PrepareFileConfiguration(fname string) (*Configuration, error) {
	data, err := os.ReadFile(fname)
	if err != nil {
		return nil, fmt.Errorf("Can't read configuration: %w", err)
	}
	cfg := DefaultConfiguration()
	metadata, err := toml.Decode(string(data), cfg)
	if err != nil {
		var parseErr toml.ParseError
		isParseError := errors.As(err, &parseErr)
		if isParseError {
			return nil, fmt.Errorf("Can't decode TOML configuration at line %d", parseErr.Line)
		}
		return nil, fmt.Errorf("Can't decode TOML configuration")
	}
	unknown := metadata.Undecoded()
	if len(unknown) != 0 {
		return nil, fmt.Errorf("Unknown configuration key: %s", unknown[0].String())
	}
	if len(cfg.DestinationsCfg.EGTS) > 0 {
		var raw egtsFileConfiguration
		metadata, err = toml.Decode(string(data), &raw)
		if err != nil {
			return nil, fmt.Errorf("Can't decode EGTS destinations")
		}
		var entries []toml.Primitive
		err = metadata.PrimitiveDecode(raw.DestinationsCfg["egts"], &entries)
		if err != nil {
			return nil, fmt.Errorf("Can't decode EGTS destination list")
		}
		for i, entry := range entries {
			relay := DefaultEGTSDestination()
			err = metadata.PrimitiveDecode(entry, &relay)
			if err != nil {
				return nil, fmt.Errorf("Can't decode EGTS destination %d", i)
			}
			cfg.DestinationsCfg.EGTS[i] = relay
		}
	}

	err = cfg.Validate()
	if err != nil {
		return nil, err
	}
	return cfg, nil
}

func PrepareEnvConfiguration() (*Configuration, error) {
	cfg := DefaultConfiguration()

	host, exists := os.LookupEnv("EGTS_SERVER_HOST")
	if exists {
		cfg.ServerCfg.Host = host
	}
	portStr, exists := os.LookupEnv("EGTS_SERVER_PORT")
	if exists {
		port, err := strconv.Atoi(portStr)
		if err != nil {
			return nil, fmt.Errorf("EGTS_SERVER_PORT: %w", err)
		}
		cfg.ServerCfg.Port = port
	}
	authEnabledStr, exists := os.LookupEnv("EGTS_AUTH_ENABLED")
	if exists {
		authEnabled, err := strconv.ParseBool(authEnabledStr)
		if err != nil {
			return nil, fmt.Errorf("EGTS_AUTH_ENABLED: %w", err)
		}
		cfg.AuthCfg.Enabled = authEnabled
	}
	password, exists := os.LookupEnv("EGTS_AUTH_PASSWORD")
	if exists {
		cfg.AuthCfg.Password = password
	}
	ackMode, exists := os.LookupEnv("EGTS_ACK_MODE")
	if exists {
		cfg.DeliveryCfg.AckMode = ackMode
	}
	queueCapacityStr, exists := os.LookupEnv("EGTS_QUEUE_CAPACITY")
	if exists {
		queueCapacity, err := strconv.Atoi(queueCapacityStr)
		if err != nil {
			return nil, fmt.Errorf("EGTS_QUEUE_CAPACITY: %w", err)
		}
		cfg.DeliveryCfg.QueueCapacity = queueCapacity
	}
	dumpAfterStr, exists := os.LookupEnv("EGTS_DUMP_AFTER_SECONDS")
	if exists {
		dumpAfter, err := strconv.Atoi(dumpAfterStr)
		if err != nil {
			return nil, fmt.Errorf("EGTS_DUMP_AFTER_SECONDS: %w", err)
		}
		cfg.DeliveryCfg.DumpAfterSeconds = dumpAfter
	}
	dumpDirectory, exists := os.LookupEnv("EGTS_DUMP_DIRECTORY")
	if exists {
		cfg.DeliveryCfg.DumpDirectory = dumpDirectory
	}
	logOutput, exists := os.LookupEnv("EGTS_LOG_OUTPUT")
	if exists {
		cfg.LogsCfg.Output = logOutput
	}
	logDirectory, exists := os.LookupEnv("EGTS_LOG_DIRECTORY")
	if exists {
		cfg.LogsCfg.Directory = logDirectory
	}
	maxFileSizeStr, exists := os.LookupEnv("EGTS_LOG_MAX_FILE_SIZE_BYTES")
	if exists {
		maxFileSize, err := strconv.ParseInt(maxFileSizeStr, 10, 64)
		if err != nil {
			return nil, fmt.Errorf("EGTS_LOG_MAX_FILE_SIZE_BYTES: %w", err)
		}
		cfg.LogsCfg.Rotation.MaxFileSizeBytes = maxFileSize
	}
	maxBackupsStr, exists := os.LookupEnv("EGTS_LOG_MAX_BACKUPS")
	if exists {
		maxBackups, err := strconv.Atoi(maxBackupsStr)
		if err != nil {
			return nil, fmt.Errorf("EGTS_LOG_MAX_BACKUPS: %w", err)
		}
		cfg.LogsCfg.Rotation.MaxBackups = maxBackups
	}
	maxAgeStr, exists := os.LookupEnv("EGTS_LOG_MAX_AGE_DAYS")
	if exists {
		maxAge, err := strconv.Atoi(maxAgeStr)
		if err != nil {
			return nil, fmt.Errorf("EGTS_LOG_MAX_AGE_DAYS: %w", err)
		}
		cfg.LogsCfg.Rotation.MaxAgeDays = maxAge
	}
	maxTotalSizeStr, exists := os.LookupEnv("EGTS_LOG_MAX_TOTAL_SIZE_BYTES")
	if exists {
		maxTotalSize, err := strconv.ParseInt(maxTotalSizeStr, 10, 64)
		if err != nil {
			return nil, fmt.Errorf("EGTS_LOG_MAX_TOTAL_SIZE_BYTES: %w", err)
		}
		cfg.LogsCfg.Rotation.MaxTotalSizeBytes = maxTotalSize
	}
	packetsStdoutStr, exists := os.LookupEnv("EGTS_PACKETS_STDOUT")
	if exists {
		packetsStdout, err := strconv.ParseBool(packetsStdoutStr)
		if err != nil {
			return nil, fmt.Errorf("EGTS_PACKETS_STDOUT: %w", err)
		}
		cfg.DestinationsCfg.Stdout = packetsStdout
	}
	packetsFileEnabledStr, exists := os.LookupEnv("EGTS_PACKETS_FILE_ENABLED")
	if exists {
		packetsFileEnabled, err := strconv.ParseBool(packetsFileEnabledStr)
		if err != nil {
			return nil, fmt.Errorf("EGTS_PACKETS_FILE_ENABLED: %w", err)
		}
		cfg.DestinationsCfg.File.Enabled = packetsFileEnabled
	}
	packetsDirectory, exists := os.LookupEnv("EGTS_PACKETS_FILE_DIRECTORY")
	if exists {
		cfg.DestinationsCfg.File.Directory = packetsDirectory
	}
	packetsMaxFileSizeStr, exists := os.LookupEnv("EGTS_PACKETS_FILE_MAX_FILE_SIZE_BYTES")
	if exists {
		maxFileSize, err := strconv.ParseInt(packetsMaxFileSizeStr, 10, 64)
		if err != nil {
			return nil, fmt.Errorf("EGTS_PACKETS_FILE_MAX_FILE_SIZE_BYTES: %w", err)
		}
		cfg.DestinationsCfg.File.Rotation.MaxFileSizeBytes = maxFileSize
	}
	packetsMaxBackupsStr, exists := os.LookupEnv("EGTS_PACKETS_FILE_MAX_BACKUPS")
	if exists {
		maxBackups, err := strconv.Atoi(packetsMaxBackupsStr)
		if err != nil {
			return nil, fmt.Errorf("EGTS_PACKETS_FILE_MAX_BACKUPS: %w", err)
		}
		cfg.DestinationsCfg.File.Rotation.MaxBackups = maxBackups
	}
	packetsMaxAgeStr, exists := os.LookupEnv("EGTS_PACKETS_FILE_MAX_AGE_DAYS")
	if exists {
		maxAge, err := strconv.Atoi(packetsMaxAgeStr)
		if err != nil {
			return nil, fmt.Errorf("EGTS_PACKETS_FILE_MAX_AGE_DAYS: %w", err)
		}
		cfg.DestinationsCfg.File.Rotation.MaxAgeDays = maxAge
	}
	packetsMaxTotalSizeStr, exists := os.LookupEnv("EGTS_PACKETS_FILE_MAX_TOTAL_SIZE_BYTES")
	if exists {
		maxTotalSize, err := strconv.ParseInt(packetsMaxTotalSizeStr, 10, 64)
		if err != nil {
			return nil, fmt.Errorf("EGTS_PACKETS_FILE_MAX_TOTAL_SIZE_BYTES: %w", err)
		}
		cfg.DestinationsCfg.File.Rotation.MaxTotalSizeBytes = maxTotalSize
	}

	relayIDs, exists := os.LookupEnv("EGTS_RELAY_IDS")
	knownRelayKeys := map[string]bool{"EGTS_RELAY_IDS": true}
	if exists && relayIDs != "" {
		for _, id := range strings.Split(relayIDs, ",") {
			relay := DefaultEGTSDestination()
			relay.ID = strings.TrimSpace(id)
			suffix := "_" + strings.ToUpper(relay.ID)
			key := "EGTS_RELAY_ENABLED" + suffix
			knownRelayKeys[key] = true
			value, found := os.LookupEnv(key)
			if found {
				parsed, err := strconv.ParseBool(value)
				if err != nil {
					return nil, fmt.Errorf("%s: %w", key, err)
				}
				relay.Enabled = parsed
			}
			key = "EGTS_RELAY_HOST" + suffix
			knownRelayKeys[key] = true
			value, found = os.LookupEnv(key)
			if found {
				relay.Host = value
			}
			key = "EGTS_RELAY_PORT" + suffix
			knownRelayKeys[key] = true
			value, found = os.LookupEnv(key)
			if found {
				parsed, err := strconv.Atoi(value)
				if err != nil {
					return nil, fmt.Errorf("%s: %w", key, err)
				}
				relay.Port = parsed
			}
			key = "EGTS_RELAY_CONNECT_TIMEOUT_SECONDS" + suffix
			knownRelayKeys[key] = true
			value, found = os.LookupEnv(key)
			if found {
				parsed, err := strconv.Atoi(value)
				if err != nil {
					return nil, fmt.Errorf("%s: %w", key, err)
				}
				relay.ConnectTimeoutSeconds = parsed
			}
			key = "EGTS_RELAY_ACK_TIMEOUT_SECONDS" + suffix
			knownRelayKeys[key] = true
			value, found = os.LookupEnv(key)
			if found {
				parsed, err := strconv.Atoi(value)
				if err != nil {
					return nil, fmt.Errorf("%s: %w", key, err)
				}
				relay.AckTimeoutSeconds = parsed
			}
			key = "EGTS_RELAY_AUTH_ENABLED" + suffix
			knownRelayKeys[key] = true
			value, found = os.LookupEnv(key)
			if found {
				parsed, err := strconv.ParseBool(value)
				if err != nil {
					return nil, fmt.Errorf("%s: %w", key, err)
				}
				relay.Auth.Enabled = parsed
			}
			key = "EGTS_RELAY_AUTH_USERNAME" + suffix
			knownRelayKeys[key] = true
			value, found = os.LookupEnv(key)
			if found {
				relay.Auth.UserName = value
			}
			key = "EGTS_RELAY_AUTH_PASSWORD" + suffix
			knownRelayKeys[key] = true
			value, found = os.LookupEnv(key)
			if found {
				relay.Auth.Password = value
			}
			cfg.DestinationsCfg.EGTS = append(cfg.DestinationsCfg.EGTS, relay)
		}
	}
	for _, variable := range os.Environ() {
		key, _, _ := strings.Cut(variable, "=")
		if strings.HasPrefix(key, "EGTS_RELAY_") && !knownRelayKeys[key] {
			return nil, fmt.Errorf("Unknown relay variable %s; use EGTS_RELAY_IDS and a destination ID suffix", key)
		}
	}

	err := cfg.Validate()
	if err != nil {
		return nil, err
	}
	return cfg, nil
}

func (cfg *Configuration) Validate() error {
	if strings.TrimSpace(cfg.ServerCfg.Host) == "" {
		return fmt.Errorf("server_cfg.host must not be empty")
	}
	if cfg.ServerCfg.Port < 1 || cfg.ServerCfg.Port > 65535 {
		return fmt.Errorf("server_cfg.port must be between 1 and 65535")
	}
	if cfg.AuthCfg.Enabled && cfg.AuthCfg.Password == "" {
		return fmt.Errorf("auth_cfg.password is required when authentication is enabled")
	}
	if cfg.AuthCfg.Enabled && (len(cfg.AuthCfg.Password) > 32 || strings.ContainsRune(cfg.AuthCfg.Password, 0)) {
		return fmt.Errorf("auth_cfg.password must not exceed 32 bytes or contain a null byte")
	}
	if cfg.DeliveryCfg.AckMode != "queued" && cfg.DeliveryCfg.AckMode != "delivered" {
		return fmt.Errorf("delivery_cfg.ack_mode must be queued or delivered")
	}
	if cfg.DeliveryCfg.QueueCapacity < 1 {
		return fmt.Errorf("delivery_cfg.queue_capacity must be positive")
	}
	if cfg.DeliveryCfg.DumpAfterSeconds < 1 || int64(cfg.DeliveryCfg.DumpAfterSeconds) > 9223372036 {
		return fmt.Errorf("delivery_cfg.dump_after_seconds must be between 1 and 9223372036")
	}
	if strings.TrimSpace(cfg.DeliveryCfg.DumpDirectory) == "" {
		return fmt.Errorf("delivery_cfg.dump_directory must not be empty")
	}
	if cfg.LogsCfg.Output != "stdout" && cfg.LogsCfg.Output != "stderr" && cfg.LogsCfg.Output != "file" {
		return fmt.Errorf("logs_cfg.output must be stdout, stderr or file")
	}
	if cfg.LogsCfg.Output == "file" && strings.TrimSpace(cfg.LogsCfg.Directory) == "" {
		return fmt.Errorf("logs_cfg.directory must not be empty for file output")
	}
	if cfg.LogsCfg.Output == "file" {
		err := cfg.LogsCfg.Rotation.Validate()
		if err != nil {
			return fmt.Errorf("logs_cfg.rotation: %w", err)
		}
	}
	if cfg.DestinationsCfg.File.Enabled && strings.TrimSpace(cfg.DestinationsCfg.File.Directory) == "" {
		return fmt.Errorf("destinations_cfg.file.directory must not be empty for file output")
	}
	if cfg.DestinationsCfg.File.Enabled {
		err := cfg.DestinationsCfg.File.Rotation.Validate()
		if err != nil {
			return fmt.Errorf("destinations_cfg.file.rotation: %w", err)
		}
	}
	ids := make(map[string]bool)
	hasRelay := false
	for _, relay := range cfg.DestinationsCfg.EGTS {
		err := relay.Validate()
		if err != nil {
			return fmt.Errorf("EGTS destination %q: %w", relay.ID, err)
		}
		if ids[relay.ID] {
			return fmt.Errorf("Duplicate EGTS destination ID %q", relay.ID)
		}
		ids[relay.ID] = true
		if relay.Enabled {
			hasRelay = true
		}
	}
	if !cfg.DestinationsCfg.Stdout && !cfg.DestinationsCfg.File.Enabled && !hasRelay {
		return fmt.Errorf("At least one packet destination must be enabled")
	}
	return cfg.ValidateOutputs(os.Stdout, os.Stderr)
}
