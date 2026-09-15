package main

import (
	"context"
	"errors"
	"os"
	"os/signal"
	"syscall"

	"github.com/LdDl/go-egts/gateway/configuration"
	"github.com/LdDl/go-egts/gateway/logger"
	"github.com/LdDl/go-egts/gateway/server"
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

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()
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
	exitCode := 0
	err = server.Run(ctx, cfg)
	if err != nil {
		exitCode = 1
		var conflict *configuration.OutputConflictError
		isConflict := errors.As(err, &conflict)
		if isConflict && conflict.StderrUnsafe {
			os.Exit(1)
		}
		log.Log().Str("scope", logger.SCOPE_SERVER).Str("event", logger.EVENT_SERVER_ERROR).
			Err(err).Msg("Gateway stopped with an error")
	}
	log.Log().
		Str("scope", logger.SCOPE_SHUTDOWN).
		Str("event", logger.EVENT_SHUTDOWN).
		Msg("Gateway stopped")
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
	if exitCode != 0 {
		os.Exit(exitCode)
	}
}
