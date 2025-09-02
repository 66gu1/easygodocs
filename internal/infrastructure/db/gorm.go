package db

import (
	"time"

	"gorm.io/gorm"
)

type Base struct {
	CreatedAt time.Time // autoCreateTime
	UpdatedAt time.Time // autoUpdateTime
	DeletedAt gorm.DeletedAt
}
