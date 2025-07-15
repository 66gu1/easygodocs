package db

import (
	"gorm.io/gorm"
	"time"
)

type Base struct {
	CreatedAt time.Time // autoCreateTime
	UpdatedAt time.Time // autoUpdateTime
	DeletedAt gorm.DeletedAt
}
