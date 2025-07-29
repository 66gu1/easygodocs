package tx

import (
	"context"
	"gorm.io/gorm"
)

type Transaction interface {
	Transaction(fc func(tx Transaction) error) (err error)
	GetDB(ctx context.Context) *gorm.DB
}

type gormTx struct {
	db *gorm.DB
}

func New(db *gorm.DB) Transaction {
	return &gormTx{db: db}
}

func (t *gormTx) Transaction(fc func(tx Transaction) error) error {
	return t.db.Transaction(func(tx *gorm.DB) error {
		return fc(&gormTx{db: tx})
	})
}

func (t *gormTx) GetDB(ctx context.Context) *gorm.DB {
	return t.db.WithContext(ctx)
}
