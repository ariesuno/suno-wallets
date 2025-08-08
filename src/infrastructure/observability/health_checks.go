package observability

import (
	"context"
	"time"

	"github.com/redis/go-redis/v9"
	"gorm.io/gorm"
)

// Comentários em pt-BR: funções utilitárias de health-check com timeouts curtos

func CheckDB(ctx context.Context, db *gorm.DB, timeout time.Duration) error {
	cctx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()
	var n int
	return db.WithContext(cctx).Raw("SELECT 1").Scan(&n).Error
}

func CheckRedis(ctx context.Context, rdb *redis.Client, timeout time.Duration) error {
	cctx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()
	return rdb.Ping(cctx).Err()
}
