package repo

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"time"

	"github.com/RealZhuoZhuo/ai-gateway/internal/repo/model"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

var ErrNotConfigured = errors.New("repo not configured")

type Postgres struct {
	db *gorm.DB
}

func NewPostgres(ctx context.Context,
	databaseURL string) (*Postgres, error) {
	if databaseURL == "" {
		return nil, ErrNotConfigured
	}
	db, err := gorm.Open(postgres.Open(databaseURL), &gorm.Config{})
	if err != nil {
		return nil, err
	}
	sqlDB, err := db.DB()
	if err != nil {
		return nil, err
	}
	sqlDB.SetMaxOpenConns(10)
	sqlDB.SetMaxIdleConns(5)
	sqlDB.SetConnMaxLifetime(time.Hour)
	if err := sqlDB.PingContext(ctx); err != nil {
		_ = sqlDB.Close()
		return nil, err
	}
	if err := db.WithContext(ctx).AutoMigrate(&model.GatewayAPIKey{}); err != nil {
		_ = sqlDB.Close()
		return nil, err
	}
	return &Postgres{db: db}, nil
}

func (r *Postgres) Close() {
	if r == nil || r.db == nil {
		return
	}
	sqlDB, err := r.db.DB()
	if err == nil {
		_ = sqlDB.Close()
	}
}

func (r *Postgres) ValidAPIKey(ctx context.Context, token string) (bool, error) {
	if r == nil || r.db == nil {
		return false, ErrNotConfigured
	}

	hash := sha256.Sum256([]byte(token))
	keyHash := hex.EncodeToString(hash[:])

	var count int64
	err := r.db.WithContext(ctx).
		Model(&model.GatewayAPIKey{}).
		Where("key_hash = ?", keyHash).
		Where("disabled_at is null").
		Where("(expires_at is null or expires_at > now())").
		Count(&count).Error
	return count > 0, err
}
