package server

import (
	"context"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"net"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/LdDl/go-egts/egts/packet"
	"github.com/LdDl/go-egts/gateway/configuration"
	amqp "github.com/rabbitmq/amqp091-go"
	"github.com/stretchr/testify/assert"
)

func TestRabbitMQIndependentDeliveryAndRestoreIntegration(t *testing.T) {
	if os.Getenv("GO_EGTS_RABBITMQ_INTEGRATION") != "1" {
		t.Skip("Set GO_EGTS_RABBITMQ_INTEGRATION=1 with the example RabbitMQ infrastructure running")
	}
	cfg, err := configuration.PrepareFileConfiguration("../examples/rabbitmq/gateway.toml")
	assert.NoError(t, err)
	if err != nil {
		return
	}
	cfg.DeliveryCfg.AckMode = "queued"
	cfg.DeliveryCfg.QueueCapacity = 2
	cfg.DeliveryCfg.DumpDirectory = t.TempDir()
	channels := make(map[string]*amqp.Channel)
	queueName := fmt.Sprintf("egts.test.%d", time.Now().UnixNano())
	for i := range cfg.DestinationsCfg.RabbitMQ {
		rabbit := cfg.DestinationsCfg.RabbitMQ[i]
		cfg.DestinationsCfg.RabbitMQ[i].QueueName = queueName
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
		_, err = channel.QueueDeclare(queueName, true, false, false, false, amqp.Table{"x-queue-type": "classic"})
		assert.NoError(t, err)
		t.Cleanup(func() {
			_, err := channel.QueueDelete(queueName, false, false, false)
			assert.NoError(t, err)
		})
		channels[rabbit.ID] = channel
	}
	blocked, err := net.Listen("tcp", "127.0.0.1:0")
	assert.NoError(t, err)
	if err != nil {
		return
	}
	t.Cleanup(func() {
		err := blocked.Close()
		assert.NoError(t, err)
	})
	backupPort := cfg.DestinationsCfg.RabbitMQ[1].Port
	cfg.DestinationsCfg.RabbitMQ[1].Port = blocked.Addr().(*net.TCPAddr).Port
	s, err := newServer(cfg)
	assert.NoError(t, err)
	if err != nil {
		return
	}
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	assert.NoError(t, err)
	if err != nil {
		return
	}
	ctx, cancel := context.WithCancel(context.Background())
	done := make(chan error, 1)
	stopped := make(chan struct{})
	go func() {
		done <- s.serve(ctx, listener)
		close(stopped)
	}()
	t.Cleanup(func() {
		cancel()
		<-stopped
	})
	conn, err := net.DialTimeout("tcp", listener.Addr().String(), time.Second)
	assert.NoError(t, err)
	if err != nil {
		return
	}
	t.Cleanup(func() {
		err := conn.Close()
		assert.NoError(t, err)
	})
	err = conn.SetDeadline(time.Now().Add(5 * time.Second))
	assert.NoError(t, err)
	raw, err := hex.DecodeString("0100000b002300000001991800000001ef0000000202101500d2312b104fba3a9ed227bc35030000b200000000006a8d")
	assert.NoError(t, err)
	for i := 0; i < 3; i++ {
		_, err = conn.Write(raw)
		assert.NoError(t, err)
		answer, err := packet.ReadFrame(conn)
		assert.NoError(t, err)
		if err != nil {
			return
		}
		response, err := packet.ReadPacket(answer)
		assert.NoError(t, err)
		if i < 2 {
			assert.Zero(t, response.ServicesFrameData.(*packet.PTResponse).ProcessingResult)
		} else {
			assert.Equal(t, packet.EGTS_PC_NO_RES_AVAIL, response.ServicesFrameData.(*packet.PTResponse).ProcessingResult)
		}
	}
	assert.Eventually(t, func() bool {
		s.mu.Lock()
		defer s.mu.Unlock()
		return len(s.queue) == 2 && !s.queue[0].rabbitmq["monitoring"] && !s.queue[1].rabbitmq["monitoring"]
	}, 3*time.Second, 10*time.Millisecond)
	cancel()
	err = <-done
	assert.NoError(t, err)
	var delivered [][]byte
	var messageIDs []string
	for i := 0; i < 2; i++ {
		message, found, err := channels["monitoring"].Get(queueName, true)
		assert.NoError(t, err)
		assert.True(t, found)
		var event receivedEvent
		err = json.Unmarshal(message.Body, &event)
		assert.NoError(t, err)
		assert.Equal(t, raw, event.Record.Raw)
		delivered = append(delivered, message.Body)
		messageIDs = append(messageIDs, message.MessageId)
	}
	assert.NotEqual(t, messageIDs[0], messageIDs[1])
	cfg.DestinationsCfg.RabbitMQ[1].Port = backupPort
	cfg.DestinationsCfg.RabbitMQ[0], cfg.DestinationsCfg.RabbitMQ[1] = cfg.DestinationsCfg.RabbitMQ[1], cfg.DestinationsCfg.RabbitMQ[0]
	restored, err := newServer(cfg)
	assert.NoError(t, err)
	if err != nil {
		return
	}
	assert.Len(t, restored.queue, 2)
	for _, item := range restored.queue {
		assert.Equal(t, map[string]bool{"backup": true}, item.rabbitmq)
	}
	last := restored.queue[1]
	listener, err = net.Listen("tcp", "127.0.0.1:0")
	assert.NoError(t, err)
	if err != nil {
		return
	}
	recovery, stopRecovery := context.WithCancel(context.Background())
	recovered := make(chan error, 1)
	finished := make(chan struct{})
	go func() {
		recovered <- restored.serve(recovery, listener)
		close(finished)
	}()
	t.Cleanup(func() {
		stopRecovery()
		<-finished
	})
	select {
	case <-last.done:
	case <-time.After(5 * time.Second):
		t.Error("Restored RabbitMQ packets were not delivered")
	}
	stopRecovery()
	err = <-recovered
	assert.NoError(t, err)
	for i, body := range delivered {
		message, found, err := channels["backup"].Get(queueName, true)
		assert.NoError(t, err)
		assert.True(t, found)
		assert.Equal(t, body, message.Body)
		assert.Equal(t, messageIDs[i], message.MessageId)
	}
	for _, channel := range channels {
		queue, err := channel.QueueInspect(queueName)
		assert.NoError(t, err)
		assert.Zero(t, queue.Messages)
	}
	data, err := os.ReadFile(filepath.Join(cfg.DeliveryCfg.DumpDirectory, configuration.DUMP_FILENAME))
	assert.NoError(t, err)
	assert.JSONEq(t, `{"version":4}`, string(data))
}
