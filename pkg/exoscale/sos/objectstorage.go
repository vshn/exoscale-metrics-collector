package sos

import (
	"context"
	"fmt"
	"time"

	pipeline "github.com/ccremer/go-command-pipeline"
	egoscale "github.com/exoscale/egoscale/v2"
	"github.com/exoscale/egoscale/v2/oapi"
	"github.com/vshn/exoscale-metrics-collector/pkg/database"
	db "github.com/vshn/exoscale-metrics-collector/pkg/database"
	common "github.com/vshn/exoscale-metrics-collector/pkg/exoscale"
	exoscalev1 "github.com/vshn/provider-exoscale/apis/exoscale/v1"
	corev1 "k8s.io/api/core/v1"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	k8s "sigs.k8s.io/controller-runtime/pkg/client"
)

// ObjectStorage gathers bucket data from exoscale provider and cluster and saves to the database
type ObjectStorage struct {
	k8sClient         k8s.Client
	exoscaleClient    *egoscale.Client
	database          *db.SosDatabase
	bucketDetails     []BucketDetail
	aggregatedBuckets map[db.Key]db.Aggregated
}

// BucketDetail a k8s bucket object with relevant data
type BucketDetail struct {
	Organization, BucketName, Namespace string
}

// NewObjectStorage creates an ObjectStorage with the initial setup
func NewObjectStorage(exoscaleClient *egoscale.Client, k8sClient k8s.Client, databaseURL string) ObjectStorage {
	return ObjectStorage{
		exoscaleClient: exoscaleClient,
		k8sClient:      k8sClient,
		database: &db.SosDatabase{
			Database: db.Database{
				URL: databaseURL,
			},
		},
	}
}

// Execute executes the main business logic for this application by gathering, matching and saving data to the database
func (o *ObjectStorage) Execute(ctx context.Context) error {
	log := ctrl.LoggerFrom(ctx)
	log.Info("Running metrics collector by step")

	p := pipeline.NewPipeline[context.Context]()
	p.WithSteps(
		p.NewStep("Fetch managed buckets and namespaces", o.fetchManagedBucketsAndNamespaces),
		p.NewStep("Get bucket usage", o.getBucketUsage),
		p.NewStep("Get billing date", o.getBillingDate),
		p.NewStep("Save to database", o.saveToDatabase),
	)
	return p.RunWithContext(ctx)
}

// getBucketUsage gets bucket usage from Exoscale and matches them with the bucket from the cluster
// If there are no buckets in Exoscale, the API will return an empty slice
func (o *ObjectStorage) getBucketUsage(ctx context.Context) error {
	log := ctrl.LoggerFrom(ctx)
	log.Info("Fetching bucket usage from Exoscale")
	resp, err := o.exoscaleClient.ListSosBucketsUsageWithResponse(ctx)
	if err != nil {
		return err
	}

	aggregatedBuckets := getAggregatedBuckets(ctx, *resp.JSON200.SosBucketsUsage, o.bucketDetails)
	if len(aggregatedBuckets) == 0 {
		log.Info("There are no bucket usage to be saved in the database")
		return nil
	}

	o.aggregatedBuckets = aggregatedBuckets
	return nil
}

func (o *ObjectStorage) fetchManagedBucketsAndNamespaces(ctx context.Context) error {
	log := ctrl.LoggerFrom(ctx)
	log.Info("Fetching buckets and namespaces from cluster")

	buckets := exoscalev1.BucketList{}
	log.V(1).Info("Listing buckets from cluster")
	err := o.k8sClient.List(ctx, &buckets)
	if err != nil {
		return fmt.Errorf("cannot list buckets: %w", err)
	}

	log.V(1).Info("Listing namespaces from cluster")
	namespaces, err := fetchNamespaceWithOrganizationMap(ctx, o.k8sClient)
	if err != nil {
		return fmt.Errorf("cannot list namespaces: %w", err)
	}

	o.bucketDetails = addOrgAndNamespaceToBucket(ctx, buckets, namespaces)
	return nil
}

func (o *ObjectStorage) saveToDatabase(ctx context.Context) error {
	log := ctrl.LoggerFrom(ctx)
	log.Info("Creating a database connection")

	dctx := &database.Context{
		Context:           ctx,
		AggregatedObjects: &o.aggregatedBuckets,
	}

	err := o.database.Execute(dctx)
	if err != nil {
		log.Error(err, "Cannot save to database")
	}
	return nil
}

func (o *ObjectStorage) getBillingDate(_ context.Context) error {
	location, err := time.LoadLocation(common.ExoscaleTimeZone)
	if err != nil {
		return fmt.Errorf("cannot initialize location from time zone %s: %w", location, err)
	}
	now := time.Now().In(location)
	previousDay := now.Day() - 1
	o.database.BillingDate = time.Date(now.Year(), now.Month(), previousDay, common.ExoscaleBillingHour, 0, 0, 0, now.Location())
	return nil
}

func getAggregatedBuckets(ctx context.Context, sosBucketsUsage []oapi.SosBucketUsage, bucketDetails []BucketDetail) map[db.Key]db.Aggregated {
	log := ctrl.LoggerFrom(ctx)
	log.Info("Aggregating buckets by namespace")

	sosBucketsUsageMap := make(map[string]oapi.SosBucketUsage, len(sosBucketsUsage))
	for _, usage := range sosBucketsUsage {
		sosBucketsUsageMap[*usage.Name] = usage
	}

	aggregatedBuckets := make(map[db.Key]db.Aggregated)
	for _, bucketDetail := range bucketDetails {
		log.V(1).Info("Checking bucket", "bucket", bucketDetail.BucketName)

		if bucketUsage, exists := sosBucketsUsageMap[bucketDetail.BucketName]; exists {
			log.V(1).Info("Found exoscale bucket usage", "bucket", bucketUsage.Name, "bucket size", bucketUsage.Name)
			key := db.NewKey(bucketDetail.Namespace)
			aggregatedBucket := aggregatedBuckets[key]
			aggregatedBucket.Key = key
			aggregatedBucket.Organization = bucketDetail.Organization
			aggregatedBucket.Value += float64(*bucketUsage.Size)
			aggregatedBuckets[key] = aggregatedBucket
		} else {
			log.Info("Could not find any bucket on exoscale", "bucket", bucketDetail.BucketName)
		}
	}
	return aggregatedBuckets
}

func addOrgAndNamespaceToBucket(ctx context.Context, buckets exoscalev1.BucketList, namespaces map[string]string) []BucketDetail {
	log := ctrl.LoggerFrom(ctx)
	log.V(1).Info("Gathering org and namespace from buckets")

	bucketDetails := make([]BucketDetail, 0, 10)
	for _, bucket := range buckets.Items {
		bucketDetail := BucketDetail{
			BucketName: bucket.Spec.ForProvider.BucketName,
		}
		if namespace, exist := bucket.ObjectMeta.Labels[common.NamespaceLabel]; exist {
			organization, ok := namespaces[namespace]
			if !ok {
				// cannot find namespace in namespace list
				log.Info("Namespace not found in namespace list, skipping...",
					"namespace", namespace,
					"bucket", bucket.Name)
				continue
			}
			bucketDetail.Namespace = namespace
			bucketDetail.Organization = organization
		} else {
			// cannot get namespace from bucket
			log.Info("Namespace label is missing in bucket, skipping...",
				"label", common.NamespaceLabel,
				"bucket", bucket.Name)
			continue
		}
		log.V(1).Info("Added namespace and organization to bucket",
			"bucket", bucket.Name,
			"namespace", bucketDetail.Namespace,
			"organization", bucketDetail.Organization)
		bucketDetails = append(bucketDetails, bucketDetail)
	}
	return bucketDetails
}

func fetchNamespaceWithOrganizationMap(ctx context.Context, k8sclient client.Client) (map[string]string, error) {
	log := ctrl.LoggerFrom(ctx)
	list := &corev1.NamespaceList{}
	if err := k8sclient.List(ctx, list); err != nil {
		return nil, fmt.Errorf("cannot get namespace list: %w", err)
	}

	namespaces := map[string]string{}
	for _, ns := range list.Items {
		org, ok := ns.Labels[common.OrganizationLabel]
		if !ok {
			log.Info("Organization label not found in namespace", "namespace", ns.Name)
			continue
		}
		namespaces[ns.Name] = org
	}
	return namespaces, nil
}
