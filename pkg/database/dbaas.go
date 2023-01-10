package database

import (
	"fmt"
	"strings"

	"github.com/appuio/appuio-cloud-reporting/pkg/db"
	"github.com/vshn/exoscale-metrics-collector/pkg/factsmodel"

	pipeline "github.com/ccremer/go-command-pipeline"
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
	Database
	plan *string
}

// Execute starts the saving process of the data in the billing database
func (d *DBaaSDatabase) Execute(dctx *Context) error {
	p := pipeline.NewPipeline[*Context]()
	p.WithSteps(
		p.NewStep("Open database connection", d.openConnection),
		p.WithNestedSteps("Save initial billing configuration", nil,
			p.NewStep("Begin transaction", d.beginTransaction),
			p.NewStep("Ensure initial billing database configuration", d.ensureInitConfiguration),
			p.NewStep("Commit transaction", d.commitTransaction),
		).WithErrorHandler(d.rollback),
		p.NewStep("Save DBaaS usage to billing database", d.saveUsageToDatabase),
		p.NewStep("Close database connection", d.closeConnection),
	)
	return p.RunWithContext(dctx)
}

// saveUsageToDatabase saves each previously aggregated DBaaS to the billing database
func (d *DBaaSDatabase) saveUsageToDatabase(dctx *Context) error {
	log := ctrl.LoggerFrom(dctx)
	for _, aggregated := range *dctx.AggregatedObjects {
		err := d.ensureAggregatedServiceUsage(dctx, aggregated)
		if err != nil {
			log.Error(err, "Cannot save aggregated DBaaS service record to billing database")
			continue
		}
	}
	return nil
}

// ensureAggregatedServiceUsage saves the aggregated database service usage by namespace and plan to the billing database
// To save the correct data to the database the function also matches a relevant product, discount (if any) and query.
func (d *DBaaSDatabase) ensureAggregatedServiceUsage(dctx *Context, aggregatedDatabaseService Aggregated) error {
	log := ctrl.LoggerFrom(dctx)
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

	dctx.organization = &aggregatedDatabaseService.Organization
	dctx.namespace = &namespace
	dctx.value = &aggregatedDatabaseService.Value
	dctx.Aggregated = &aggregatedDatabaseService
	d.plan = &plan

	d.sourceString = dbaasSourceString{
		Query:        billingTypes[dbType],
		Organization: *dctx.organization,
		Namespace:    namespace,
		Plan:         plan,
	}

	p := pipeline.NewPipeline[*Context]()
	p.WithSteps(
		p.WithNestedSteps(fmt.Sprintf("Saving DBaaS usage namespace %s", namespace), nil,
			p.NewStep("Begin database transaction", d.beginTransaction),
			p.NewStep("Ensure necessary models", d.ensureModels),
			p.NewStep("Get best match", d.getBestMatch),
			p.When(isFactsUpdatable, "Save to billing database", d.saveFacts),
			p.NewStep("Commit transaction", d.commitTransaction),
		).WithErrorHandler(d.rollback),
	)

	return p.RunWithContext(dctx)
}

// isFactsUpdatable makes sure that only missing data or higher quantity values are saved in the billing database
func isFactsUpdatable(dctx *Context) bool {
	log := ctrl.LoggerFrom(dctx)
	fact, _ := factsmodel.GetByFact(dctx, dctx.transaction, &db.Fact{
		DateTimeId: dctx.dateTime.Id,
		QueryId:    dctx.query.Id,
		TenantId:   dctx.tenant.Id,
		CategoryId: dctx.category.Id,
		ProductId:  dctx.product.Id,
		DiscountId: dctx.discount.Id,
	})
	if fact == nil || fact.Quantity < *dctx.value {
		return true
	}
	log.Info(fmt.Sprintf("Skipped saving, higher or equal number of instances is already recorded in the billing database "+
		"for this hour: saved instance count %.0f, newer instance count %.0f", fact.Quantity, *dctx.value))
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
