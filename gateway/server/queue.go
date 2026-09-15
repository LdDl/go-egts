package server

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/LdDl/go-egts/gateway/destination"
	"github.com/LdDl/go-egts/gateway/logger"
	"github.com/rs/zerolog/log"
)

var (
	ErrQueueFull = errors.New("Packet queue is full")
	ErrStopped   = errors.New("Gateway is stopping")
)

type pendingPacket struct {
	record *destination.Record
	stdout bool
	file   bool
	egts   bool
	relay  *relayPacket
	done   chan struct{}
}

func (s *server) enqueue(record *destination.Record, relay *relayPacket) (*pendingPacket, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if !s.accepting {
		return nil, ErrStopped
	}
	if len(s.queue) >= s.cfg.DeliveryCfg.QueueCapacity {
		return nil, ErrQueueFull
	}
	item := &pendingPacket{record: record, stdout: s.cfg.DestinationsCfg.Stdout, file: s.cfg.DestinationsCfg.File.Enabled, done: make(chan struct{})}
	if s.cfg.DestinationsCfg.EGTS.Enabled {
		forward, err := shouldRelay(&record.Packet)
		if err != nil {
			return nil, err
		}
		if forward {
			if relay == nil || relay.session == nil {
				return nil, fmt.Errorf("Missing EGTS relay session")
			}
			item.egts = true
			item.relay = relay
			relay.session.pending++
		}
	}
	s.queue = append(s.queue, item)
	select {
	case s.wake <- struct{}{}:
	default:
	}
	return item, nil
}

func (s *server) deliver(ctx context.Context) error {
	var failedSince time.Time
	for {
		if ctx.Err() != nil {
			return nil
		}
		s.mu.Lock()
		var item *pendingPacket
		if len(s.queue) > 0 {
			item = s.queue[0]
		}
		s.mu.Unlock()
		if item == nil {
			select {
			case <-ctx.Done():
				return nil
			case <-s.wake:
				continue
			}
		}
		var deliveryErr error
		attemptStarted := time.Now()
		changed := false
		if item.stdout {
			attempt, cancel := context.WithTimeout(ctx, time.Second)
			err := s.stdout.WriteContext(attempt, item.record)
			cancel()
			if err != nil {
				deliveryErr = err
			} else {
				item.stdout = false
				changed = true
			}
		}
		if item.file && ctx.Err() == nil {
			err := s.file.WriteContext(ctx, item.record)
			if err != nil {
				if deliveryErr == nil {
					deliveryErr = err
				}
			} else {
				item.file = false
				changed = true
			}
		}
		if item.egts && ctx.Err() == nil {
			remaining := time.Duration(s.cfg.DeliveryCfg.DumpAfterSeconds) * time.Second
			if !failedSince.IsZero() {
				remaining -= time.Since(failedSince)
			}
			attempt, cancel := context.WithTimeout(ctx, remaining)
			err := item.relay.session.writer.WriteContext(attempt, item.record, item.relay.identity)
			cancel()
			if err != nil {
				if deliveryErr == nil {
					deliveryErr = err
				}
			} else {
				item.egts = false
				changed = true
				s.mu.Lock()
				session := item.relay.session
				session.pending--
				finished := session.pending == 0 && session.closed
				if finished {
					delete(s.relays, session.id)
				}
				s.mu.Unlock()
				if finished {
					err = session.writer.Close()
					if err != nil {
						return err
					}
				}
			}
		}
		if s.hasDump && changed {
			err := s.saveDump()
			if err != nil {
				return err
			}
		}
		if !item.stdout && !item.file && !item.egts {
			s.mu.Lock()
			s.queue[0] = nil
			s.queue = s.queue[1:]
			close(item.done)
			s.mu.Unlock()
			failedSince = time.Time{}
			continue
		}
		if ctx.Err() != nil {
			return nil
		}
		if failedSince.IsZero() {
			failedSince = attemptStarted
			log.Log().Str("scope", logger.SCOPE_DELIVERY).Str("event", logger.EVENT_DELIVERY_ERROR).
				Err(deliveryErr).Msg("Packet delivery failed; retrying")
		}
		if time.Since(failedSince) >= time.Duration(s.cfg.DeliveryCfg.DumpAfterSeconds)*time.Second {
			return fmt.Errorf("Packet delivery stalled: %w", deliveryErr)
		}
		timer := time.NewTimer(time.Second)
		select {
		case <-ctx.Done():
			timer.Stop()
			return nil
		case <-timer.C:
		}
	}
}
