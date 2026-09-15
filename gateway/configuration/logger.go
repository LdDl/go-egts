package configuration

import (
	"bytes"
	"errors"
	"fmt"
	"io"
	"os"
	"time"

	"github.com/LdDl/go-egts/gateway/logger"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
)

func PrepareLogger(cfg *Configuration) (*RotatingFile, error) {
	err := cfg.ValidateOutputs(os.Stdout, os.Stderr)
	if err != nil {
		return nil, err
	}
	var writer io.Writer
	var logFile *RotatingFile
	switch cfg.LogsCfg.Output {
	case "stdout":
		writer = os.Stdout
	case "stderr":
		writer = os.Stderr
	case "file":
		logFile, err = prepareRotatingFile(cfg, cfg.LogsCfg.Directory, LOG_FILENAME, cfg.LogsCfg.Rotation)
		if err != nil {
			return nil, fmt.Errorf("Can't prepare application log: %w", err)
		}
		writer = logFile
	default:
		return nil, fmt.Errorf("logs_cfg.output must be stdout, stderr or file")
	}
	zerolog.TimeFieldFormat = time.RFC3339
	log.Logger = zerolog.New(writer).With().Timestamp().Str("application", "egts_gateway").Logger()
	zerolog.ErrorHandler = func(err error) {
		var conflict *OutputConflictError
		isConflict := errors.As(err, &conflict)
		if isConflict && conflict.StderrUnsafe {
			return
		}
		// Format in memory so a stderr write failure cannot recursively invoke this handler.
		var buffer bytes.Buffer
		errorLogger := zerolog.New(&buffer).With().Timestamp().Str("application", "egts_gateway").Logger()
		errorLogger.Log().
			Str("scope", logger.SCOPE_LOGGER).
			Str("event", logger.EVENT_LOGGER_ERROR).
			Err(err).
			Msg("Can't write application log")
		_, writeErr := os.Stderr.Write(buffer.Bytes())
		if writeErr != nil {
			return
		}
	}
	return logFile, nil
}
