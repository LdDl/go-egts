package destination

import (
	"fmt"
	"io"
	"os"
	"sync"

	"github.com/LdDl/go-egts/gateway/configuration"
)

type Writer struct {
	mu       sync.Mutex
	cfg      configuration.Configuration
	name     string
	output   io.Writer
	stdout   *os.File
	file     *configuration.RotatingFile
	closed   bool
	writeErr error
}

func PrepareStdout(cfg *configuration.Configuration) (*Writer, error) {
	if !cfg.DestinationsCfg.Stdout {
		return nil, fmt.Errorf("Packet stdout destination is disabled")
	}
	err := cfg.ValidateOutputs(os.Stdout, os.Stderr)
	if err != nil {
		return nil, err
	}
	return &Writer{cfg: *cfg, name: "stdout", output: os.Stdout, stdout: os.Stdout}, nil
}

func PrepareFile(cfg *configuration.Configuration) (*Writer, error) {
	file, err := configuration.PreparePacketFile(cfg)
	if err != nil {
		return nil, err
	}
	return &Writer{cfg: *cfg, name: configuration.PACKETS_FILENAME, output: file, stdout: os.Stdout, file: file}, nil
}

// Write delivers one record to this destination. Other destinations are written separately.
func (w *Writer) Write(record *Record) error {
	w.mu.Lock()
	defer w.mu.Unlock()
	if w.closed {
		return os.ErrClosed
	}
	if w.writeErr != nil {
		return w.writeErr
	}
	data, err := record.Encode()
	if err != nil {
		return err
	}
	err = w.cfg.ValidateOutputs(w.stdout, os.Stderr)
	if err != nil {
		return err
	}
	n, err := w.output.Write(data)
	if err == nil && n != len(data) {
		err = io.ErrShortWrite
	}
	if err != nil {
		err = fmt.Errorf("Can't write packet to %s: %w", w.name, err)
		// Stop after a partial write so later records cannot join an incomplete JSON line.
		if n > 0 {
			w.writeErr = err
		}
		return err
	}
	return nil
}

func (w *Writer) Close() error {
	w.mu.Lock()
	defer w.mu.Unlock()
	if w.closed {
		return w.writeErr
	}
	w.closed = true
	if w.file != nil {
		err := w.file.Close()
		if err != nil {
			w.writeErr = fmt.Errorf("Can't close packet file: %w", err)
		}
	}
	return w.writeErr
}
