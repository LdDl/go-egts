package destination

import (
	"bytes"
	"context"
	"encoding/hex"
	"net"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/LdDl/go-egts/gateway/configuration"
	"github.com/stretchr/testify/assert"
)

func TestStdoutWriteCancellation(t *testing.T) {
	reader, output, err := os.Pipe()
	assert.NoError(t, err)
	if err != nil {
		return
	}
	t.Cleanup(func() {
		err := reader.Close()
		assert.NoError(t, err)
		err = output.Close()
		assert.NoError(t, err)
	})
	cfg := configuration.DefaultConfiguration()
	cfg.DeliveryCfg.DumpDirectory = filepath.Join(t.TempDir(), "queue")
	previousStdout := os.Stdout
	os.Stdout = output
	writer, err := PrepareStdout(cfg)
	os.Stdout = previousStdout
	assert.NoError(t, err)
	if err != nil {
		return
	}
	t.Cleanup(func() {
		err := writer.Close()
		assert.NoError(t, err)
	})
	if !writer.deadline {
		t.Skip("Pipe write deadlines are unavailable")
	}
	err = writer.stream.SetWriteDeadline(time.Now().Add(20 * time.Millisecond))
	assert.NoError(t, err)
	_, err = writer.stream.Write(bytes.Repeat([]byte("x"), 4*1024*1024))
	assert.ErrorIs(t, err, os.ErrDeadlineExceeded)
	raw, err := hex.DecodeString("0100030b001000000000b3000000060000005802020003000000002ec1")
	assert.NoError(t, err)
	record, err := NewRecord(time.Now(), Source{}, raw)
	assert.NoError(t, err)
	if err != nil {
		return
	}
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	timer := time.AfterFunc(50*time.Millisecond, cancel)
	defer timer.Stop()
	err = writer.WriteContext(ctx, record)
	assert.ErrorIs(t, err, context.Canceled)
	ctx, cancelDeadline := context.WithTimeout(context.Background(), 50*time.Millisecond)
	defer cancelDeadline()
	err = writer.WriteContext(ctx, record)
	assert.Error(t, err)
	var timeout net.Error
	assert.ErrorAs(t, err, &timeout)
	if timeout != nil {
		assert.True(t, timeout.Timeout())
	}
}
