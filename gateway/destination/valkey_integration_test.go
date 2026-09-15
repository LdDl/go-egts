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
	"github.com/gomodule/redigo/redis"
	"github.com/stretchr/testify/assert"
)

func TestValkeyIntegration(t *testing.T) {
	if os.Getenv("GO_EGTS_VALKEY_INTEGRATION") != "1" {
		t.Skip("Set GO_EGTS_VALKEY_INTEGRATION=1 with the example Valkey infrastructure running")
	}
	cfg, err := configuration.PrepareFileConfiguration("../examples/valkey/gateway.toml")
	assert.NoError(t, err)
	if err != nil {
		return
	}
	for _, settings := range cfg.DestinationsCfg.Valkey {
		t.Run(settings.ID, func(t *testing.T) {
			settings.Database = 3
			settings.StreamName = fmt.Sprintf("egts.test.%d", time.Now().UnixNano())
			connection, err := redis.Dial("tcp", fmt.Sprintf("%s:%d", settings.Host, settings.Port),
				redis.DialUsername(settings.UserName), redis.DialPassword(settings.Password), redis.DialDatabase(settings.Database),
				redis.DialConnectTimeout(time.Second), redis.DialReadTimeout(time.Second), redis.DialWriteTimeout(time.Second))
			assert.NoError(t, err)
			if err != nil {
				return
			}
			t.Cleanup(func() {
				_, err := connection.Do("DEL", settings.StreamName)
				assert.NoError(t, err)
				err = connection.Close()
				assert.NoError(t, err)
			})
			writer, err := PrepareValkey(settings)
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
			ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
			defer cancel()
			_, err = connection.Do("SET", settings.StreamName, "existing value")
			assert.NoError(t, err)
			err = writer.WriteContext(ctx, record)
			assert.ErrorContains(t, err, "WRONGTYPE")
			assert.Nil(t, writer.connection)
			reply, err := connection.Do("GET", settings.StreamName)
			assert.NoError(t, err)
			value, err := redis.String(reply, nil)
			assert.NoError(t, err)
			assert.Equal(t, "existing value", value)
			_, err = connection.Do("DEL", settings.StreamName)
			assert.NoError(t, err)
			for attempt := 0; attempt < 2; attempt++ {
				if attempt == 1 {
					err = writer.connection.Close()
					assert.NoError(t, err)
				}
				err = writer.WriteContext(ctx, record)
				assert.NoError(t, err)
				if err != nil {
					return
				}
			}
			reply, err = connection.Do("XRANGE", settings.StreamName, "-", "+")
			assert.NoError(t, err)
			entries, err := redis.Values(reply, nil)
			assert.NoError(t, err)
			assert.Len(t, entries, 2)
			var ids []string
			for _, entry := range entries {
				values, err := redis.Values(entry, nil)
				assert.NoError(t, err)
				var id string
				var fields []interface{}
				_, err = redis.Scan(values, &id, &fields)
				assert.NoError(t, err)
				data, err := redis.StringMap(fields, nil)
				assert.NoError(t, err)
				assert.Equal(t, map[string]string{"message_id": hex.EncodeToString(digest[:]), "payload": string(body)}, data)
				ids = append(ids, id)
			}
			if len(ids) == 2 {
				assert.NotEqual(t, ids[0], ids[1])
			}
			_, err = connection.Do("SELECT", 0)
			assert.NoError(t, err)
			reply, err = connection.Do("EXISTS", settings.StreamName)
			assert.NoError(t, err)
			assert.Equal(t, int64(0), reply)
			_, err = connection.Do("SELECT", settings.Database)
			assert.NoError(t, err)
			settings.Password = "incorrect-test-password"
			wrongCredentials, err := PrepareValkey(settings)
			assert.NoError(t, err)
			err = wrongCredentials.WriteContext(ctx, record)
			assert.Error(t, err)
			if err != nil {
				assert.NotContains(t, err.Error(), settings.Password)
			}
			err = wrongCredentials.Close()
			assert.NoError(t, err)
		})
	}
}
