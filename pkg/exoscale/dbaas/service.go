package dbaas

import (
	"context"
	"fmt"
	"time"

	common "github.com/vshn/exoscale-metrics-collector/pkg/exoscale"
	"k8s.io/apimachinery/pkg/api/errors"

	pipeline "github.com/ccremer/go-command-pipeline"
	egoscale "github.com/exoscale/egoscale/v2"
	"github.com/go-logr/logr"
	"github.com/vshn/exoscale-metrics-collector/pkg/clients/exoscale"
	"github.com/vshn/exoscale-metrics-collector/pkg/database"
	db "github.com/vshn/exoscale-metrics-collector/pkg/database"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/client-go/dynamic"
	ctrl "sigs.k8s.io/controller-runtime"
)

var (
	groupVersionResources = map[string]schema.GroupVersionResource{
		"pg": {
			Group:    "exoscale.crossplane.io",
			Version:  "v1",
			Resource: "postgresqls",
		},
		"mysql": {
			Group:    "exoscale.crossplane.io",
			Version:  "v1",
			Resource: "mysqls",
		},
		"opensearch": {
			Group:    "exoscale.crossplane.io",
			Version:  "v1",
			Resource: "opensearches",
		},
		"redis": {
			Group:    "exoscale.crossplane.io",
			Version:  "v1",
			Resource: "redis",
		},
		"kafka": {
			Group:    "exoscale.crossplane.io",
			Version:  "v1",
			Resource: "kafkas",
		},
	}
)

// Detail a helper structure for intermediate operations
type Detail struct {
	Organization, DBName, Namespace, Plan, Zone, Type string
}

// Context is the context of the DBaaS service
type Context struct {
	context.Context
	dbaasDetails     []Detail
	exoscaleDBaasS   []*egoscale.DatabaseService
	aggregatedDBaasS map[db.Key]db.Aggregated
}

// Service provides DBaaS Database info and required clients
type Service struct {
	exoscaleClient *egoscale.Client
	k8sClient      dynamic.Interface
	database       *db.DBaaSDatabase
}

// NewDBaaSService creates a Service with the initial setup
func NewDBaaSService(exoscaleClient *egoscale.Client, k8sClient dynamic.Interface, databaseURL string) Service {
	return Service{
		exoscaleClient: exoscaleClient,
		k8sClient:      k8sClient,
		database: &db.DBaaSDatabase{
			Database: db.Database{
				URL: databaseURL,
			},
		},
	}
}

// Execute executes the main business logic for this application by gathering, matching and saving data to the database
func (s *Service) Execute(ctx context.Context) error {
	log := ctrl.LoggerFrom(ctx)
	log.Info("Running metrics collector by step")

	p := pipeline.NewPipeline[*Context]()
	p.WithSteps(
		p.NewStep("Fetch cluster managed DBaaS and namespaces", s.fetchManagedDBaaSAndNamespaces),
		p.NewStep("Fetch exoscale DBaaS usage", s.fetchDBaaSUsage),
		p.NewStep("Aggregate DBaaS services by namespace and plan", aggregateDBaaS),
		p.WithNestedSteps("Save to billing database", hasAggregatedInstances,
			p.NewStep("Get billing date", s.getBillingDate),
			p.NewStep("Save to database", s.saveToDatabase),
		),
	)

	return p.RunWithContext(&Context{Context: ctx})
}

// fetchManagedDBaaSAndNamespaces fetches instances and namespaces from kubernetes cluster
func (s *Service) fetchManagedDBaaSAndNamespaces(ctx *Context) error {
	log := ctrl.LoggerFrom(ctx)

	log.V(1).Info("Listing namespaces from cluster")
	namespaces, err := fetchNamespaceWithOrganizationMap(ctx, s.k8sClient)
	if err != nil {
		return fmt.Errorf("cannot list namespaces: %w", err)
	}

	var dbaasDetails []Detail
	for _, groupVersionResource := range groupVersionResources {
		managedResources, err := s.k8sClient.Resource(groupVersionResource).List(ctx, metav1.ListOptions{})
		if err != nil {
			if errors.IsNotFound(err) {
				continue
			}
			return fmt.Errorf("cannot list managed resource %s from cluster: %w", groupVersionResource.Resource, err)
		}

		for _, managedResource := range managedResources.Items {
			dbaasDetail := findDBaaSDetailInNamespacesMap(managedResource, groupVersionResource, namespaces, log)
			if dbaasDetail == nil {
				continue
			}
			dbaasDetails = append(dbaasDetails, *dbaasDetail)
		}
	}

	ctx.dbaasDetails = dbaasDetails
	return nil
}

func findDBaaSDetailInNamespacesMap(managedResource unstructured.Unstructured, groupVersionResource schema.GroupVersionResource, namespaces map[string]string, log logr.Logger) *Detail {
	dbaasDetail := Detail{
		DBName: managedResource.GetName(),
		Type:   groupVersionResource.Resource,
	}
	if namespace, exist := managedResource.GetLabels()[common.NamespaceLabel]; exist {
		organization, ok := namespaces[namespace]
		if !ok {
			// cannot find namespace in namespace list
			log.Info("Namespace not found in namespace list, skipping...",
				"namespace", namespace,
				"dbaas", managedResource.GetName())
			return nil
		}
		dbaasDetail.Namespace = namespace
		dbaasDetail.Organization = organization
	} else {
		// cannot get namespace from DBaaS
		log.Info("Namespace label is missing in DBaaS, skipping...",
			"label", common.NamespaceLabel,
			"dbaas", managedResource.GetName())
		return nil
	}
	log.V(1).Info("Added namespace and organization to DBaaS",
		"dbaas", managedResource.GetName(),
		"namespace", dbaasDetail.Namespace,
		"organization", dbaasDetail.Organization)
	return &dbaasDetail
}

// fetchDBaaSUsage gets DBaaS service usage from Exoscale
func (s *Service) fetchDBaaSUsage(ctx *Context) error {
	log := ctrl.LoggerFrom(ctx)
	log.Info("Fetching DBaaS usage from Exoscale")

	var databaseServices []*egoscale.DatabaseService
	for _, zone := range exoscale.Zones {
		databaseServicesByZone, err := s.exoscaleClient.ListDatabaseServices(ctx, zone)
		if err != nil {
			log.V(1).Error(err, "Cannot get exoscale database services on zone", "zone", zone)
			return err
		}
		databaseServices = append(databaseServices, databaseServicesByZone...)
	}
	ctx.exoscaleDBaasS = databaseServices
	return nil
}

// aggregateDBaaS aggregates DBaaS services by namespaces and plan
func aggregateDBaaS(ctx *Context) error {
	log := ctrl.LoggerFrom(ctx)
	log.Info("Aggregating DBaaS instances by namespace and plan")

	// The DBaaS names are unique across DB types in an Exoscale organization.
	dbaasServiceUsageMap := make(map[string]egoscale.DatabaseService, len(ctx.exoscaleDBaasS))
	for _, usage := range ctx.exoscaleDBaasS {
		dbaasServiceUsageMap[*usage.Name] = *usage
	}

	aggregatedDBaasS := make(map[db.Key]db.Aggregated)
	for _, dbaasDetail := range ctx.dbaasDetails {
		log.V(1).Info("Checking DBaaS", "instance", dbaasDetail.DBName)

		dbaasUsage, exists := dbaasServiceUsageMap[dbaasDetail.DBName]
		if exists && dbaasDetail.Type == groupVersionResources[*dbaasUsage.Type].Resource {
			log.V(1).Info("Found exoscale dbaas usage", "instance", dbaasUsage.Name, "instance created", dbaasUsage.CreatedAt)
			key := db.NewKey(dbaasDetail.Namespace, *dbaasUsage.Plan, *dbaasUsage.Type)
			aggregated := aggregatedDBaasS[key]
			aggregated.Key = key
			aggregated.Organization = dbaasDetail.Organization
			aggregated.Value++
			aggregatedDBaasS[key] = aggregated
		} else {
			log.Info("Could not find any DBaaS on exoscale", "instance", dbaasDetail.DBName)
		}
	}

	ctx.aggregatedDBaasS = aggregatedDBaasS
	return nil
}

// getBillingDate sets the date for which the billing takes place.
func (s *Service) getBillingDate(_ *Context) error {
	location, err := time.LoadLocation(common.ExoscaleTimeZone)
	if err != nil {
		return fmt.Errorf("cannot initialize location from time zone %s: %w", location, err)
	}
	now := time.Now().In(location)
	s.database.BillingDate = time.Date(now.Year(), now.Month(), now.Day(), now.Hour(), 0, 0, 0, now.Location())
	return nil
}

// saveToDatabase tries to save metrics in the database.
func (s *Service) saveToDatabase(ctx *Context) error {
	log := ctrl.LoggerFrom(ctx)
	log.Info("Creating a database connection")

	dctx := &database.Context{
		Context:           ctx,
		AggregatedObjects: &ctx.aggregatedDBaasS,
	}
	err := s.database.Execute(dctx)
	if err != nil {
		log.Error(err, "Cannot save to database")
	}
	return nil
}

func hasAggregatedInstances(ctx *Context) bool {
	log := ctrl.LoggerFrom(ctx)
	nrAggregatedInstances := len(ctx.aggregatedDBaasS)
	if nrAggregatedInstances == 0 {
		log.Info("There are no DBaaS instances to be saved in the database")
		return false
	}
	return true
}

func fetchNamespaceWithOrganizationMap(ctx context.Context, k8sClient dynamic.Interface) (map[string]string, error) {
	log := ctrl.LoggerFrom(ctx)
	nsGroupVersionResource := schema.GroupVersionResource{
		Group:    "",
		Version:  "v1",
		Resource: "namespaces",
	}
	list, err := k8sClient.Resource(nsGroupVersionResource).List(ctx, metav1.ListOptions{})
	if err != nil {
		return nil, fmt.Errorf("cannot get namespace list: %w", err)
	}

	namespaces := map[string]string{}
	for _, ns := range list.Items {
		org, ok := ns.GetLabels()[common.OrganizationLabel]
		if !ok {
			log.Info("Organization label not found in namespace", "namespace", ns.GetName())
			continue
		}
		namespaces[ns.GetName()] = org
	}
	return namespaces, nil
}
