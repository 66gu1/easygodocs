package db

import (
	"time"

	"gorm.io/gorm"
)

const DuplicateCode = "23505"

type Base struct {
	CreatedAt time.Time // autoCreateTime
	UpdatedAt time.Time // autoUpdateTime
	DeletedAt gorm.DeletedAt
}
