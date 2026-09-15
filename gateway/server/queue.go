package server

import (
	"context"
	"errors"
	"fmt"
	"sync"
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
	record   *destination.Record
	stdout   bool
	file     bool
	egts     map[string]bool
	rabbitmq map[string]bool
	relay    *relayPacket
	done     chan struct{}
}

type deliveryTarget struct {
	kind        string
	id          string
	busy        bool
	started     time.Time
	failedSince time.Time
	retryAt     time.Time
}

type deliveryResult struct {
	target int
	item   *pendingPacket
	err    error
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
	for _, cfg := range s.cfg.DestinationsCfg.EGTS {
		if !cfg.Enabled {
			continue
		}
		forward, err := shouldRelay(&record.Packet)
		if err != nil {
			return nil, err
		}
		if !forward {
			break
		}
		if relay == nil || relay.session == nil {
			return nil, fmt.Errorf("Missing EGTS relay session")
		}
		if item.egts == nil {
			item.egts = make(map[string]bool)
			item.relay = relay
		}
		item.egts[cfg.ID] = true
	}
	if item.relay != nil {
		item.relay.session.pending += len(item.egts)
	}
	for _, cfg := range s.cfg.DestinationsCfg.RabbitMQ {
		if cfg.Enabled {
			if item.rabbitmq == nil {
				item.rabbitmq = make(map[string]bool)
			}
			item.rabbitmq[cfg.ID] = true
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
	ctx, cancel := context.WithCancel(ctx)
	var workers sync.WaitGroup
	defer func() {
		cancel()
		workers.Wait()
	}()
	var targets []deliveryTarget
	if s.cfg.DestinationsCfg.Stdout {
		targets = append(targets, deliveryTarget{kind: "stdout"})
	}
	if s.cfg.DestinationsCfg.File.Enabled {
		targets = append(targets, deliveryTarget{kind: "file"})
	}
	for _, cfg := range s.cfg.DestinationsCfg.EGTS {
		if cfg.Enabled {
			targets = append(targets, deliveryTarget{kind: "egts", id: cfg.ID})
		}
	}
	for _, cfg := range s.cfg.DestinationsCfg.RabbitMQ {
		if cfg.Enabled {
			targets = append(targets, deliveryTarget{kind: "rabbitmq", id: cfg.ID})
		}
	}
	results := make(chan deliveryResult, len(targets))
	var resultErr error
	for {
		stopping := ctx.Err() != nil
		s.mu.Lock()
		pending := s.queue[:0]
		for _, item := range s.queue {
			if !item.stdout && !item.file && len(item.egts) == 0 && len(item.rabbitmq) == 0 {
				close(item.done)
			} else {
				pending = append(pending, item)
			}
		}
		for i := len(pending); i < len(s.queue); i++ {
			s.queue[i] = nil
		}
		s.queue = pending
		s.mu.Unlock()
		var retryAt time.Time
		active := 0
		for i := range targets {
			target := &targets[i]
			if target.busy {
				active++
				continue
			}
			if stopping {
				continue
			}
			if time.Now().Before(target.retryAt) {
				if retryAt.IsZero() || target.retryAt.Before(retryAt) {
					retryAt = target.retryAt
				}
				continue
			}
			s.mu.Lock()
			var item *pendingPacket
			for _, candidate := range s.queue {
				if (target.kind == "stdout" && candidate.stdout) || (target.kind == "file" && candidate.file) || (target.kind == "egts" && candidate.egts[target.id]) || (target.kind == "rabbitmq" && candidate.rabbitmq[target.id]) {
					item = candidate
					break
				}
			}
			s.mu.Unlock()
			if item == nil {
				continue
			}
			target.busy = true
			active++
			target.started = time.Now()
			remaining := time.Duration(s.cfg.DeliveryCfg.DumpAfterSeconds) * time.Second
			if !target.failedSince.IsZero() {
				remaining -= time.Since(target.failedSince)
			}
			if target.kind == "stdout" && remaining > time.Second {
				remaining = time.Second
			}
			workers.Add(1)
			go func(index int, target deliveryTarget, item *pendingPacket, timeout time.Duration) {
				defer workers.Done()
				attempt, stop := context.WithTimeout(ctx, timeout)
				var err error
				switch target.kind {
				case "stdout":
					err = s.stdout.WriteContext(attempt, item.record)
				case "file":
					err = s.file.WriteContext(attempt, item.record)
				case "egts":
					err = item.relay.session.writers[target.id].WriteContext(attempt, item.record, item.relay.identity)
				case "rabbitmq":
					err = s.rabbits[target.id].WriteContext(attempt, item.record)
				}
				stop()
				results <- deliveryResult{target: index, item: item, err: err}
			}(i, *target, item, remaining)
		}
		if stopping && active == 0 {
			return resultErr
		}
		ctxDone := ctx.Done()
		if stopping {
			ctxDone = nil
		}
		var timer *time.Timer
		var retry <-chan time.Time
		if !retryAt.IsZero() {
			timer = time.NewTimer(time.Until(retryAt))
			retry = timer.C
		}
		var result deliveryResult
		received := false
		select {
		case <-ctxDone:
		case <-s.wake:
		case <-retry:
		case result = <-results:
			received = true
		}
		if timer != nil {
			timer.Stop()
		}
		if !received {
			continue
		}
		target := &targets[result.target]
		target.busy = false
		if result.err != nil {
			if ctx.Err() != nil {
				continue
			}
			if target.failedSince.IsZero() {
				target.failedSince = target.started
				log.Log().Str("scope", logger.SCOPE_DELIVERY).Str("event", logger.EVENT_DELIVERY_ERROR).
					Str("destination", target.kind).Str("destination_id", target.id).
					Err(result.err).Msg("Packet delivery failed; retrying")
			}
			deadline := target.failedSince.Add(time.Duration(s.cfg.DeliveryCfg.DumpAfterSeconds) * time.Second)
			if !time.Now().Before(deadline) {
				resultErr = fmt.Errorf("Packet delivery stalled for %s %s: %w", target.kind, target.id, result.err)
				cancel()
				continue
			}
			target.retryAt = time.Now().Add(time.Second)
			if deadline.Before(target.retryAt) {
				target.retryAt = deadline
			}
			continue
		}
		target.failedSince = time.Time{}
		target.retryAt = time.Time{}
		s.mu.Lock()
		var finished *relaySession
		switch target.kind {
		case "stdout":
			result.item.stdout = false
		case "file":
			result.item.file = false
		case "rabbitmq":
			delete(result.item.rabbitmq, target.id)
		case "egts":
			delete(result.item.egts, target.id)
			session := result.item.relay.session
			session.pending--
			if session.pending == 0 && session.closed {
				delete(s.relays, session.id)
				finished = session
			}
		}
		s.mu.Unlock()
		if s.hasDump {
			err := s.saveDump()
			if err != nil {
				if resultErr == nil {
					resultErr = err
				}
				cancel()
			}
		}
		if finished != nil {
			err := finished.close()
			if err != nil {
				if resultErr == nil {
					resultErr = err
				}
				cancel()
			}
		}
	}
}
