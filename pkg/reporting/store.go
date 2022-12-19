package reporting

import (
	"context"
	"database/sql"
	"errors"
	"fmt"

	"github.com/appuio/appuio-cloud-reporting/pkg/db"
	"github.com/go-logr/logr"
	"github.com/jmoiron/sqlx"
	"github.com/vshn/exoscale-metrics-collector/pkg/discountsmodel"
	"github.com/vshn/exoscale-metrics-collector/pkg/productsmodel"
	"github.com/vshn/exoscale-metrics-collector/pkg/queriesmodel"
)

type Store struct {
	db     *sqlx.DB
	logger logr.Logger
}

// NewStore opens up a db connection to the specified db.
func NewStore(url string, logger logr.Logger) (*Store, error) {
	rdb, err := db.Openx(url)
	if err != nil {
		return nil, fmt.Errorf("newReporting: open db failed: %w", err)
	}

	return &Store{
		db:     rdb,
		logger: logger,
	}, nil
}

// Close the db connection.
func (r *Store) Close() error {
	return r.db.Close()
}

// WithTransaction runs fn within a transaction and does commit/rollback when necessary.
func (r *Store) WithTransaction(ctx context.Context, fn func(*sqlx.Tx) error) error {
	tx, err := r.db.BeginTxx(ctx, &sql.TxOptions{})
	if err != nil {
		return fmt.Errorf("beginTransaction: transaction failed: %w", err)
	}

	defer func() {
		if p := recover(); p != nil {
			// a panic occurred, rollback and repanic
			err := tx.Rollback()
			if err != nil && !errors.Is(err, sql.ErrTxDone) {
				r.logger.Error(err, "unable to rollback after panic")
			}
			panic(p)
		} else if err != nil {
			// something went wrong, rollback
			err := tx.Rollback()
			if err != nil && !errors.Is(err, sql.ErrTxDone) {
				r.logger.Error(err, "unable to rollback after error")
			}
		}
	}()

	if err := fn(tx); err != nil {
		return err
	}

	if err := tx.Commit(); err != nil {
		return fmt.Errorf("tx: unable to commit: %w", err)
	}
	return nil
}

// Initialize uses a transaction to ensure given entities in reporting store.
func (r *Store) Initialize(ctx context.Context, products []*db.Product, discounts []*db.Discount, queries []*db.Query) error {
	err := r.WithTransaction(ctx, func(tx *sqlx.Tx) error {
		for _, product := range products {
			_, err := productsmodel.Ensure(ctx, tx, product)
			if err != nil {
				return fmt.Errorf("product ensure: %w", err)
			}
		}
		for _, discount := range discounts {
			_, err := discountsmodel.Ensure(ctx, tx, discount)
			if err != nil {
				return fmt.Errorf("discount ensure: %w", err)
			}
		}
		for _, query := range queries {
			_, err := queriesmodel.Ensure(ctx, tx, query)
			if err != nil {
				return fmt.Errorf("query ensure: %w", err)
			}
		}
		return nil
	})
	if err != nil {
		return fmt.Errorf("initialize: %w", err)
	}
	return nil
}
