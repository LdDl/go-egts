package destination

import (
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"sync"
	"time"

	"github.com/LdDl/go-egts/gateway/configuration"
)

type Writer struct {
	mu       sync.Mutex
	cfg      configuration.Configuration
	name     string
	output   io.Writer
	stdout   *os.File
	stream   *os.File
	deadline bool
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
	stream, err := openStdout()
	if err != nil {
		return nil, fmt.Errorf("Can't open packet stdout: %w", err)
	}
	err = stream.SetWriteDeadline(time.Time{})
	deadline := err == nil
	if err != nil && !errors.Is(err, os.ErrNoDeadline) {
		if stream != os.Stdout {
			closeErr := stream.Close()
			if closeErr != nil {
				return nil, fmt.Errorf("%w; Can't close stdout: %v", err, closeErr)
			}
		}
		return nil, err
	}
	return &Writer{cfg: *cfg, name: "stdout", output: stream, stdout: os.Stdout, stream: stream, deadline: deadline}, nil
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
	return w.WriteContext(context.Background(), record)
}

func (w *Writer) WriteContext(ctx context.Context, record *Record) error {
	w.mu.Lock()
	defer w.mu.Unlock()
	if w.closed {
		return os.ErrClosed
	}
	if w.writeErr != nil {
		return w.writeErr
	}
	if ctx.Err() != nil {
		return ctx.Err()
	}
	data, err := record.Encode()
	if err != nil {
		return err
	}
	err = w.cfg.ValidateOutputs(w.stdout, os.Stderr)
	if err != nil {
		return err
	}
	if w.deadline {
		deadline, _ := ctx.Deadline()
		err = w.stream.SetWriteDeadline(deadline)
		if err != nil {
			return err
		}
	}
	var finished chan struct{}
	var interrupted chan error
	if w.deadline && ctx.Done() != nil {
		finished = make(chan struct{})
		interrupted = make(chan error, 1)
		go func() {
			select {
			case <-ctx.Done():
				interrupted <- w.stream.SetWriteDeadline(time.Now())
			case <-finished:
				interrupted <- nil
			}
		}()
	}
	n, err := w.output.Write(data)
	if finished != nil {
		close(finished)
		interruptErr := <-interrupted
		if err == nil && interruptErr != nil {
			err = interruptErr
		}
		if err != nil && ctx.Err() != nil {
			err = ctx.Err()
		}
	}
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
	if w.stream != nil && w.stream != w.stdout {
		err := w.stream.Close()
		if err != nil {
			w.writeErr = fmt.Errorf("Can't close packet stdout: %w", err)
		}
	}
	return w.writeErr
}
