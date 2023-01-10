package database

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/appuio/appuio-cloud-reporting/pkg/db"
	"github.com/jmoiron/sqlx"
	"github.com/vshn/exoscale-metrics-collector/pkg/reporting"
	ctrl "sigs.k8s.io/controller-runtime"
)

const (
	queryDBaaSPostgres   = string(PostgresDBaaSType) + ":" + provider
	queryDBaaSMysql      = string(MysqlDBaaSType) + ":" + provider
	queryDBaaSOpensearch = string(OpensearchDBaaSType) + ":" + provider
	queryDBaaSRedis      = string(RedisDBaaSType) + ":" + provider
	queryDBaaSKafka      = string(KafkaDBaaSType) + ":" + provider
	defaultUnitDBaaS     = "Instances"
)

// exoscale service types to query billing Database types
var (
	billingTypes = map[string]string{
		"pg":         queryDBaaSPostgres,
		"mysql":      queryDBaaSMysql,
		"opensearch": queryDBaaSOpensearch,
		"redis":      queryDBaaSRedis,
		"kafka":      queryDBaaSKafka,
	}
)

// DBaaSDatabase contains the Database struct needed with the plan, specific of DBaaS
type DBaaSDatabase struct {
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

// Execute starts the saving process of the data in the billing database
func (d *DBaaSDatabase) Execute(ctx context.Context, aggregatedDBaaS map[Key]Aggregated) error {
	log := ctrl.LoggerFrom(ctx)
	s, err := reporting.NewStore(d.URL, log.WithName("reporting-store"))
	if err != nil {
		return fmt.Errorf("reporting.NewStore: %w", err)
	}
	defer func() {
		if err := s.Close(); err != nil {
			log.Error(err, "unable to close")
		}
	}()

	// TODO: split sos/dbaas
	for t, config := range initConfigs {
		if err := s.Initialize(ctx, config.products, []*db.Discount{&config.discount}, []*db.Query{&config.query}); err != nil {
			return fmt.Errorf("init(%q): %w", t, err)
		}
	}

	if err := d.saveUsageToDatabase(ctx, s, aggregatedDBaaS); err != nil {
		return fmt.Errorf("save usage: %w", err)
	}
	return nil
}

// saveUsageToDatabase saves each previously aggregated DBaaS to the billing database
func (d *DBaaSDatabase) saveUsageToDatabase(ctx context.Context, s *reporting.Store, aggregatedDBaaS map[Key]Aggregated) error {
	log := ctrl.LoggerFrom(ctx)
	for _, aggregated := range aggregatedDBaaS {
		err := d.ensureAggregatedServiceUsage(ctx, aggregated, s)
		if err != nil {
			log.Error(err, "Cannot save aggregated DBaaS service record to billing database")
			continue
		}
	}
	return nil
}

// ensureAggregatedServiceUsage saves the aggregated database service usage by namespace and plan to the billing database
// To save the correct data to the database the function also matches a relevant product, discount (if any) and query.
func (d *DBaaSDatabase) ensureAggregatedServiceUsage(ctx context.Context, aggregatedDatabaseService Aggregated, s *reporting.Store) error {
	log := ctrl.LoggerFrom(ctx)
	tokens, err := aggregatedDatabaseService.DecodeKey()
	if err != nil {
		return fmt.Errorf("cannot decode key namespace-plan-dbtype - %s, organization %s, number of instances %f: %w",
			aggregatedDatabaseService.Key,
			aggregatedDatabaseService.Organization,
			aggregatedDatabaseService.Value,
			err)
	}
	namespace := tokens[0]
	plan := tokens[1]
	dbType := tokens[2]

	log.Info("Saving DBaaS usage", "namespace", namespace, "plan", plan, "type", dbType, "quantity", aggregatedDatabaseService.Value)

	sourceString := dbaasSourceString{
		Query:        billingTypes[dbType],
		Organization: aggregatedDatabaseService.Organization,
		Namespace:    namespace,
		Plan:         plan,
	}

	return s.WithTransaction(ctx, func(tx *sqlx.Tx) error {
		tenant, err := reporting.EnsureTenant(ctx, tx, &db.Tenant{Source: aggregatedDatabaseService.Organization})
		if err != nil {
			return fmt.Errorf("cannot ensure organization for namespace %s: %w", namespace, err)
		}

		category, err := reporting.EnsureCategory(ctx, tx, &db.Category{Source: provider + ":" + namespace})
		if err != nil {
			return fmt.Errorf("cannot ensure category for namespace %s: %w", namespace, err)
		}

		dateTime := reporting.NewDateTime(d.BillingDate)
		dateTime, err = reporting.EnsureDateTime(ctx, tx, dateTime)
		if err != nil {
			return fmt.Errorf("cannot ensure date time for namespace %s: %w", namespace, err)
		}

		product, err := reporting.GetBestMatchingProduct(ctx, tx, sourceString.getSourceString(), d.BillingDate)
		if err != nil {
			return fmt.Errorf("cannot get product best match for namespace %s: %w", namespace, err)
		}

		discount, err := reporting.GetBestMatchingDiscount(ctx, tx, sourceString.getSourceString(), d.BillingDate)
		if err != nil {
			return fmt.Errorf("cannot get discount best match for namespace %s: %w", namespace, err)
		}

		query, err := reporting.GetQueryByName(ctx, tx, sourceString.getQuery())
		if err != nil {
			return fmt.Errorf("cannot get query by name for namespace %s: %w", namespace, err)
		}

		storageFact := reporting.NewFact(dateTime, query, tenant, category, product, discount, aggregatedDatabaseService.Value)
		if !isFactsUpdatable(ctx, tx, storageFact, aggregatedDatabaseService.Value) {
			return nil
		}

		_, err = reporting.EnsureFact(ctx, tx, storageFact)
		if err != nil {
			return fmt.Errorf("cannot save fact for namespace %s: %w", namespace, err)
		}
		return nil
	})
}

// isFactsUpdatable makes sure that only missing data or higher quantity values are saved in the billing database
func isFactsUpdatable(ctx context.Context, tx *sqlx.Tx, f *db.Fact, value float64) bool {
	log := ctrl.LoggerFrom(ctx)
	fact, _ := reporting.GetByFact(ctx, tx, f)
	if fact == nil || fact.Quantity < value {
		return true
	}
	log.Info(fmt.Sprintf("Skipped saving, higher or equal number of instances is already recorded in the billing database "+
		"for this hour: saved instance count %.0f, newer instance count %.0f", fact.Quantity, value))
	return false
}

type dbaasSourceString struct {
	Query        string
	Organization string
	Namespace    string
	Plan         string
}

func (ss dbaasSourceString) getQuery() string {
	return ss.Query
}

func (ss dbaasSourceString) getSourceString() string {
	return strings.Join([]string{ss.Query, ss.Organization, ss.Namespace, ss.Plan}, ":")
}
