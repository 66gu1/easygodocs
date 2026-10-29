//go:build testutil

package db

import (
	"context"
	"errors"
	"os"
	"testing"

	"github.com/stretchr/testify/require"
	"gorm.io/gorm"
)

var shared *TestDB

func TestMain(m *testing.M) {
	var stop func()
	shared, stop = StartPostgres()
	code := m.Run()
	stop()
	os.Exit(code)
}

func newTxManager(t *testing.T) (*TxManager, *gorm.DB) {
	t.Helper()
	gdb, _, cleanup := shared.CreateIsolatedDB(t)
	t.Cleanup(cleanup)
	require.NoError(t, gdb.Exec(`CREATE TABLE items (name TEXT NOT NULL)`).Error)
	m, err := NewTxManager(gdb)
	require.NoError(t, err)
	return m, gdb
}

func insertItem(ctx context.Context, base *gorm.DB, name string) error {
	return Conn(ctx, base).Exec(`INSERT INTO items (name) VALUES (?)`, name).Error
}

func countItems(t *testing.T, gdb *gorm.DB) int64 {
	t.Helper()
	var n int64
	require.NoError(t, gdb.Raw(`SELECT COUNT(*) FROM items`).Scan(&n).Error)
	return n
}

func TestTxManager_CommitsOnSuccess(t *testing.T) {
	t.Parallel()
	m, gdb := newTxManager(t)

	err := m.WithinTx(t.Context(), func(ctx context.Context) error {
		return insertItem(ctx, gdb, "a")
	})
	require.NoError(t, err)
	require.EqualValues(t, 1, countItems(t, gdb))
}

func TestTxManager_RollsBackOnError(t *testing.T) {
	t.Parallel()
	m, gdb := newTxManager(t)
	errExp := errors.New("expected")

	err := m.WithinTx(t.Context(), func(ctx context.Context) error {
		require.NoError(t, insertItem(ctx, gdb, "a"))
		return errExp
	})
	require.ErrorIs(t, err, errExp)
	require.EqualValues(t, 0, countItems(t, gdb))
}

func TestTxManager_NestedCallsShareTransaction(t *testing.T) {
	t.Parallel()
	m, gdb := newTxManager(t)
	errExp := errors.New("expected")

	err := m.WithinTx(t.Context(), func(ctx context.Context) error {
		innerErr := m.WithinTx(ctx, func(ctx context.Context) error {
			return insertItem(ctx, gdb, "inner")
		})
		require.NoError(t, innerErr)
		return errExp // the outer rollback must undo the inner insert too
	})
	require.ErrorIs(t, err, errExp)
	require.EqualValues(t, 0, countItems(t, gdb))
}

func TestNewTxManager_NilDB(t *testing.T) {
	t.Parallel()
	_, err := NewTxManager(nil)
	require.Error(t, err)
}
