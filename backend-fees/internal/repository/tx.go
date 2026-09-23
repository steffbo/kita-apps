package repository

import (
	"context"
	"database/sql"
	"fmt"

	"github.com/jmoiron/sqlx"
)

// querier is the subset of *sqlx.DB / *sqlx.Tx the repositories use.
type querier interface {
	ExecContext(ctx context.Context, query string, args ...interface{}) (sql.Result, error)
	GetContext(ctx context.Context, dest interface{}, query string, args ...interface{}) error
	SelectContext(ctx context.Context, dest interface{}, query string, args ...interface{}) error
	QueryContext(ctx context.Context, query string, args ...interface{}) (*sql.Rows, error)
}

type txContextKey struct{}

// conn returns the transaction bound to ctx by TxManager.WithTx, or db.
// Repositories call it for every statement so they transparently join an
// outer transaction.
func conn(ctx context.Context, db *sqlx.DB) querier {
	if tx, ok := ctx.Value(txContextKey{}).(*sqlx.Tx); ok {
		return tx
	}
	return db
}

// TxManager runs service-level units of work atomically.
type TxManager struct {
	db *sqlx.DB
}

// NewTxManager creates a TxManager.
func NewTxManager(db *sqlx.DB) *TxManager {
	return &TxManager{db: db}
}

// WithTx runs fn inside a transaction. Every repository call made with the
// ctx passed to fn uses that transaction. It commits when fn returns nil and
// rolls back on error or panic. Nested calls join the outer transaction.
// A nil TxManager runs fn without a transaction (for unit tests with fakes).
func (m *TxManager) WithTx(ctx context.Context, fn func(ctx context.Context) error) (err error) {
	if m == nil {
		return fn(ctx)
	}
	if _, ok := ctx.Value(txContextKey{}).(*sqlx.Tx); ok {
		return fn(ctx)
	}

	tx, err := m.db.BeginTxx(ctx, nil)
	if err != nil {
		return fmt.Errorf("begin transaction: %w", err)
	}
	defer func() {
		if p := recover(); p != nil {
			_ = tx.Rollback()
			panic(p)
		}
		if err != nil {
			_ = tx.Rollback()
		}
	}()

	if err = fn(context.WithValue(ctx, txContextKey{}, tx)); err != nil {
		return err
	}
	if err = tx.Commit(); err != nil {
		return fmt.Errorf("commit transaction: %w", err)
	}
	return nil
}

// scopedTx is a transaction opened by a repository method. When the method
// runs inside TxManager.WithTx it reuses the outer transaction and Commit /
// Rollback become no-ops, leaving the outcome to the outer unit of work.
type scopedTx struct {
	*sqlx.Tx
	owned bool
}

func (t *scopedTx) Commit() error {
	if !t.owned {
		return nil
	}
	return t.Tx.Commit()
}

func (t *scopedTx) Rollback() error {
	if !t.owned {
		return nil
	}
	return t.Tx.Rollback()
}

// beginTx opens a repository-local transaction or joins the one bound to ctx.
func beginTx(ctx context.Context, db *sqlx.DB) (*scopedTx, error) {
	if tx, ok := ctx.Value(txContextKey{}).(*sqlx.Tx); ok {
		return &scopedTx{Tx: tx}, nil
	}
	tx, err := db.BeginTxx(ctx, nil)
	if err != nil {
		return nil, err
	}
	return &scopedTx{Tx: tx, owned: true}, nil
}
