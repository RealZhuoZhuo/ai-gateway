package model

import "time"

type GatewayAPIKey struct {
	ID         int64      `gorm:"primaryKey;column:id" json:"id"`
	Name       string     `gorm:"not null;column:name" json:"name"`
	KeyHash    string     `gorm:"uniqueIndex;not null;column:key_hash" json:"key_hash"`
	DisabledAt *time.Time `gorm:"column:disabled_at" json:"disabled_at"`
	ExpiresAt  *time.Time `gorm:"index;column:expires_at" json:"expires_at"`
	CreatedAt  time.Time  `gorm:"column:created_at" json:"created_at"`
}

func (GatewayAPIKey) TableName() string {
	return "gateway_api_keys"
}
