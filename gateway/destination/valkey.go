package destination

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"net"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/LdDl/go-egts/gateway/configuration"
	"github.com/gomodule/redigo/redis"
)

type Valkey struct {
	mu         sync.Mutex
	cfg        configuration.ValkeyDestinationConf
	connection redis.Conn
	closed     bool
}

func PrepareValkey(cfg configuration.ValkeyDestinationConf) (*Valkey, error) {
	if !cfg.Enabled {
		return nil, fmt.Errorf("Valkey destination is disabled")
	}
	err := cfg.Validate()
	if err != nil {
		return nil, err
	}
	return &Valkey{cfg: cfg}, nil
}

func (v *Valkey) WriteContext(ctx context.Context, record *Record) (err error) {
	v.mu.Lock()
	defer v.mu.Unlock()
	if v.closed {
		return net.ErrClosed
	}
	if ctx.Err() != nil {
		return ctx.Err()
	}
	body, err := record.Encode()
	if err != nil {
		return err
	}
	defer func() {
		if err != nil && v.connection != nil {
			// Redigo closes the socket itself on transport errors and cancellation.
			if v.connection.Err() == nil {
				closeErr := v.connection.Close()
				if closeErr != nil {
					err = fmt.Errorf("%w; Can't close Valkey connection: %v", err, closeErr)
				}
			}
			v.connection = nil
		}
	}()
	if v.connection == nil || v.connection.Err() != nil {
		connect, cancel := context.WithTimeout(ctx, time.Duration(v.cfg.ConnectTimeoutSeconds)*time.Second)
		v.connection, err = redis.DialContext(connect, "tcp", net.JoinHostPort(v.cfg.Host, strconv.Itoa(v.cfg.Port)),
			redis.DialUsername(v.cfg.UserName), redis.DialPassword(v.cfg.Password), redis.DialDatabase(v.cfg.Database),
			redis.DialWriteTimeout(time.Duration(v.cfg.WriteTimeoutSeconds)*time.Second))
		if err != nil && connect.Err() != nil {
			err = connect.Err()
		}
		cancel()
		if err != nil {
			return fmt.Errorf("Can't prepare Valkey connection: %w", err)
		}
	}
	ctx, cancel := context.WithTimeout(ctx, time.Duration(v.cfg.WriteTimeoutSeconds)*time.Second)
	defer cancel()
	digest := sha256.Sum256(body)
	messageID := hex.EncodeToString(digest[:])
	// Keep entries until the consumer explicitly removes them.
	reply, err := redis.DoContext(v.connection, ctx, "XADD", v.cfg.StreamName, "*", "message_id", messageID, "payload", body)
	if err != nil {
		if ctx.Err() != nil {
			return ctx.Err()
		}
		return fmt.Errorf("Can't write Valkey stream entry: %w", err)
	}
	id, err := redis.String(reply, nil)
	if err != nil {
		return fmt.Errorf("Invalid Valkey stream entry ID: %w", err)
	}
	milliseconds, sequence, found := strings.Cut(id, "-")
	if !found {
		return fmt.Errorf("Invalid Valkey stream entry ID")
	}
	ms, err := strconv.ParseUint(milliseconds, 10, 64)
	if err != nil {
		return fmt.Errorf("Invalid Valkey stream entry time: %w", err)
	}
	seq, err := strconv.ParseUint(sequence, 10, 64)
	if err != nil {
		return fmt.Errorf("Invalid Valkey stream entry sequence: %w", err)
	}
	if ms == 0 && seq == 0 {
		return fmt.Errorf("Invalid zero Valkey stream entry ID")
	}
	return nil
}

func (v *Valkey) Close() error {
	v.mu.Lock()
	defer v.mu.Unlock()
	v.closed = true
	var err error
	if v.connection != nil {
		if v.connection.Err() == nil {
			err = v.connection.Close()
		}
		v.connection = nil
	}
	return err
}
