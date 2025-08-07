package infrastructure_test

import (
	"context"
	"os"
	"testing"
	"time"

	"suno-wallets/src/infrastructure/data"
	"suno-wallets/src/shared/config"

	"github.com/stretchr/testify/require"
)

// TestRedisConnection verifica ping e set/get básicos
func TestRedisConnection(t *testing.T) {
	password := os.Getenv("REDIS_PASSWORD")
	if password == "" {
		password = "suno_redis_password"
	}
	host := os.Getenv("REDIS_HOST")
	if host == "" {
		host = "localhost"
	}
	port := os.Getenv("REDIS_PORT")
	if port == "" {
		port = "6379"
	}
	cfg := &config.Config{
		RedisHost:     host,
		RedisPort:     port,
		RedisPassword: password,
		RedisDB:       0,
	}
	client := data.NewRedisClient(cfg)
	t.Cleanup(func() { _ = client.Close() })

	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	require.NoError(t, data.PingRedis(ctx, client))

	// Operação simples de set/get
	require.NoError(t, client.Set(ctx, "it:test", "ok", time.Second*5).Err())
	val, err := client.Get(ctx, "it:test").Result()
	require.NoError(t, err)
	require.Equal(t, "ok", val)
}
