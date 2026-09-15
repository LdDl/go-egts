package destination

import (
	"bytes"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sync"
	"testing"
	"time"

	"github.com/LdDl/go-egts/gateway/configuration"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
	"github.com/stretchr/testify/assert"
)

type destinationRotationTestCase struct {
	name     string
	backups  int
	capacity int
	expected []string
}

type destinationWriteErrorTestCase struct {
	name string
	n    int
	err  error
}

type failingWriter struct {
	n     int
	err   error
	calls int
}

func (w *failingWriter) Write(data []byte) (int, error) {
	w.calls++
	return w.n, w.err
}

func TestStdoutDestination(t *testing.T) {
	root := t.TempDir()
	cfg := configuration.DefaultConfiguration()
	cfg.DeliveryCfg.DumpDirectory = filepath.Join(root, "queue")
	output, err := os.Create(filepath.Join(root, "stdout.ndjson"))
	assert.NoError(t, err)
	if err != nil {
		return
	}
	t.Cleanup(func() {
		err := output.Close()
		assert.NoError(t, err)
	})
	previousStdout := os.Stdout
	os.Stdout = output
	writer, err := PrepareStdout(cfg)
	os.Stdout = previousStdout
	assert.NoError(t, err)
	if err != nil {
		return
	}
	raw, err := hex.DecodeString("0100030b001000000000b3000000060000005802020003000000002ec1")
	assert.NoError(t, err)
	record, err := NewRecord(time.Now(), Source{RemoteAddress: "127.0.0.1:12345"}, raw)
	assert.NoError(t, err)
	if err != nil {
		return
	}
	expected, err := record.Encode()
	assert.NoError(t, err)
	err = writer.Write(record)
	assert.NoError(t, err)
	err = writer.Close()
	assert.NoError(t, err)
	err = writer.Close()
	assert.NoError(t, err)
	err = writer.Write(record)
	assert.ErrorIs(t, err, os.ErrClosed)
	// Closing a destination must leave the process stdout descriptor open.
	_, err = output.Write(expected)
	assert.NoError(t, err)
	actual, err := os.ReadFile(output.Name())
	assert.NoError(t, err)
	assert.Equal(t, bytes.Repeat(expected, 2), actual)
}

func TestFileDestinationRotation(t *testing.T) {
	cases := []destinationRotationTestCase{
		{name: "all records", backups: 5, capacity: 6, expected: []string{"source-0", "source-1", "source-2", "source-3", "source-4"}},
		{name: "archive count", backups: 1, capacity: 6, expected: []string{"source-2", "source-3", "source-4"}},
		{name: "total bytes", backups: 5, capacity: 2, expected: []string{"source-4"}},
		{name: "no archives", backups: 0, capacity: 6, expected: []string{"source-4"}},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			root := t.TempDir()
			cfg := configuration.DefaultConfiguration()
			cfg.DestinationsCfg.Stdout = false
			cfg.DestinationsCfg.File.Enabled = true
			cfg.DestinationsCfg.File.Directory = filepath.Join(root, "packets")
			cfg.DeliveryCfg.DumpDirectory = filepath.Join(root, "queue")
			raw, err := hex.DecodeString("0100000b002300000001991800000001ef0000000202101500d2312b104fba3a9ed227bc35030000b200000000006a8d")
			assert.NoError(t, err)
			record, err := NewRecord(time.Now(), Source{RemoteAddress: "source-0"}, raw)
			assert.NoError(t, err)
			if err != nil {
				return
			}
			data, err := record.Encode()
			assert.NoError(t, err)
			cfg.DestinationsCfg.File.Rotation = configuration.RotationConf{
				MaxFileSizeBytes: int64(len(data) * 2), MaxBackups: tc.backups, MaxTotalSizeBytes: int64(len(data) * tc.capacity),
			}
			writer, err := PrepareFile(cfg)
			assert.NoError(t, err)
			if err != nil {
				return
			}
			for i := 0; i < 5; i++ {
				record.Source.RemoteAddress = fmt.Sprintf("source-%d", i)
				err = writer.Write(record)
				assert.NoError(t, err)
				if i == 2 {
					err = writer.Close()
					assert.NoError(t, err)
					writer, err = PrepareFile(cfg)
					assert.NoError(t, err)
					if err != nil {
						return
					}
				}
			}
			err = writer.Close()
			assert.NoError(t, err)
			entries, err := os.ReadDir(cfg.DestinationsCfg.File.Directory)
			assert.NoError(t, err)
			assert.LessOrEqual(t, len(entries), tc.backups+1)
			var sources []string
			var total int64
			for _, entry := range entries {
				assert.True(t, entry.Name() == "packets.ndjson" || filepath.Ext(entry.Name()) == ".ndjson")
				data, err := os.ReadFile(filepath.Join(cfg.DestinationsCfg.File.Directory, entry.Name()))
				assert.NoError(t, err)
				assert.LessOrEqual(t, int64(len(data)), cfg.DestinationsCfg.File.Rotation.MaxFileSizeBytes)
				assert.True(t, bytes.HasSuffix(data, []byte("\n")))
				total += int64(len(data))
				for _, line := range bytes.Split(bytes.TrimSuffix(data, []byte("\n")), []byte("\n")) {
					var event encodedEvent
					err = json.Unmarshal(line, &event)
					assert.NoError(t, err)
					assert.Equal(t, raw, event.Record.Raw)
					sources = append(sources, event.Record.Source.RemoteAddress)
				}
			}
			assert.Equal(t, tc.expected, sources)
			assert.LessOrEqual(t, total, cfg.DestinationsCfg.File.Rotation.MaxTotalSizeBytes)
		})
	}
}

func TestDestinationWriteErrors(t *testing.T) {
	sentinel := errors.New("destination unavailable")
	cases := []destinationWriteErrorTestCase{
		{name: "write failure", n: 0, err: sentinel},
		{name: "empty short write", n: 0},
		{name: "partial short write", n: 7},
		{name: "partial failure", n: 7, err: sentinel},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			cfg := configuration.DefaultConfiguration()
			cfg.DeliveryCfg.DumpDirectory = filepath.Join(t.TempDir(), "queue")
			writer, err := PrepareStdout(cfg)
			assert.NoError(t, err)
			if err != nil {
				return
			}
			output := &failingWriter{n: tc.n, err: tc.err}
			writer.output = output
			raw, err := hex.DecodeString("0100030b001000000000b3000000060000005802020003000000002ec1")
			assert.NoError(t, err)
			record, err := NewRecord(time.Now(), Source{}, raw)
			assert.NoError(t, err)
			if err != nil {
				return
			}
			err = writer.Write(nil)
			assert.Error(t, err)
			assert.Zero(t, output.calls)
			err = writer.Write(record)
			expectedError := tc.err
			if expectedError == nil {
				expectedError = io.ErrShortWrite
			}
			assert.ErrorIs(t, err, expectedError)
			assert.Contains(t, err.Error(), "stdout")
			assert.Equal(t, 1, output.calls)
			var recovered bytes.Buffer
			writer.output = &recovered
			err = writer.Write(record)
			if tc.n > 0 {
				assert.ErrorIs(t, err, expectedError)
				assert.Empty(t, recovered.Bytes())
			} else {
				assert.NoError(t, err)
				assert.True(t, json.Valid(recovered.Bytes()))
			}
			err = writer.Close()
			if tc.n > 0 {
				assert.ErrorIs(t, err, expectedError)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestDestinationsIndependent(t *testing.T) {
	previousLogger := log.Logger
	t.Cleanup(func() {
		log.Logger = previousLogger
	})
	var applicationLogs bytes.Buffer
	log.Logger = zerolog.New(&applicationLogs)
	root := t.TempDir()
	cfg := configuration.DefaultConfiguration()
	cfg.DeliveryCfg.DumpDirectory = filepath.Join(root, "queue")
	cfg.DestinationsCfg.File.Enabled = true
	cfg.DestinationsCfg.File.Directory = filepath.Join(root, "packets")
	cfg.DestinationsCfg.File.Rotation.MaxFileSizeBytes = 64
	cfg.DestinationsCfg.File.Rotation.MaxTotalSizeBytes = 128
	stdout, err := PrepareStdout(cfg)
	assert.NoError(t, err)
	if err != nil {
		return
	}
	var output bytes.Buffer
	stdout.output = &output
	file, err := PrepareFile(cfg)
	assert.NoError(t, err)
	if err != nil {
		return
	}
	raw, err := hex.DecodeString("0100030b001000000000b3000000060000005802020003000000002ec1")
	assert.NoError(t, err)
	record, err := NewRecord(time.Now(), Source{}, raw)
	assert.NoError(t, err)
	err = stdout.Write(record)
	assert.NoError(t, err)
	err = file.Write(record)
	assert.Error(t, err)
	err = file.Write(record)
	assert.Error(t, err)
	expected, err := record.Encode()
	assert.NoError(t, err)
	assert.Equal(t, expected, output.Bytes())
	assert.Empty(t, applicationLogs.Bytes())
	log.Log().Str("scope", "startup").Str("event", "startup").Msg("application only")
	assert.Contains(t, applicationLogs.String(), "application only")
	assert.NotContains(t, output.String(), "application only")
	data, err := os.ReadFile(filepath.Join(cfg.DestinationsCfg.File.Directory, configuration.PACKETS_FILENAME))
	assert.NoError(t, err)
	assert.Empty(t, data)
	err = file.Close()
	assert.Error(t, err)
	err = stdout.Close()
	assert.NoError(t, err)
}

func TestDestinationConcurrentWrites(t *testing.T) {
	for _, destination := range []string{"stdout", "file"} {
		t.Run(destination, func(t *testing.T) {
			root := t.TempDir()
			cfg := configuration.DefaultConfiguration()
			cfg.DeliveryCfg.DumpDirectory = filepath.Join(root, "queue")
			cfg.DestinationsCfg.File.Enabled = true
			cfg.DestinationsCfg.File.Directory = filepath.Join(root, "packets")
			cfg.DestinationsCfg.File.Rotation = configuration.RotationConf{MaxFileSizeBytes: 2048, MaxBackups: 64, MaxTotalSizeBytes: 131072}
			var output bytes.Buffer
			var writer *Writer
			var err error
			if destination == "stdout" {
				writer, err = PrepareStdout(cfg)
			} else {
				writer, err = PrepareFile(cfg)
			}
			assert.NoError(t, err)
			if err != nil {
				return
			}
			if destination == "stdout" {
				writer.output = &output
			}
			raw, err := hex.DecodeString("0100030b001000000000b3000000060000005802020003000000002ec1")
			assert.NoError(t, err)
			var group sync.WaitGroup
			for i := 0; i < 64; i++ {
				group.Add(1)
				go func(i int) {
					defer group.Done()
					record, err := NewRecord(time.Now(), Source{RemoteAddress: fmt.Sprintf("source-%d", i)}, raw)
					assert.NoError(t, err)
					err = writer.Write(record)
					assert.NoError(t, err)
				}(i)
			}
			group.Wait()
			err = writer.Close()
			assert.NoError(t, err)
			if destination == "file" {
				entries, err := os.ReadDir(cfg.DestinationsCfg.File.Directory)
				assert.NoError(t, err)
				for _, entry := range entries {
					data, err := os.ReadFile(filepath.Join(cfg.DestinationsCfg.File.Directory, entry.Name()))
					assert.NoError(t, err)
					assert.LessOrEqual(t, len(data), 2048)
					output.Write(data)
				}
			}
			decoder := json.NewDecoder(&output)
			var sources []string
			for decoder.More() {
				var event encodedEvent
				err = decoder.Decode(&event)
				assert.NoError(t, err)
				if err != nil {
					break
				}
				assert.Equal(t, raw, event.Record.Raw)
				sources = append(sources, event.Record.Source.RemoteAddress)
			}
			var expected []string
			for i := 0; i < 64; i++ {
				expected = append(expected, fmt.Sprintf("source-%d", i))
			}
			assert.ElementsMatch(t, expected, sources)
		})
	}
}

func TestDestinationOutputConflicts(t *testing.T) {
	for _, stage := range []string{"prepare", "write", "rotate"} {
		t.Run(stage, func(t *testing.T) {
			root := t.TempDir()
			cfg := configuration.DefaultConfiguration()
			cfg.DeliveryCfg.DumpDirectory = filepath.Join(root, "queue")
			cfg.LogsCfg.Output = "file"
			cfg.LogsCfg.Directory = filepath.Join(root, "logs")
			cfg.DestinationsCfg.File.Enabled = true
			cfg.DestinationsCfg.File.Directory = filepath.Join(root, "packets")
			err := os.MkdirAll(cfg.LogsCfg.Directory, 0750)
			assert.NoError(t, err)
			err = os.MkdirAll(cfg.DestinationsCfg.File.Directory, 0750)
			assert.NoError(t, err)
			applicationLog := filepath.Join(cfg.LogsCfg.Directory, configuration.LOG_FILENAME)
			err = os.WriteFile(applicationLog, []byte("application log\n"), 0640)
			assert.NoError(t, err)
			raw, err := hex.DecodeString("0100030b001000000000b3000000060000005802020003000000002ec1")
			assert.NoError(t, err)
			record, err := NewRecord(time.Now(), Source{}, raw)
			assert.NoError(t, err)
			if err != nil {
				return
			}
			data, err := record.Encode()
			assert.NoError(t, err)
			cfg.DestinationsCfg.File.Rotation.MaxFileSizeBytes = int64(len(data))
			packetPath := filepath.Join(cfg.DestinationsCfg.File.Directory, configuration.PACKETS_FILENAME)
			var file *Writer
			if stage == "prepare" {
				err = os.Link(applicationLog, packetPath)
				assert.NoError(t, err)
				file, err = PrepareFile(cfg)
				assert.Nil(t, file)
				var conflict *configuration.OutputConflictError
				assert.ErrorAs(t, err, &conflict)
			} else {
				file, err = PrepareFile(cfg)
				assert.NoError(t, err)
				if err != nil {
					return
				}
				if stage == "rotate" {
					err = file.Write(record)
					assert.NoError(t, err)
				}
				archive := filepath.Join(cfg.DestinationsCfg.File.Directory, "packets-20000101T000000.000000000Z.ndjson")
				err = os.Link(applicationLog, archive)
				assert.NoError(t, err)
				err = file.Write(record)
				var conflict *configuration.OutputConflictError
				assert.ErrorAs(t, err, &conflict)
				err = file.Close()
				assert.ErrorAs(t, err, &conflict)
				content, err := os.ReadFile(archive)
				assert.NoError(t, err)
				assert.Equal(t, "application log\n", string(content))
				content, err = os.ReadFile(packetPath)
				assert.NoError(t, err)
				if stage == "rotate" {
					assert.Equal(t, data, content)
				} else {
					assert.Empty(t, content)
				}
			}
			content, err := os.ReadFile(applicationLog)
			assert.NoError(t, err)
			assert.Equal(t, "application log\n", string(content))
		})
	}
}

func TestDisabledDestinations(t *testing.T) {
	cfg := configuration.DefaultConfiguration()
	cfg.DestinationsCfg.Stdout = false
	writer, err := PrepareStdout(cfg)
	assert.Error(t, err)
	assert.Nil(t, writer)
	writer, err = PrepareFile(cfg)
	assert.Error(t, err)
	assert.Nil(t, writer)
	// Configuration conflicts must be reported before opening a destination.
	cfg.DestinationsCfg.Stdout = true
	cfg.LogsCfg.Output = "stdout"
	writer, err = PrepareStdout(cfg)
	assert.Error(t, err)
	assert.Nil(t, writer)
	var conflict *configuration.OutputConflictError
	assert.ErrorAs(t, err, &conflict)
}
