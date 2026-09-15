package configuration

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"time"

	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
)

func PrepareLogger(cfg *Configuration) (*os.File, error) {
	err := cfg.ValidateOutputs(os.Stdout, os.Stderr)
	if err != nil {
		return nil, err
	}
	var writer io.Writer
	var logFile *os.File
	switch cfg.LogsCfg.Output {
	case "stdout":
		writer = os.Stdout
	case "stderr":
		writer = os.Stderr
	case "file":
		directory, err := resolveDirectory(cfg.LogsCfg.Directory)
		if err != nil {
			return nil, fmt.Errorf("Can't resolve application log directory: %w", err)
		}
		err = os.MkdirAll(directory, 0750)
		if err != nil {
			return nil, fmt.Errorf("Can't create application log directory: %w", err)
		}
		logFile, err = os.OpenFile(filepath.Join(directory, LOG_FILENAME), os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0640)
		if err != nil {
			return nil, fmt.Errorf("Can't open application log: %w", err)
		}
		err = cfg.ValidateOutputs(os.Stdout, os.Stderr)
		if err != nil {
			closeErr := logFile.Close()
			if closeErr != nil {
				return nil, fmt.Errorf("%w; Can't close application log: %v", err, closeErr)
			}
			return nil, err
		}
		writer = logFile
	default:
		return nil, fmt.Errorf("logs_cfg.output must be stdout, stderr or file")
	}
	zerolog.TimeFieldFormat = time.RFC3339
	log.Logger = zerolog.New(writer).With().Timestamp().Str("application", "egts_gateway").Logger()
	return logFile, nil
}
