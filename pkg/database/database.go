package database

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	"github.com/appuio/appuio-cloud-reporting/pkg/db"
	"github.com/jmoiron/sqlx"
	"github.com/vshn/exoscale-metrics-collector/pkg/categoriesmodel"
	"github.com/vshn/exoscale-metrics-collector/pkg/datetimesmodel"
	"github.com/vshn/exoscale-metrics-collector/pkg/discountsmodel"
	"github.com/vshn/exoscale-metrics-collector/pkg/factsmodel"
	"github.com/vshn/exoscale-metrics-collector/pkg/productsmodel"
	"github.com/vshn/exoscale-metrics-collector/pkg/queriesmodel"
	"github.com/vshn/exoscale-metrics-collector/pkg/tenantsmodel"
	ctrl "sigs.k8s.io/controller-runtime"
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
	URL          string
	BillingDate  time.Time
	connection   *sqlx.DB
	sourceString SourceString
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

// ensureModels ensures database models are present
func (d *Database) ensureModels(dctx *Context) error {
	namespace := *dctx.namespace
	tenant, err := tenantsmodel.Ensure(dctx, dctx.transaction, &db.Tenant{Source: *dctx.organization})
	if err != nil {
		return fmt.Errorf("cannot ensure organization for namespace %s: %w", namespace, err)
	}
	dctx.tenant = tenant

	category, err := categoriesmodel.Ensure(dctx, dctx.transaction, &db.Category{Source: provider + ":" + namespace})
	if err != nil {
		return fmt.Errorf("cannot ensure category for namespace %s: %w", namespace, err)
	}
	dctx.category = category

	dateTime := datetimesmodel.New(d.BillingDate)
	dateTime, err = datetimesmodel.Ensure(dctx, dctx.transaction, dateTime)
	if err != nil {
		return fmt.Errorf("cannot ensure date time for namespace %s: %w", namespace, err)
	}
	dctx.dateTime = dateTime
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

// commitTransaction commits a transaction in the billing database
func (d *Database) commitTransaction(dctx *Context) error {
	err := dctx.transaction.Commit()
	if err != nil {
		return fmt.Errorf("cannot commit transaction in the database: %w", err)
	}
	return nil
}

// beginTransaction creates a new transaction in the billing database
func (d *Database) beginTransaction(dctx *Context) error {
	tx, err := d.connection.BeginTxx(dctx, &sql.TxOptions{})
	if err != nil {
		return fmt.Errorf("cannot create database transaction: %w", err)
	}
	dctx.transaction = tx
	return nil
}

// rollback rolls back transaction in case of an error in previous steps
func (d *Database) rollback(dctx *Context, err error) error {
	log := ctrl.LoggerFrom(dctx)
	if err != nil {
		log.Error(err, "error found in pipeline")
		e := dctx.transaction.Rollback()
		if e != nil {
			log.Error(e, "cannot rollback transaction")
			return fmt.Errorf("cannot rollback transaction from error: %w", err)
		}
		return fmt.Errorf("error found in pipeline: %w", err)
	}
	return nil
}

// openConnection opens the connection to the billing database
func (d *Database) openConnection(*Context) error {
	connection, err := db.Openx(d.URL)
	if err != nil {
		return fmt.Errorf("cannot create a connection to the database: %w", err)
	}
	d.connection = connection
	return nil
}

// closeConnection closes the connection to the billing database
func (d *Database) closeConnection(*Context) error {
	err := d.connection.Close()
	if err != nil {
		return fmt.Errorf("cannot close database connection: %w", err)
	}
	return nil
}
