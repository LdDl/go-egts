package destination

import (
	"context"
	"encoding/hex"
	"io"
	"net"
	"testing"
	"time"

	"github.com/LdDl/go-egts/gateway/configuration"
	amqp "github.com/rabbitmq/amqp091-go"
	"github.com/stretchr/testify/assert"
)

type rabbitConfirmationTestCase struct {
	name      string
	mode      string
	errorText string
}

func TestRabbitMQConfirmations(t *testing.T) {
	cases := []rabbitConfirmationTestCase{
		{name: "confirmed", mode: "ack"},
		{name: "rejected", mode: "nack", errorText: "rejected"},
		{name: "wrong sequence", mode: "wrong", errorText: "Unexpected"},
		{name: "returned", mode: "return", errorText: "returned"},
		{name: "returned before ACK", mode: "return and ack", errorText: "returned"},
		{name: "closed confirmations", mode: "closed", errorText: "closed"},
		{name: "closed returns", mode: "returns closed", errorText: "closed"},
		{name: "channel error", mode: "channel error", errorText: "PRECONDITION_FAILED"},
		{name: "channel closed", mode: "channel closed", errorText: "closed"},
		{name: "deadline", mode: "timeout", errorText: "deadline exceeded"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			confirmations := make(chan amqp.Confirmation, 1)
			returned := make(chan amqp.Return, 1)
			closed := make(chan *amqp.Error, 1)
			writer := &RabbitMQ{confirmations: confirmations, returned: returned, channelClosed: closed}
			switch tc.mode {
			case "ack":
				confirmations <- amqp.Confirmation{DeliveryTag: 7, Ack: true}
			case "nack":
				confirmations <- amqp.Confirmation{DeliveryTag: 7, Ack: false}
			case "wrong":
				confirmations <- amqp.Confirmation{DeliveryTag: 6, Ack: true}
			case "return", "return and ack":
				returned <- amqp.Return{ReplyCode: 312, ReplyText: "NO_ROUTE"}
				if tc.mode == "return and ack" {
					confirmations <- amqp.Confirmation{DeliveryTag: 7, Ack: true}
				}
			case "closed":
				close(confirmations)
			case "returns closed":
				close(returned)
			case "channel error":
				closed <- &amqp.Error{Code: 406, Reason: "PRECONDITION_FAILED"}
			case "channel closed":
				close(closed)
			}
			ctx, cancel := context.WithTimeout(context.Background(), 50*time.Millisecond)
			defer cancel()
			err := writer.waitConfirmation(ctx, 7)
			if tc.errorText != "" {
				assert.ErrorContains(t, err, tc.errorText)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestRabbitMQHandshakeCancellation(t *testing.T) {
	for _, mode := range []string{"cancel", "timeout"} {
		t.Run(mode, func(t *testing.T) {
			listener, err := net.Listen("tcp", "127.0.0.1:0")
			assert.NoError(t, err)
			if err != nil {
				return
			}
			t.Cleanup(func() {
				err := listener.Close()
				assert.NoError(t, err)
			})
			cfg := configuration.DefaultRabbitMQDestination()
			cfg.ID = "primary"
			cfg.Enabled = true
			cfg.Port = listener.Addr().(*net.TCPAddr).Port
			cfg.UserName = "user"
			cfg.Password = "secret"
			cfg.ConnectTimeoutSeconds = 1
			writer, err := PrepareRabbitMQ(cfg)
			assert.NoError(t, err)
			t.Cleanup(func() {
				err := writer.Close()
				assert.NoError(t, err)
			})
			raw, err := hex.DecodeString("0100000b002300000001991800000001ef0000000202101500d2312b104fba3a9ed227bc35030000b200000000006a8d")
			assert.NoError(t, err)
			record, err := NewRecord(time.Now(), Source{}, raw)
			assert.NoError(t, err)
			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()
			done := make(chan error, 1)
			go func() {
				done <- writer.WriteContext(ctx, record)
			}()
			err = listener.(*net.TCPListener).SetDeadline(time.Now().Add(2 * time.Second))
			assert.NoError(t, err)
			peer, err := listener.Accept()
			assert.NoError(t, err)
			if err != nil {
				return
			}
			t.Cleanup(func() {
				err := peer.Close()
				assert.NoError(t, err)
			})
			err = peer.SetDeadline(time.Now().Add(2 * time.Second))
			assert.NoError(t, err)
			header := make([]byte, 8)
			_, err = io.ReadFull(peer, header)
			assert.NoError(t, err)
			assert.Equal(t, []byte{'A', 'M', 'Q', 'P', 0, 0, 9, 1}, header)
			if mode == "cancel" {
				cancel()
			}
			select {
			case err = <-done:
				if mode == "cancel" {
					assert.ErrorIs(t, err, context.Canceled)
				} else {
					assert.ErrorIs(t, err, context.DeadlineExceeded)
				}
			case <-time.After(2 * time.Second):
				t.Fatal("RabbitMQ handshake did not stop")
			}
			assert.Nil(t, writer.socket)
			assert.Nil(t, writer.connection)
			err = writer.Close()
			assert.NoError(t, err)
			err = writer.WriteContext(context.Background(), record)
			assert.ErrorIs(t, err, net.ErrClosed)
		})
	}
}
