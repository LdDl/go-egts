package server

import (
	"bytes"
	"encoding/hex"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/LdDl/go-egts/gateway/configuration"
	"github.com/LdDl/go-egts/gateway/destination"
	"github.com/stretchr/testify/assert"
)

type rabbitDumpTestCase struct {
	name      string
	version   int
	pending   []string
	disabled  string
	errorText string
}

func TestRabbitMQDumpDestinations(t *testing.T) {
	cases := []rabbitDumpTestCase{
		{name: "restore only pending destination", version: 4, pending: []string{"backup"}},
		{name: "missing destination", version: 4, pending: []string{"missing"}, errorText: "unavailable destination"},
		{name: "disabled destination", version: 4, pending: []string{"backup"}, disabled: "backup", errorText: "unavailable destination"},
		{name: "duplicate destination", version: 4, pending: []string{"backup", "backup"}, errorText: "Duplicate"},
		{name: "RabbitMQ in old dump", version: 3, pending: []string{"backup"}, errorText: "legacy packet dump"},
		{name: "legacy local destination", version: 1},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			cfg := configuration.DefaultConfiguration()
			cfg.DeliveryCfg.DumpDirectory = t.TempDir()
			cfg.DestinationsCfg.Stdout = false
			cfg.DestinationsCfg.File.Enabled = true
			cfg.DestinationsCfg.File.Directory = t.TempDir()
			for _, id := range []string{"backup", "monitoring"} {
				rabbit := configuration.DefaultRabbitMQDestination()
				rabbit.ID = id
				rabbit.Enabled = id != tc.disabled
				rabbit.UserName = "user"
				rabbit.Password = "secret"
				cfg.DestinationsCfg.RabbitMQ = append(cfg.DestinationsCfg.RabbitMQ, rabbit)
			}
			raw, err := hex.DecodeString("0100000b002300000001991800000001ef0000000202101500d2312b104fba3a9ed227bc35030000b200000000006a8d")
			assert.NoError(t, err)
			saved := dumpRecord{ReceivedAt: time.Now().UTC(), Raw: raw, RabbitMQIDs: tc.pending, File: true}
			var buffer bytes.Buffer
			encoder := json.NewEncoder(&buffer)
			err = encoder.Encode(dumpHeader{Version: tc.version})
			assert.NoError(t, err)
			err = encoder.Encode(saved)
			assert.NoError(t, err)
			name := filepath.Join(cfg.DeliveryCfg.DumpDirectory, configuration.DUMP_FILENAME)
			err = os.WriteFile(name, buffer.Bytes(), 0600)
			assert.NoError(t, err)
			s, err := newServer(cfg)
			if tc.errorText != "" {
				assert.ErrorContains(t, err, tc.errorText)
				assert.Nil(t, s)
			} else {
				assert.NoError(t, err)
				if err != nil {
					return
				}
				assert.Len(t, s.queue, 1)
				item := s.queue[0]
				assert.Equal(t, raw, item.record.Raw)
				assert.Equal(t, saved.ReceivedAt, item.record.ReceivedAt)
				assert.True(t, item.file)
				assert.False(t, item.stdout)
				assert.Nil(t, item.relay)
				assert.Empty(t, item.egts)
				if tc.version == 1 {
					assert.Empty(t, item.rabbitmq)
				} else {
					assert.Equal(t, map[string]bool{"backup": true}, item.rabbitmq)
				}
				err = s.file.Close()
				assert.NoError(t, err)
				for _, writer := range s.rabbits {
					err = writer.Close()
					assert.NoError(t, err)
				}
			}
			data, err := os.ReadFile(name)
			assert.NoError(t, err)
			assert.Equal(t, buffer.Bytes(), data)
		})
	}
}

func TestRabbitMQEnqueueAndSave(t *testing.T) {
	cfg := configuration.DefaultConfiguration()
	cfg.DeliveryCfg.DumpDirectory = t.TempDir()
	cfg.DestinationsCfg.Stdout = false
	for _, id := range []string{"monitoring", "backup", "disabled"} {
		rabbit := configuration.DefaultRabbitMQDestination()
		rabbit.ID = id
		rabbit.Enabled = id != "disabled"
		rabbit.UserName = "user"
		rabbit.Password = "secret"
		cfg.DestinationsCfg.RabbitMQ = append(cfg.DestinationsCfg.RabbitMQ, rabbit)
	}
	s, err := newServer(cfg)
	assert.NoError(t, err)
	if err != nil {
		return
	}
	t.Cleanup(func() {
		for _, writer := range s.rabbits {
			err := writer.Close()
			assert.NoError(t, err)
		}
	})
	s.accepting = true
	raw, err := hex.DecodeString("0100000b002300000001991800000001ef0000000202101500d2312b104fba3a9ed227bc35030000b200000000006a8d")
	assert.NoError(t, err)
	record, err := destination.NewRecord(time.Now(), destination.Source{}, raw)
	assert.NoError(t, err)
	item, err := s.enqueue(record, nil)
	assert.NoError(t, err)
	assert.Equal(t, map[string]bool{"monitoring": true, "backup": true}, item.rabbitmq)
	assert.Nil(t, item.relay)
	delete(item.rabbitmq, "monitoring")
	err = s.saveDump()
	assert.NoError(t, err)
	data, err := os.ReadFile(filepath.Join(cfg.DeliveryCfg.DumpDirectory, configuration.DUMP_FILENAME))
	assert.NoError(t, err)
	decoder := json.NewDecoder(bytes.NewReader(data))
	var header dumpHeader
	err = decoder.Decode(&header)
	assert.NoError(t, err)
	assert.Equal(t, 4, header.Version)
	var saved dumpRecord
	err = decoder.Decode(&saved)
	assert.NoError(t, err)
	assert.Equal(t, []string{"backup"}, saved.RabbitMQIDs)
	assert.Equal(t, record.Raw, saved.Raw)
	assert.False(t, saved.Stdout)
	assert.Nil(t, saved.SessionID)
	assert.Contains(t, string(data), `"identity":null`)
}
