package db

import (
	"context"
	"fmt"

	"gorm.io/gorm"
)

type txKey struct{}

// TxManager runs functions inside a database transaction.
// The transaction is passed through ctx, and repositories pick it up via Conn,
// so business logic doesn't depend on gorm.
type TxManager struct {
	db *gorm.DB
}

func NewTxManager(db *gorm.DB) (*TxManager, error) {
	if db == nil {
		return nil, fmt.Errorf("db.NewTxManager: %w", fmt.Errorf("nil db"))
	}
	return &TxManager{db: db}, nil
}

// WithinTx runs fn in a transaction: it commits if fn returns nil and rolls back otherwise.
// Nested calls reuse the outer transaction.
func (m *TxManager) WithinTx(ctx context.Context, fn func(ctx context.Context) error) error {
	if _, ok := ctx.Value(txKey{}).(*gorm.DB); ok {
		return fn(ctx)
	}

	err := m.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		return fn(context.WithValue(ctx, txKey{}, tx))
	})
	if err != nil {
		return fmt.Errorf("TxManager.WithinTx: %w", err)
	}

	return nil
}

// Conn returns the transaction stored in ctx, or base if there is none.
func Conn(ctx context.Context, base *gorm.DB) *gorm.DB {
	if tx, ok := ctx.Value(txKey{}).(*gorm.DB); ok {
		return tx.WithContext(ctx)
	}
	return base.WithContext(ctx)
}
