package cloudscale

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"os"
	"time"

	"github.com/appuio/appuio-cloud-reporting/pkg/db"
	"github.com/cloudscale-ch/cloudscale-go-sdk/v2"
	"github.com/jmoiron/sqlx"
	"github.com/vshn/exoscale-metrics-collector/pkg/categoriesmodel"
	"github.com/vshn/exoscale-metrics-collector/pkg/datetimesmodel"
	"github.com/vshn/exoscale-metrics-collector/pkg/discountsmodel"
	"github.com/vshn/exoscale-metrics-collector/pkg/factsmodel"
	"github.com/vshn/exoscale-metrics-collector/pkg/log"
	"github.com/vshn/exoscale-metrics-collector/pkg/productsmodel"
	"github.com/vshn/exoscale-metrics-collector/pkg/queriesmodel"
	"github.com/vshn/exoscale-metrics-collector/pkg/tenantsmodel"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

var location *time.Location

func init() {
	l, err := time.LoadLocation("Europe/Zurich")
	if err != nil {
		panic(fmt.Errorf("load loaction: %w", err))
	}
	location = l
}

type ObjectStorage struct {
	client      *cloudscale.Client
	k8sClient   client.Client
	date        time.Time
	databaseURL string
}

func NewObjectStorage(client *cloudscale.Client, k8sClient client.Client, days int, databaseURL string) *ObjectStorage {
	now := time.Now().In(location)
	date := time.Date(now.Year(), now.Month(), now.Day()-days, 0, 0, 0, 0, now.Location())

	return &ObjectStorage{
		client:      client,
		k8sClient:   k8sClient,
		date:        date,
		databaseURL: databaseURL,
	}
}

func (obj *ObjectStorage) Execute(ctx context.Context) error {
	logger := log.AppLogger(ctx)

	rdb, err := db.Openx(obj.databaseURL)
	if err != nil {
		return err
	}
	defer rdb.Close()

	// initialize DB
	tx, err := rdb.BeginTxx(ctx, &sql.TxOptions{})
	if err != nil {
		return err
	}
	defer func(tx *sqlx.Tx) {
		err := tx.Rollback()
		if err != nil && !errors.Is(err, sql.ErrTxDone) {
			fmt.Fprintf(os.Stderr, "rollback failed: %v", err)
		}
	}(tx)
	err = initDb(ctx, tx)
	if err != nil {
		return err
	}
	err = tx.Commit()
	if err != nil {
		return err
	}

	accumulated, err := accumulateBucketMetrics(ctx, obj.date, obj.client, obj.k8sClient)
	if err != nil {
		return err
	}

	for source, value := range accumulated {
		if value == 0 {
			continue
		}

		logger.Info("accumulating source", "source", source)

		// start new transaction for actual work
		tx, err = rdb.BeginTxx(ctx, &sql.TxOptions{})
		if err != nil {
			return err
		}

		tenant, err := tenantsmodel.Ensure(ctx, tx, &db.Tenant{Source: source.Tenant})
		if err != nil {
			return err
		}

		category, err := categoriesmodel.Ensure(ctx, tx, &db.Category{Source: source.Zone + ":" + source.Namespace})
		if err != nil {
			return err
		}

		dateTime := datetimesmodel.New(source.Start)
		dateTime, err = datetimesmodel.Ensure(ctx, tx, dateTime)
		if err != nil {
			return err
		}

		product, err := productsmodel.GetBestMatch(ctx, tx, source.String(), source.Start)
		if err != nil {
			return err
		}

		discount, err := discountsmodel.GetBestMatch(ctx, tx, source.String(), source.Start)
		if err != nil {
			return err
		}

		query, err := queriesmodel.GetByName(ctx, tx, source.Query+":"+source.Zone)
		if err != nil {
			return err
		}

		var quantity float64
		if query.Unit == "GB" || query.Unit == "GBDay" {
			quantity = float64(value) / 1000 / 1000 / 1000
		} else if query.Unit == "KReq" {
			quantity = float64(value) / 1000
		} else {
			return errors.New("Unknown query unit " + query.Unit)
		}
		storageFact := factsmodel.New(dateTime, query, tenant, category, product, discount, quantity)
		_, err = factsmodel.Ensure(ctx, tx, storageFact)
		if err != nil {
			return err
		}

		err = tx.Commit()
		if err != nil {
			return err
		}
	}
	return nil
}

func convertUnit(query *db.Query, value uint64) (float64, error) {
	if query.Unit == "GB" || query.Unit == "GBDay" {
		return float64(value) / 1000 / 1000 / 1000, nil
	}
	if query.Unit == "KReq" {
		return float64(value) / 1000, nil
	}
	return 0, errors.New("Unknown query unit " + query.Unit)
}

func initDb(ctx context.Context, tx *sqlx.Tx) error {
	for _, product := range ensureProducts {
		_, err := productsmodel.Ensure(ctx, tx, product)
		if err != nil {
			return err
		}
	}

	for _, discount := range ensureDiscounts {
		_, err := discountsmodel.Ensure(ctx, tx, discount)
		if err != nil {
			return err
		}
	}

	for _, query := range ensureQueries {
		_, err := queriesmodel.Ensure(ctx, tx, query)
		if err != nil {
			return err
		}
	}
	return nil
}
