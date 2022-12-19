package database

import (
	"context"
	"fmt"
	"time"

	"github.com/appuio/appuio-cloud-reporting/pkg/db"
	"github.com/jmoiron/sqlx"
	"github.com/vshn/exoscale-metrics-collector/pkg/discountsmodel"
	"github.com/vshn/exoscale-metrics-collector/pkg/factsmodel"
	"github.com/vshn/exoscale-metrics-collector/pkg/productsmodel"
	"github.com/vshn/exoscale-metrics-collector/pkg/queriesmodel"
)

// Context contains necessary data that will be saved in database
type Context struct {
	context.Context
	Aggregated        *Aggregated
	AggregatedObjects *map[Key]Aggregated
	namespace         *string
	organization      *string
	transaction       *sqlx.Tx
	tenant            *db.Tenant
	category          *db.Category
	dateTime          *db.DateTime
	product           *db.Product
	discount          *db.Discount
	query             *db.Query
	value             *float64
}

// Database holds raw url of the postgresql database with the opened connection
type Database struct {
	URL         string
	BillingDate time.Time
}

// SourceString allows to get the full source or query substring
type SourceString interface {
	getSourceString() string
	getQuery() string
}

// ensureInitConfiguration ensures the minimum exoscale service configuration data is present in the database
// before saving service usage
func (d *Database) ensureInitConfiguration(dctx *Context) error {
	for _, config := range initConfigs {
		for _, product := range config.products {
			_, err := productsmodel.Ensure(dctx, dctx.transaction, &product)
			if err != nil {
				return fmt.Errorf("cannot ensure exoscale product model in the billing database: %w", err)
			}
		}
		_, err := discountsmodel.Ensure(dctx, dctx.transaction, &config.discount)
		if err != nil {
			return fmt.Errorf("cannot ensure exoscale discount model in the billing database: %w", err)
		}
		_, err = queriesmodel.Ensure(dctx, dctx.transaction, &config.query)
		if err != nil {
			return fmt.Errorf("cannot ensure exoscale query model in the billing database: %w", err)
		}
	}
	return nil
}

// getBestMatch tries to get the best match for product, discount and query
func (d *Database) getBestMatch(dctx *Context) error {
	namespace := *dctx.namespace
	productMatch, err := productsmodel.GetBestMatch(dctx, dctx.transaction, d.sourceString.getSourceString(), d.BillingDate)
	if err != nil {
		return fmt.Errorf("cannot get product best match for namespace %s: %w", namespace, err)
	}
	dctx.product = productMatch

	discountMatch, err := discountsmodel.GetBestMatch(dctx, dctx.transaction, d.sourceString.getSourceString(), d.BillingDate)
	if err != nil {
		return fmt.Errorf("cannot get discount best match for namespace %s: %w", namespace, err)
	}
	dctx.discount = discountMatch

	queryMatch, err := queriesmodel.GetByName(dctx, dctx.transaction, d.sourceString.getQuery())
	if err != nil {
		return fmt.Errorf("cannot get query by name for namespace %s: %w", namespace, err)
	}
	dctx.query = queryMatch

	return nil
}

func (d *Database) saveFacts(dctx *Context) error {
	storageFact := factsmodel.New(dctx.dateTime, dctx.query, dctx.tenant, dctx.category, dctx.product, dctx.discount, *dctx.value)
	_, err := factsmodel.Ensure(dctx, dctx.transaction, storageFact)
	if err != nil {
		return fmt.Errorf("cannot save fact for namespace %s: %w", *dctx.namespace, err)
	}

	return nil
}
