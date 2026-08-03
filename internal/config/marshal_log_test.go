package config

import (
	"testing"

	"github.com/stretchr/testify/require"
	"go.uber.org/zap/zapcore"
)

func TestMarshalLogObject(t *testing.T) {
	c := &C{
		AppEnv:           "test",
		HTTPAddr:         ":8080",
		MySQLDSN:         "user:pass@tcp(db:3306)/app",
		RedisAddr:        "redis:6379",
		RedisDB:          1,
		RedisPoolSize:    10,
		RabbitMQURL:      "amqp://guest:guest@mq:5672/",
		ElasticsearchURL: "http://es:9200",
	}
	enc := zapcore.NewMapObjectEncoder()
	require.NoError(t, c.MarshalLogObject(enc))
	require.Equal(t, "test", enc.Fields["app_env"])
	require.Equal(t, ":8080", enc.Fields["http_addr"])
	require.Equal(t, "redis:6379", enc.Fields["redis_addr"])
	require.Equal(t, 10, enc.Fields["redis_pool_size"])
}

func TestMarshalLogObject_Empty(t *testing.T) {
	enc := zapcore.NewMapObjectEncoder()
	require.NoError(t, (&C{}).MarshalLogObject(enc))
	require.NotNil(t, enc.Fields)
}
