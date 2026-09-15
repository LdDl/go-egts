package destination

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"net"
	"strconv"
	"sync"
	"time"

	"github.com/LdDl/go-egts/gateway/configuration"
	amqp "github.com/rabbitmq/amqp091-go"
)

type RabbitMQ struct {
	mu            sync.Mutex
	cfg           configuration.RabbitMQDestinationConf
	socket        net.Conn
	connection    *amqp.Connection
	channel       *amqp.Channel
	confirmations <-chan amqp.Confirmation
	returned      <-chan amqp.Return
	channelClosed <-chan *amqp.Error
	closed        bool
}

func PrepareRabbitMQ(cfg configuration.RabbitMQDestinationConf) (*RabbitMQ, error) {
	if !cfg.Enabled {
		return nil, fmt.Errorf("RabbitMQ destination is disabled")
	}
	err := cfg.Validate()
	if err != nil {
		return nil, err
	}
	return &RabbitMQ{cfg: cfg}, nil
}

func (r *RabbitMQ) WriteContext(ctx context.Context, record *Record) (err error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	if r.closed {
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
		if err != nil {
			closeErr := r.closeConnection()
			if closeErr != nil {
				err = fmt.Errorf("%w; Can't close RabbitMQ connection: %v", err, closeErr)
			}
		}
	}()
	err = r.connect(ctx)
	if err != nil {
		return fmt.Errorf("Can't prepare RabbitMQ connection: %w", err)
	}
	ctx, cancel := context.WithTimeout(ctx, time.Duration(r.cfg.PublishTimeoutSeconds)*time.Second)
	defer cancel()
	finished := make(chan struct{})
	interrupted := make(chan error, 1)
	socket := r.socket
	// Closing the socket also interrupts AMQP calls blocked while writing.
	go func() {
		select {
		case <-ctx.Done():
			interrupted <- socket.Close()
		case <-finished:
			interrupted <- nil
		}
	}()
	defer func() {
		close(finished)
		closeErr := <-interrupted
		if err != nil && ctx.Err() != nil {
			err = ctx.Err()
		}
		if closeErr != nil && !errors.Is(closeErr, net.ErrClosed) && err == nil {
			err = closeErr
		}
	}()
	digest := sha256.Sum256(body)
	message := amqp.Publishing{
		ContentType: "application/json", DeliveryMode: amqp.Persistent,
		MessageId: hex.EncodeToString(digest[:]), Timestamp: record.ReceivedAt,
		Type: "egts.packet", AppId: "egts_gateway", Body: body,
	}
	sequence := r.channel.GetNextPublishSeqNo()
	err = r.channel.PublishWithContext(ctx, "", r.cfg.QueueName, true, false, message)
	if err != nil {
		return fmt.Errorf("Can't publish RabbitMQ packet: %w", err)
	}
	return r.waitConfirmation(ctx, sequence)
}

func (r *RabbitMQ) connect(ctx context.Context) (err error) {
	if r.connection != nil && !r.connection.IsClosed() && r.channel != nil && !r.channel.IsClosed() {
		return nil
	}
	err = r.closeConnection()
	if err != nil {
		return err
	}
	ctx, cancel := context.WithTimeout(ctx, time.Duration(r.cfg.ConnectTimeoutSeconds)*time.Second)
	defer cancel()
	dialer := net.Dialer{}
	r.socket, err = dialer.DialContext(ctx, "tcp", net.JoinHostPort(r.cfg.Host, strconv.Itoa(r.cfg.Port)))
	if err != nil {
		return err
	}
	socket := r.socket
	finished := make(chan struct{})
	interrupted := make(chan error, 1)
	go func() {
		select {
		case <-ctx.Done():
			interrupted <- socket.Close()
		case <-finished:
			interrupted <- nil
		}
	}()
	defer func() {
		close(finished)
		closeErr := <-interrupted
		if ctx.Err() != nil {
			err = ctx.Err()
		}
		if closeErr != nil && !errors.Is(closeErr, net.ErrClosed) && err == nil {
			err = closeErr
		}
	}()
	r.connection, err = amqp.Open(socket, amqp.Config{
		SASL:  []amqp.Authentication{&amqp.PlainAuth{Username: r.cfg.UserName, Password: r.cfg.Password}},
		Vhost: r.cfg.Vhost, Heartbeat: 10 * time.Second, Locale: "en_US",
	})
	if err != nil {
		return err
	}
	r.channel, err = r.connection.Channel()
	if err != nil {
		return err
	}
	r.confirmations = r.channel.NotifyPublish(make(chan amqp.Confirmation, 1))
	r.returned = r.channel.NotifyReturn(make(chan amqp.Return, 1))
	r.channelClosed = r.channel.NotifyClose(make(chan *amqp.Error, 1))
	err = r.channel.Confirm(false)
	if err != nil {
		return err
	}
	_, err = r.channel.QueueDeclare(r.cfg.QueueName, true, false, false, false, amqp.Table{"x-queue-type": r.cfg.QueueType})
	if err != nil {
		return fmt.Errorf("Can't declare RabbitMQ queue %s: %w", r.cfg.QueueName, err)
	}
	return nil
}

func (r *RabbitMQ) waitConfirmation(ctx context.Context, sequence uint64) error {
	select {
	case <-ctx.Done():
		return ctx.Err()
	case returned, open := <-r.returned:
		if !open {
			return fmt.Errorf("RabbitMQ connection closed before confirmation")
		}
		return fmt.Errorf("RabbitMQ returned packet: %d %s", returned.ReplyCode, returned.ReplyText)
	case closed, open := <-r.channelClosed:
		if open && closed != nil {
			return fmt.Errorf("RabbitMQ channel closed before confirmation: %w", closed)
		}
		return fmt.Errorf("RabbitMQ channel closed before confirmation")
	case confirmation, open := <-r.confirmations:
		if !open {
			return fmt.Errorf("RabbitMQ confirmation channel closed")
		}
		if confirmation.DeliveryTag != sequence {
			return fmt.Errorf("Unexpected RabbitMQ confirmation: got %d, expected %d", confirmation.DeliveryTag, sequence)
		}
		if !confirmation.Ack {
			return fmt.Errorf("RabbitMQ rejected packet publication")
		}
		// RabbitMQ sends mandatory returns before the corresponding confirmation.
		select {
		case returned, open := <-r.returned:
			if open {
				return fmt.Errorf("RabbitMQ returned packet: %d %s", returned.ReplyCode, returned.ReplyText)
			}
		default:
		}
		return nil
	}
}

func (r *RabbitMQ) closeConnection() error {
	var err error
	if r.socket != nil {
		err = r.socket.Close()
		if errors.Is(err, net.ErrClosed) {
			err = nil
		}
	}
	r.socket = nil
	r.connection = nil
	r.channel = nil
	r.confirmations = nil
	r.returned = nil
	r.channelClosed = nil
	return err
}

func (r *RabbitMQ) Close() error {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.closed = true
	var err error
	if r.connection != nil && !r.connection.IsClosed() {
		err = r.connection.CloseDeadline(time.Now().Add(time.Second))
		if errors.Is(err, amqp.ErrClosed) {
			err = nil
		}
	}
	closeErr := r.closeConnection()
	if err != nil {
		return err
	}
	return closeErr
}
