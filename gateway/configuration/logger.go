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

func PrepareLogger(cfg *Configuration) (*LogFile, error) {
	err := cfg.ValidateOutputs(os.Stdout, os.Stderr)
	if err != nil {
		return nil, err
	}
	var writer io.Writer
	var logFile *LogFile
	switch cfg.LogsCfg.Output {
	case "stdout":
		writer = os.Stdout
	case "stderr":
		writer = os.Stderr
	case "file":
		err = cfg.LogsCfg.Rotation.Validate()
		if err != nil {
			return nil, err
		}
		directory, err := resolveDirectory(cfg.LogsCfg.Directory)
		if err != nil {
			return nil, fmt.Errorf("Can't resolve application log directory: %w", err)
		}
		err = os.MkdirAll(directory, 0750)
		if err != nil {
			return nil, fmt.Errorf("Can't create application log directory: %w", err)
		}
		logFile = &LogFile{cfg: *cfg}
		logFile.cfg.LogsCfg.Directory = directory
		err = logFile.open()
		if err == nil && logFile.size > cfg.LogsCfg.Rotation.MaxFileSizeBytes {
			err = logFile.rotate()
		}
		if err == nil {
			err = logFile.cleanup(0)
		}
		if err != nil {
			logFile.writeErr = err
			closeErr := logFile.Close()
			return nil, closeErr
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
