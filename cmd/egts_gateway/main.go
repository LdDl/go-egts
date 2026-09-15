package main

import (
	"errors"
	"os"

	"github.com/LdDl/go-egts/gateway/configuration"
	"github.com/LdDl/go-egts/gateway/logger"
	"github.com/rs/zerolog/log"
)

func main() {
	cfg, err := configuration.PrepareConfiguration()
	if err != nil {
		var conflict *configuration.OutputConflictError
		isConflict := errors.As(err, &conflict)
		if isConflict && conflict.StderrUnsafe {
			os.Exit(1)
		}
		log.Log().
			Str("scope", logger.SCOPE_CONFIG).
			Str("event", logger.EVENT_CONFIG_ERROR).
			Err(err).
			Msg("Can't prepare configuration")
		os.Exit(1)
	}
	logFile, err := configuration.PrepareLogger(cfg)
	if err != nil {
		var conflict *configuration.OutputConflictError
		isConflict := errors.As(err, &conflict)
		if isConflict && conflict.StderrUnsafe {
			os.Exit(1)
		}
		log.Log().
			Str("scope", logger.SCOPE_CONFIG).
			Str("event", logger.EVENT_LOGGER_ERROR).
			Err(err).
			Msg("Can't prepare logger")
		os.Exit(1)
	}

	log.Log().
		Str("scope", logger.SCOPE_STARTUP).
		Str("event", logger.EVENT_STARTUP).
		Str("host", cfg.ServerCfg.Host).
		Int("port", cfg.ServerCfg.Port).
		Bool("auth_enabled", cfg.AuthCfg.Enabled).
		Str("ack_mode", cfg.DeliveryCfg.AckMode).
		Int("queue_capacity", cfg.DeliveryCfg.QueueCapacity).
		Int("dump_after_seconds", cfg.DeliveryCfg.DumpAfterSeconds).
		Str("dump_directory", cfg.DeliveryCfg.DumpDirectory).
		Msg("Starting egts_gateway")
	log.Log().
		Str("scope", logger.SCOPE_SHUTDOWN).
		Str("event", logger.EVENT_SHUTDOWN).
		Msg("Configuration checked; shutting down without starting the server")
	if logFile != nil {
		err = logFile.Close()
		if err != nil {
			var conflict *configuration.OutputConflictError
			isConflict := errors.As(err, &conflict)
			if isConflict && conflict.StderrUnsafe {
				os.Exit(1)
			}
			log.Logger = log.Output(os.Stderr)
			log.Log().
				Str("scope", logger.SCOPE_SHUTDOWN).
				Str("event", logger.EVENT_LOGGER_ERROR).
				Err(err).
				Msg("Can't close application log")
			os.Exit(1)
		}
	}
}
