package server

import (
	"context"
	"errors"
	"fmt"
	"net"
	"os"
	"path/filepath"
	"strconv"
	"sync"

	"github.com/LdDl/go-egts/gateway/configuration"
	"github.com/LdDl/go-egts/gateway/destination"
	"github.com/LdDl/go-egts/gateway/logger"
	"github.com/rs/zerolog/log"
)

type server struct {
	cfg         configuration.Configuration
	stdout      *destination.Writer
	file        *destination.Writer
	mu          sync.Mutex
	clients     map[net.Conn]struct{}
	connections sync.WaitGroup
	accepting   bool
	queue       []*pendingPacket
	wake        chan struct{}
	hasDump     bool
	relays      map[string]*relaySession
	rabbits     map[string]*destination.RabbitMQ
}

func Run(ctx context.Context, cfg *configuration.Configuration) error {
	err := cfg.Validate()
	if err != nil {
		return err
	}
	listener, err := net.Listen("tcp", net.JoinHostPort(cfg.ServerCfg.Host, strconv.Itoa(cfg.ServerCfg.Port)))
	if err != nil {
		return fmt.Errorf("Can't listen for EGTS: %w", err)
	}
	s, err := newServer(cfg)
	if err != nil {
		closeErr := listener.Close()
		if closeErr != nil {
			return fmt.Errorf("%w; Can't close listener: %v", err, closeErr)
		}
		return err
	}
	return s.serve(ctx, listener)
}

func newServer(cfg *configuration.Configuration) (*server, error) {
	s := &server{cfg: *cfg, clients: make(map[net.Conn]struct{}), wake: make(chan struct{}, 1), relays: make(map[string]*relaySession), rabbits: make(map[string]*destination.RabbitMQ)}
	s.cfg.DestinationsCfg.EGTS = append([]configuration.EGTSDestinationConf(nil), cfg.DestinationsCfg.EGTS...)
	s.cfg.DestinationsCfg.RabbitMQ = append([]configuration.RabbitMQDestinationConf(nil), cfg.DestinationsCfg.RabbitMQ...)
	err := cfg.Validate()
	if err != nil {
		return nil, err
	}
	err = os.MkdirAll(cfg.DeliveryCfg.DumpDirectory, 0750)
	if err != nil {
		return nil, fmt.Errorf("Can't create queue directory: %w", err)
	}
	directory, err := filepath.EvalSymlinks(cfg.DeliveryCfg.DumpDirectory)
	if err != nil {
		return nil, err
	}
	s.cfg.DeliveryCfg.DumpDirectory, err = filepath.Abs(directory)
	if err != nil {
		return nil, err
	}
	err = s.restoreDump()
	if err != nil {
		return nil, err
	}
	for _, rabbit := range s.cfg.DestinationsCfg.RabbitMQ {
		if !rabbit.Enabled {
			continue
		}
		writer, err := destination.PrepareRabbitMQ(rabbit)
		if err != nil {
			return nil, err
		}
		s.rabbits[rabbit.ID] = writer
	}
	if cfg.DestinationsCfg.File.Enabled {
		s.file, err = destination.PrepareFile(&s.cfg)
		if err != nil {
			return nil, err
		}
	}
	if cfg.DestinationsCfg.Stdout {
		s.stdout, err = destination.PrepareStdout(&s.cfg)
		if err != nil {
			if s.file != nil {
				closeErr := s.file.Close()
				if closeErr != nil {
					return nil, fmt.Errorf("%w; Can't close packet file: %v", err, closeErr)
				}
			}
			return nil, err
		}
	}
	return s, nil
}

func (s *server) serve(ctx context.Context, listener net.Listener) error {
	ctx, cancel := context.WithCancel(ctx)
	defer cancel()
	s.mu.Lock()
	s.accepting = true
	s.mu.Unlock()
	accepted := make(chan error, 1)
	delivered := make(chan error, 1)
	go func() {
		accepted <- s.accept(ctx, listener)
	}()
	go func() {
		delivered <- s.deliver(ctx)
	}()
	log.Log().Str("scope", logger.SCOPE_SERVER).Str("event", logger.EVENT_LISTENING).
		Str("address", listener.Addr().String()).Msg("Accepting EGTS connections")
	var result error
	acceptFinished := false
	deliveryFinished := false
	select {
	case <-ctx.Done():
	case result = <-accepted:
		acceptFinished = true
	case result = <-delivered:
		deliveryFinished = true
	}
	s.mu.Lock()
	s.accepting = false
	s.mu.Unlock()
	cancel()
	err := listener.Close()
	if err != nil && !errors.Is(err, net.ErrClosed) && result == nil {
		result = err
	}
	if !acceptFinished {
		err = <-accepted
		if err != nil && result == nil {
			result = err
		}
	}
	s.mu.Lock()
	for conn := range s.clients {
		err = conn.Close()
		if err != nil && !errors.Is(err, net.ErrClosed) && result == nil {
			result = err
		}
	}
	s.mu.Unlock()
	s.connections.Wait()
	if !deliveryFinished {
		err = <-delivered
		if err != nil && result == nil {
			result = err
		}
	}
	if len(s.queue) > 0 || s.hasDump {
		err = s.saveDump()
		if err != nil {
			if result == nil {
				result = err
			} else {
				result = fmt.Errorf("%w; Can't save pending packets: %v", result, err)
			}
		} else {
			log.Log().Str("scope", logger.SCOPE_DELIVERY).Str("event", logger.EVENT_DUMP_SAVED).
				Int("packets", len(s.queue)).Msg("Saved pending packets")
		}
	}
	if s.file != nil {
		err = s.file.Close()
		if err != nil {
			if result == nil {
				result = err
			} else {
				result = fmt.Errorf("%w; Can't close packet file: %v", result, err)
			}
		}
	}
	if s.stdout != nil {
		err = s.stdout.Close()
		if err != nil {
			if result == nil {
				result = err
			} else {
				result = fmt.Errorf("%w; Can't close packet stdout: %v", result, err)
			}
		}
	}
	for _, session := range s.relays {
		err = session.close()
		if err != nil {
			if result == nil {
				result = err
			} else {
				result = fmt.Errorf("%w; Can't close EGTS destination: %v", result, err)
			}
		}
	}
	for id, rabbit := range s.rabbits {
		err = rabbit.Close()
		if err != nil {
			if result == nil {
				result = err
			} else {
				result = fmt.Errorf("%w; Can't close RabbitMQ destination %s: %v", result, id, err)
			}
		}
	}

	return result
}

func (s *server) accept(ctx context.Context, listener net.Listener) error {
	for {
		conn, err := listener.Accept()
		if err != nil {
			if ctx.Err() != nil {
				return nil
			}
			return fmt.Errorf("Can't accept EGTS connection: %w", err)
		}
		s.mu.Lock()
		if !s.accepting {
			s.mu.Unlock()
			return conn.Close()
		}
		s.clients[conn] = struct{}{}
		s.connections.Add(1)
		s.mu.Unlock()
		go func() {
			defer s.connections.Done()
			err := s.handleConnection(ctx, conn)
			if err != nil && ctx.Err() == nil {
				log.Log().Str("scope", logger.SCOPE_SERVER).Str("event", logger.EVENT_CONNECTION_ERROR).
					Str("remote_address", conn.RemoteAddr().String()).Err(err).Msg("EGTS connection failed")
			}
			closeErr := conn.Close()
			if closeErr != nil && !errors.Is(closeErr, net.ErrClosed) && ctx.Err() == nil {
				log.Log().Str("scope", logger.SCOPE_SERVER).Str("event", logger.EVENT_CONNECTION_ERROR).
					Err(closeErr).Msg("Can't close EGTS connection")
			}
			s.mu.Lock()
			delete(s.clients, conn)
			s.mu.Unlock()
		}()
	}
}
