package data

import (
	"context"
	"time"

	"suno-wallets/src/shared/config"

	"github.com/redis/go-redis/v9"
)

// NewRedisClient cria um cliente Redis com timeouts configurados
func NewRedisClient(cfg *config.Config) *redis.Client {
	return redis.NewClient(&redis.Options{
		Addr:         cfg.RedisHost + ":" + cfg.RedisPort,
		Password:     cfg.RedisPassword,
		DB:           cfg.RedisDB,
		DialTimeout:  5 * time.Second,
		ReadTimeout:  3 * time.Second,
		WriteTimeout: 3 * time.Second,
		PoolTimeout:  4 * time.Second,
		MinIdleConns: 2,
		MaxRetries:   2,
	})
}

// PingRedis valida a conexão com o Redis usando contexto com timeout
func PingRedis(ctx context.Context, client *redis.Client) error {
	cctx, cancel := context.WithTimeout(ctx, 2*time.Second)
	defer cancel()
	return client.Ping(cctx).Err()
}
