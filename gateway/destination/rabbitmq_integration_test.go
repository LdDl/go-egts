package destination

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"os"
	"testing"
	"time"

	"github.com/LdDl/go-egts/gateway/configuration"
	amqp "github.com/rabbitmq/amqp091-go"
	"github.com/stretchr/testify/assert"
)

func TestRabbitMQIntegration(t *testing.T) {
	if os.Getenv("GO_EGTS_RABBITMQ_INTEGRATION") != "1" {
		t.Skip("Set GO_EGTS_RABBITMQ_INTEGRATION=1 with the example RabbitMQ infrastructure running")
	}
	cfg, err := configuration.PrepareFileConfiguration("../examples/rabbitmq/gateway.toml")
	assert.NoError(t, err)
	if err != nil {
		return
	}
	for _, rabbit := range cfg.DestinationsCfg.RabbitMQ {
		for _, queueType := range []string{"classic", "quorum"} {
			t.Run(rabbit.ID+"/"+queueType, func(t *testing.T) {
				rabbit.QueueName = fmt.Sprintf("egts.test.%d", time.Now().UnixNano())
				rabbit.QueueType = queueType
				uri := amqp.URI{Scheme: "amqp", Host: rabbit.Host, Port: rabbit.Port, Username: rabbit.UserName, Password: rabbit.Password, Vhost: rabbit.Vhost}
				connection, err := amqp.Dial(uri.String())
				assert.NoError(t, err)
				if err != nil {
					return
				}
				t.Cleanup(func() {
					err := connection.Close()
					assert.NoError(t, err)
				})
				channel, err := connection.Channel()
				assert.NoError(t, err)
				if err != nil {
					return
				}
				t.Cleanup(func() {
					_, err := channel.QueueDelete(rabbit.QueueName, false, false, false)
					assert.NoError(t, err)
				})
				writer, err := PrepareRabbitMQ(rabbit)
				assert.NoError(t, err)
				t.Cleanup(func() {
					err := writer.Close()
					assert.NoError(t, err)
				})
				raw, err := hex.DecodeString("0100000b002300000001991800000001ef0000000202101500d2312b104fba3a9ed227bc35030000b200000000006a8d")
				assert.NoError(t, err)
				record, err := NewRecord(time.Now(), Source{}, raw)
				assert.NoError(t, err)
				body, err := record.Encode()
				assert.NoError(t, err)
				digest := sha256.Sum256(body)
				ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
				defer cancel()
				for _, mode := range []string{"initial", "reconnect", "returned"} {
					switch mode {
					case "reconnect":
						err = writer.connection.Close()
						assert.NoError(t, err)
					case "returned":
						_, err = channel.QueueDelete(rabbit.QueueName, false, false, false)
						assert.NoError(t, err)
						err = writer.WriteContext(ctx, record)
						assert.ErrorContains(t, err, "returned")
						assert.Nil(t, writer.connection)
					}
					err = writer.WriteContext(ctx, record)
					assert.NoError(t, err)
					if err != nil {
						return
					}
					message, found, err := channel.Get(rabbit.QueueName, true)
					assert.NoError(t, err)
					assert.True(t, found)
					assert.Equal(t, body, message.Body)
					assert.Equal(t, "application/json", message.ContentType)
					assert.Equal(t, uint8(amqp.Persistent), message.DeliveryMode)
					assert.Equal(t, hex.EncodeToString(digest[:]), message.MessageId)
					assert.Equal(t, "egts_gateway", message.AppId)
					assert.Equal(t, "egts.packet", message.Type)
					assert.Equal(t, record.ReceivedAt.Unix(), message.Timestamp.Unix())
					queue, err := channel.QueueInspect(rabbit.QueueName)
					assert.NoError(t, err)
					assert.Zero(t, queue.Messages)
				}
			})
		}
	}
}

func TestRabbitMQQueueMismatchIntegration(t *testing.T) {
	if os.Getenv("GO_EGTS_RABBITMQ_INTEGRATION") != "1" {
		t.Skip("Set GO_EGTS_RABBITMQ_INTEGRATION=1 with the example RabbitMQ infrastructure running")
	}
	cfg, err := configuration.PrepareFileConfiguration("../examples/rabbitmq/gateway.toml")
	assert.NoError(t, err)
	if err != nil {
		return
	}
	rabbit := cfg.DestinationsCfg.RabbitMQ[0]
	rabbit.QueueName = fmt.Sprintf("egts.test.%d", time.Now().UnixNano())
	uri := amqp.URI{Scheme: "amqp", Host: rabbit.Host, Port: rabbit.Port, Username: rabbit.UserName, Password: rabbit.Password, Vhost: rabbit.Vhost}
	connection, err := amqp.Dial(uri.String())
	assert.NoError(t, err)
	if err != nil {
		return
	}
	t.Cleanup(func() {
		err := connection.Close()
		assert.NoError(t, err)
	})
	channel, err := connection.Channel()
	assert.NoError(t, err)
	if err != nil {
		return
	}
	_, err = channel.QueueDeclare(rabbit.QueueName, true, false, false, false, amqp.Table{"x-queue-type": "classic", "x-max-length": int64(100)})
	assert.NoError(t, err)
	if err != nil {
		return
	}
	t.Cleanup(func() {
		_, err := channel.QueueDelete(rabbit.QueueName, false, false, false)
		assert.NoError(t, err)
	})
	writer, err := PrepareRabbitMQ(rabbit)
	assert.NoError(t, err)
	t.Cleanup(func() {
		err := writer.Close()
		assert.NoError(t, err)
	})
	raw, err := hex.DecodeString("0100000b002300000001991800000001ef0000000202101500d2312b104fba3a9ed227bc35030000b200000000006a8d")
	assert.NoError(t, err)
	record, err := NewRecord(time.Now(), Source{}, raw)
	assert.NoError(t, err)
	err = writer.WriteContext(context.Background(), record)
	assert.ErrorContains(t, err, "PRECONDITION_FAILED")
	assert.Nil(t, writer.connection)
	queue, err := channel.QueueInspect(rabbit.QueueName)
	assert.NoError(t, err)
	assert.Zero(t, queue.Messages)
	rabbit.Password = "incorrect-test-password"
	wrongCredentials, err := PrepareRabbitMQ(rabbit)
	assert.NoError(t, err)
	err = wrongCredentials.WriteContext(context.Background(), record)
	assert.Error(t, err)
	if err != nil {
		assert.NotContains(t, err.Error(), rabbit.Password)
	}
	err = wrongCredentials.Close()
	assert.NoError(t, err)
}
