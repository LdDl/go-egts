package destination

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"net"
	"strconv"
	"testing"
	"time"

	"github.com/LdDl/go-egts/gateway/configuration"
	"github.com/gomodule/redigo/redis"
	"github.com/stretchr/testify/assert"
)

type valkeyProtocolTestCase struct {
	name      string
	username  string
	password  string
	database  int
	failureAt string
	reply     string
	errorText string
}

func TestValkeyProtocol(t *testing.T) {
	cases := []valkeyProtocolTestCase{
		{name: "no authentication"},
		{name: "password only", password: "secret"},
		{name: "named user", username: "operator", password: "secret", database: 2},
		{name: "explicit default user", username: "default", password: "secret"},
		{name: "special characters", username: "operator", password: "spaces $ \" \\ \r\n", database: 1},
		{name: "AUTH rejected", password: "secret", failureAt: "AUTH", reply: "-WRONGPASS invalid credentials\r\n", errorText: "WRONGPASS"},
		{name: "SELECT rejected", database: 17, failureAt: "SELECT", reply: "-ERR DB index is out of range\r\n", errorText: "DB index"},
		{name: "wrong key type", failureAt: "XADD", reply: "-WRONGTYPE key is not a stream\r\n", errorText: "WRONGTYPE"},
		{name: "out of memory", failureAt: "XADD", reply: "-OOM command not allowed\r\n", errorText: "OOM"},
		{name: "unsupported server", failureAt: "XADD", reply: "-ERR unknown command XADD\r\n", errorText: "unknown command"},
		{name: "null ID", failureAt: "XADD", reply: "$-1\r\n", errorText: "Invalid Valkey"},
		{name: "integer ID", failureAt: "XADD", reply: ":1\r\n", errorText: "Invalid Valkey"},
		{name: "unexpected OK", failureAt: "XADD", reply: "+OK\r\n", errorText: "Invalid Valkey"},
		{name: "invalid time", failureAt: "XADD", reply: "+wrong-1\r\n", errorText: "entry time"},
		{name: "invalid sequence", failureAt: "XADD", reply: "+1-wrong\r\n", errorText: "entry sequence"},
		{name: "zero ID", failureAt: "XADD", reply: "+0-0\r\n", errorText: "zero Valkey"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			listener, err := net.Listen("tcp", "127.0.0.1:0")
			assert.NoError(t, err)
			if err != nil {
				return
			}
			t.Cleanup(func() {
				err := listener.Close()
				assert.NoError(t, err)
			})
			cfg := configuration.DefaultValkeyDestination()
			cfg.ID = "primary"
			cfg.Enabled = true
			cfg.Port = listener.Addr().(*net.TCPAddr).Port
			cfg.UserName = tc.username
			cfg.Password = tc.password
			cfg.Database = tc.database
			writer, err := PrepareValkey(cfg)
			assert.NoError(t, err)
			if err != nil {
				return
			}
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
			ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
			defer cancel()
			var peer net.Conn
			var reader redis.Conn
			for attempt := 0; attempt < 2; attempt++ {
				done := make(chan error, 1)
				go func() {
					done <- writer.WriteContext(ctx, record)
				}()
				var commands [][]string
				if attempt == 0 || tc.errorText != "" {
					err = listener.(*net.TCPListener).SetDeadline(time.Now().Add(2 * time.Second))
					assert.NoError(t, err)
					accepted, err := listener.Accept()
					assert.NoError(t, err)
					if err != nil {
						return
					}
					t.Cleanup(func() {
						err := accepted.Close()
						assert.NoError(t, err)
					})
					peer = accepted
					reader = redis.NewConn(peer, 2*time.Second, 2*time.Second)
					if tc.password != "" {
						auth := []string{"AUTH"}
						if tc.username != "" {
							auth = append(auth, tc.username)
						}
						auth = append(auth, tc.password)
						commands = append(commands, auth)
					}
					if tc.database != 0 {
						commands = append(commands, []string{"SELECT", strconv.Itoa(tc.database)})
					}
				}
				commands = append(commands, []string{"XADD", cfg.StreamName, "*", "message_id", hex.EncodeToString(digest[:]), "payload", string(body)})
				for _, expected := range commands {
					request, err := reader.Receive()
					assert.NoError(t, err)
					if err != nil {
						return
					}
					actual, err := redis.Strings(request, nil)
					assert.NoError(t, err)
					assert.Equal(t, expected, actual)
					reply := "+OK\r\n"
					if expected[0] == "XADD" {
						reply = "$5\r\n123-1\r\n"
					}
					if attempt == 0 && expected[0] == tc.failureAt {
						reply = tc.reply
					}
					_, err = peer.Write([]byte(reply))
					assert.NoError(t, err)
					if attempt == 0 && expected[0] == tc.failureAt {
						break
					}
				}
				err = <-done
				if attempt == 0 && tc.errorText != "" {
					assert.ErrorContains(t, err, tc.errorText)
					assert.Nil(t, writer.connection)
				} else {
					assert.NoError(t, err)
				}
			}
			err = writer.WriteContext(ctx, nil)
			assert.ErrorContains(t, err, "nil")
			err = writer.Close()
			assert.NoError(t, err)
			err = writer.WriteContext(ctx, record)
			assert.ErrorIs(t, err, net.ErrClosed)
		})
	}
}

func TestValkeyCancellation(t *testing.T) {
	for _, stage := range []string{"connect", "write"} {
		for _, mode := range []string{"cancel", "timeout"} {
			t.Run(stage+"/"+mode, func(t *testing.T) {
				listener, err := net.Listen("tcp", "127.0.0.1:0")
				assert.NoError(t, err)
				if err != nil {
					return
				}
				t.Cleanup(func() {
					err := listener.Close()
					assert.NoError(t, err)
				})
				cfg := configuration.DefaultValkeyDestination()
				cfg.ID = "primary"
				cfg.Enabled = true
				cfg.Port = listener.Addr().(*net.TCPAddr).Port
				cfg.ConnectTimeoutSeconds = 1
				cfg.WriteTimeoutSeconds = 1
				if stage == "connect" {
					cfg.Password = "secret"
				}
				writer, err := PrepareValkey(cfg)
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
				reader := redis.NewConn(peer, 2*time.Second, 2*time.Second)
				request, err := reader.Receive()
				assert.NoError(t, err)
				command, err := redis.Strings(request, nil)
				assert.NoError(t, err)
				if stage == "connect" {
					assert.Equal(t, []string{"AUTH", "secret"}, command)
				} else {
					assert.Equal(t, "XADD", command[0])
				}
				if mode == "cancel" {
					cancel()
				}
				select {
				case err = <-done:
					if mode == "cancel" {
						assert.ErrorIs(t, err, context.Canceled)
					} else {
						var timeout net.Error
						assert.ErrorAs(t, err, &timeout)
						if timeout != nil {
							assert.True(t, timeout.Timeout())
						}
					}
				case <-time.After(2 * time.Second):
					t.Fatal("Valkey operation did not stop")
				}
				assert.Nil(t, writer.connection)
			})
		}
	}
}
