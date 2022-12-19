package database

import (
	"context"
	"fmt"
	"strings"

	"github.com/appuio/appuio-cloud-reporting/pkg/db"
	"github.com/jmoiron/sqlx"
	"github.com/vshn/exoscale-metrics-collector/pkg/categoriesmodel"
	"github.com/vshn/exoscale-metrics-collector/pkg/datetimesmodel"
	"github.com/vshn/exoscale-metrics-collector/pkg/discountsmodel"
	"github.com/vshn/exoscale-metrics-collector/pkg/factsmodel"
	"github.com/vshn/exoscale-metrics-collector/pkg/productsmodel"
	"github.com/vshn/exoscale-metrics-collector/pkg/queriesmodel"
	"github.com/vshn/exoscale-metrics-collector/pkg/reporting"
	"github.com/vshn/exoscale-metrics-collector/pkg/tenantsmodel"
	ctrl "sigs.k8s.io/controller-runtime"
)

const (
	// SosType represents object storage storage type
	SosType        ObjectType = "object-storage-storage"
	querySos                  = string(SosType) + ":" + provider
	defaultUnitSos            = "GBDay"
)

// SosDatabase contains the Database struct needed
type SosDatabase struct {
	Database
}

// Execute starts the saving process of the data in the billing database
func (s *SosDatabase) Execute(ctx context.Context) error {
	log := ctrl.LoggerFrom(ctx)
	store, err := reporting.NewStore(s.URL, log.WithName("reporting-store"))
	if err != nil {
		return fmt.Errorf("reporting.NewStore: %w", err)
	}
	defer func() {
		if err := store.Close(); err != nil {
			log.Error(err, "unable to close")
		}
	}()

	// TODO: split sos/dbaas
	for t, config := range initConfigs {
		if err := store.Initialize(ctx, config.products, []*db.Discount{&config.discount}, []*db.Query{&config.query}); err != nil {
			return fmt.Errorf("init(%q): %w", t, err)
		}
	}

	if err := s.saveUsageToDatabase(ctx, store, aggregatedDBaaS); err != nil {
		return fmt.Errorf("save usage: %w", err)
	}
	return nil
}

// saveUsageToDatabase saves each previously aggregated buckets to the billing database
func (s *SosDatabase) saveUsageToDatabase(ctx context.Context, store *reporting.Store, aggregatedObjects map[Key]Aggregated) error {
	log := ctrl.LoggerFrom(ctx)
	for _, aggregated := range aggregatedObjects {
		err := s.ensureBucketUsage(ctx, store, aggregated)
		if err != nil {
			log.Error(err, "cannot save aggregated buckets service record to billing database")
			continue
		}
	}
	return nil
}

// ensureBucketUsage saves the aggregated buckets usage by namespace to the billing database
// To save the correct data to the database the function also matches a relevant product, discount (if any) and query.
// The storage usage is referred to a day before the application ran (yesterday)
func (s *SosDatabase) ensureBucketUsage(ctx context.Context, store *reporting.Store, aggregatedBucket Aggregated) error {
	log := ctrl.LoggerFrom(ctx)

	tokens, err := aggregatedBucket.DecodeKey()
	if err != nil {
		return fmt.Errorf("cannot decode key namespace-plan-dbtype - %s, organization %s, number of instances %f: %w",
			aggregatedBucket.Key,
			aggregatedBucket.Organization,
			aggregatedBucket.Value,
			err)
	}
	namespace := tokens[0]

	log.Info("Saving buckets usage for namespace", "namespace", namespace, "storage used", aggregatedBucket.Value)
	organization := aggregatedBucket.Organization
	value := aggregatedBucket.Value

	sourceString := sosSourceString{
		ObjectType: SosType,
		provider:   provider,
	}

	return store.WithTransaction(ctx, func(tx *sqlx.Tx) error {
		tenant, err := tenantsmodel.Ensure(ctx, tx, &db.Tenant{Source: organization})
		if err != nil {
			return fmt.Errorf("cannot ensure organization for namespace %s: %w", namespace, err)
		}

		category, err := categoriesmodel.Ensure(ctx, tx, &db.Category{Source: provider + ":" + namespace})
		if err != nil {
			return fmt.Errorf("cannot ensure category for namespace %s: %w", namespace, err)
		}

		dateTime := datetimesmodel.New(s.BillingDate)
		dateTime, err = datetimesmodel.Ensure(ctx, tx, dateTime)
		if err != nil {
			return fmt.Errorf("cannot ensure date time for namespace %s: %w", namespace, err)
		}

		product, err := productsmodel.GetBestMatch(ctx, tx, sourceString.getSourceString(), s.BillingDate)
		if err != nil {
			return fmt.Errorf("cannot get product best match for namespace %s: %w", namespace, err)
		}

		discount, err := discountsmodel.GetBestMatch(ctx, tx, sourceString.getSourceString(), s.BillingDate)
		if err != nil {
			return fmt.Errorf("cannot get discount best match for namespace %s: %w", namespace, err)
		}

		query, err := queriesmodel.GetByName(ctx, tx, sourceString.getQuery())
		if err != nil {
			return fmt.Errorf("cannot get query by name for namespace %s: %w", namespace, err)
		}

		quantity, err := adjustStorageSizeUnit(value)
		if err != nil {
			return fmt.Errorf("adjustStorageSizeUnit: %w", err)
		}

		storageFact := factsmodel.New(dateTime, query, tenant, category, product, discount, value)
		_, err = factsmodel.Ensure(ctx, tx, storageFact)
		if err != nil {
			return fmt.Errorf("cannot save fact for namespace %s: %w", namespace, err)
		}
	})
}

func adjustStorageSizeUnit(value float64) (float64, error) {
	sosUnit := initConfigs[SosType].query.Unit
	if sosUnit == defaultUnitSos {
		return value / 1024 / 1024 / 1024, nil
	}
	return 0, fmt.Errorf("unknown query unit %s", sosUnit)
}

type sosSourceString struct {
	ObjectType
	provider string
}

func (ss sosSourceString) getQuery() string {
	return strings.Join([]string{string(ss.ObjectType), ss.provider}, ":")
}

func (ss sosSourceString) getSourceString() string {
	return strings.Join([]string{string(ss.ObjectType), ss.provider}, ":")
}
