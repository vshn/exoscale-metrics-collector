package cloudscale

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/appuio/appuio-cloud-reporting/pkg/db"
	"github.com/cloudscale-ch/cloudscale-go-sdk/v2"
	"github.com/jmoiron/sqlx"
	"github.com/vshn/exoscale-metrics-collector/pkg/reporting"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

type ObjectStorage struct {
	client      *cloudscale.Client
	k8sClient   client.Client
	date        time.Time
	databaseURL string
}

func NewObjectStorage(client *cloudscale.Client, k8sClient client.Client, days int, databaseURL string) (*ObjectStorage, error) {
	location, err := time.LoadLocation("Europe/Zurich")
	if err != nil {
		return nil, fmt.Errorf("load loaction: %w", err)
	}
	now := time.Now().In(location)
	date := time.Date(now.Year(), now.Month(), now.Day()-days, 0, 0, 0, 0, now.Location())

	return &ObjectStorage{
		client:      client,
		k8sClient:   k8sClient,
		date:        date,
		databaseURL: databaseURL,
	}, nil
}

func (obj *ObjectStorage) Execute(ctx context.Context) error {
	logger := ctrl.LoggerFrom(ctx)
	s, err := reporting.NewStore(obj.databaseURL, logger.WithName("reporting-store"))
	if err != nil {
		return fmt.Errorf("reporting.NewStore: %w", err)
	}
	defer func() {
		if err := s.Close(); err != nil {
			logger.Error(err, "unable to close")
		}
	}()

	if err := s.Initialize(ctx, ensureProducts, ensureDiscounts, ensureQueries); err != nil {
		return fmt.Errorf("init: %w", err)
	}

	return s.WithTransaction(ctx, func(tx *sqlx.Tx) error {
		accumulated, err := accumulateBucketMetrics(ctx, obj.date, obj.client, obj.k8sClient)
		if err != nil {
			return err
		}

		for source, value := range accumulated {
			if value == 0 {
				continue
			}

			logger.Info("accumulating source", "source", source)

			tenant, err := reporting.EnsureTenant(ctx, tx, &db.Tenant{Source: source.Tenant})
			if err != nil {
				return err
			}

			category, err := reporting.EnsureCategory(ctx, tx, &db.Category{Source: source.Zone + ":" + source.Namespace})
			if err != nil {
				return err
			}

			dateTime := reporting.NewDateTime(source.Start)
			dateTime, err = reporting.EnsureDateTime(ctx, tx, dateTime)
			if err != nil {
				return err
			}

			product, err := reporting.GetBestMatchingProduct(ctx, tx, source.String(), source.Start)
			if err != nil {
				return err
			}

			discount, err := reporting.GetBestMatchingDiscount(ctx, tx, source.String(), source.Start)
			if err != nil {
				return err
			}

			query, err := reporting.GetQueryByName(ctx, tx, source.Query+":"+source.Zone)
			if err != nil {
				return err
			}

			quantity, err := convertUnit(query, value)
			if err != nil {
				return fmt.Errorf("convertUnit(%q): %w", value, err)
			}
			storageFact := reporting.NewFact(dateTime, query, tenant, category, product, discount, quantity)
			_, err = reporting.EnsureFact(ctx, tx, storageFact)
			if err != nil {
				return err
			}

			err = tx.Commit()
			if err != nil {
				return err
			}
		}
		return nil
	})
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
