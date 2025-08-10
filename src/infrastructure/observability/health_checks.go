package observability

import (
	"context"
	"errors"
	"time"

	"github.com/redis/go-redis/v9"
	"gorm.io/gorm"

	"suno-wallets/src/domain/entities"
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

// CheckMigrations verifica se as migrações mínimas foram aplicadas.
// Aqui validamos a existência da tabela wallets via Migrator (somente leitura).
func CheckMigrations(ctx context.Context, db *gorm.DB, timeout time.Duration) error {
	cctx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()
	if db.WithContext(cctx).Migrator().HasTable(&entities.Wallet{}) {
		return nil
	}
	return errors.New("migrations_not_applied")
}
