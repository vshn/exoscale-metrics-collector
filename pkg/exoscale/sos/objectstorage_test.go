package sos

import (
	"context"
	"reflect"
	"testing"
	"time"

	"github.com/exoscale/egoscale/v2/oapi"
	"github.com/stretchr/testify/assert"
	db "github.com/vshn/exoscale-metrics-collector/pkg/database"
	common "github.com/vshn/exoscale-metrics-collector/pkg/exoscale"
	exoscalev1 "github.com/vshn/provider-exoscale/apis/exoscale/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func TestObjectStorage_GetBillingDate(t *testing.T) {
	t.Run("GivenContext_WhenGetBillingDate_ThenReturnYesterdayDate", func(t *testing.T) {
		utc, _ := time.LoadLocation("UTC")
		now := time.Now().In(utc)
		expected := time.Date(now.Year(), now.Month(), time.Now().Day()-1, 6, 0, 0, 0, now.Location())

		o, err := NewObjectStorage(nil, nil, "")
		assert.NoError(t, err)
		assert.Equal(t, o.database.BillingDate, expected)
	})
}

func TestObjectStorage_GetAggregated(t *testing.T) {
	defaultKey := db.NewKey("default")
	alphaKey := db.NewKey("alpha")
	omegaKey := db.NewKey("omega")

	tests := map[string]struct {
		givenSosBucketsUsage []oapi.SosBucketUsage
		givenBucketDetails   []BucketDetail
		expectedAggregated   map[db.Key]db.Aggregated
	}{
		"GivenSosBucketsUsageAndBuckets_WhenMatch_ThenExpectAggregatedObjects": {
			givenSosBucketsUsage: []oapi.SosBucketUsage{
				createSosBucketUsage("bucket-test-1", 1),
				createSosBucketUsage("bucket-test-2", 4),
				createSosBucketUsage("bucket-test-3", 9),
				createSosBucketUsage("bucket-test-4", 0),
				createSosBucketUsage("bucket-test-5", 5),
			},
			givenBucketDetails: []BucketDetail{
				createBucketDetail("bucket-test-1", "default", "orgA"),
				createBucketDetail("bucket-test-2", "alpha", "orgB"),
				createBucketDetail("bucket-test-3", "alpha", "orgB"),
				createBucketDetail("bucket-test-4", "omega", "orgC"),
				createBucketDetail("no-metrics-bucket", "beta", "orgD"),
			},
			expectedAggregated: map[db.Key]db.Aggregated{
				defaultKey: createAggregated(defaultKey, "orgA", 1),
				alphaKey:   createAggregated(alphaKey, "orgB", 13),
				omegaKey:   createAggregated(omegaKey, "orgC", 0),
			},
		},
		"GivenSosBucketsUsageAndBuckets_WhenMatch_ThenExpectNoAggregatedObjects": {
			givenSosBucketsUsage: []oapi.SosBucketUsage{
				createSosBucketUsage("bucket-test-1", 1),
				createSosBucketUsage("bucket-test-2", 4),
			},
			givenBucketDetails: []BucketDetail{
				createBucketDetail("bucket-test-3", "default", "orgA"),
				createBucketDetail("bucket-test-4", "alpha", "orgB"),
				createBucketDetail("bucket-test-5", "alpha", "orgB"),
			},
			expectedAggregated: map[db.Key]db.Aggregated{},
		},
		"GivenSosBucketsUsageAndBuckets_WhenSosBucketsUsageEmpty_ThenExpectNoAggregatedObjects": {
			givenSosBucketsUsage: []oapi.SosBucketUsage{
				createSosBucketUsage("bucket-test-1", 1),
				createSosBucketUsage("bucket-test-2", 4),
			},
			givenBucketDetails: []BucketDetail{},
			expectedAggregated: map[db.Key]db.Aggregated{},
		},
		"GivenSosBucketsUsageAndBuckets_WhenNoBuckets_ThenExpectNoAggregatedObjects": {
			givenSosBucketsUsage: []oapi.SosBucketUsage{},
			givenBucketDetails: []BucketDetail{
				createBucketDetail("bucket-test-3", "default", "orgA"),
				createBucketDetail("bucket-test-4", "alpha", "orgB"),
			},
			expectedAggregated: map[db.Key]db.Aggregated{},
		},
	}
	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			// Given
			ctx := context.Background()

			// When
			aggregated := getAggregatedBuckets(ctx, tc.givenSosBucketsUsage, tc.givenBucketDetails)

			// Then
			assert.True(t, reflect.DeepEqual(aggregated, tc.expectedAggregated))
		})
	}
}

func TestObjectStorage_addOrgAndNamespaceToBucket(t *testing.T) {
	tests := map[string]struct {
		givenBucketList       exoscalev1.BucketList
		givenNamespaces       map[string]string
		expectedBucketDetails []BucketDetail
	}{
		"GivenBucketListFromExoscale_WhenOrgAndNamespaces_ThenExpectBucketDetailObjects": {
			givenBucketList: exoscalev1.BucketList{
				Items: []exoscalev1.Bucket{
					createBucket("bucket-1", "alpha", "orgA"),
					createBucket("bucket-2", "beta", "orgB"),
					createBucket("bucket-3", "alpha", "orgA"),
					createBucket("bucket-4", "omega", "orgB"),
					createBucket("bucket-5", "theta", "orgC"),
				},
			},
			givenNamespaces: map[string]string{
				"alpha": "orgA",
				"beta":  "orgB",
				"omega": "orgB",
				"theta": "orgC",
			},
			expectedBucketDetails: []BucketDetail{
				createBucketDetail("bucket-1", "alpha", "orgA"),
				createBucketDetail("bucket-2", "beta", "orgB"),
				createBucketDetail("bucket-3", "alpha", "orgA"),
				createBucketDetail("bucket-4", "omega", "orgB"),
				createBucketDetail("bucket-5", "theta", "orgC"),
			},
		},
		"GivenBucketListFromExoscale_WhenNoOrgOrNamespaces_ThenExpectNoBucketDetailObjects": {
			givenBucketList: exoscalev1.BucketList{
				Items: []exoscalev1.Bucket{
					createBucket("bucket-1", "", "orgA"),
					createBucket("bucket-2", "beta", ""),
					createBucket("bucket-3", "", ""),
				},
			},
			givenNamespaces:       map[string]string{},
			expectedBucketDetails: []BucketDetail{},
		},
	}
	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			// Given
			ctx := context.Background()

			// When
			bucketDetails := addOrgAndNamespaceToBucket(ctx, tc.givenBucketList, tc.givenNamespaces)

			// Then
			assert.ElementsMatch(t, tc.expectedBucketDetails, bucketDetails)
		})
	}
}

func createBucket(name, namespace, organization string) exoscalev1.Bucket {
	labels := make(map[string]string)
	if namespace != "" {
		labels[common.NamespaceLabel] = namespace
	}
	if organization != "" {
		labels[common.OrganizationLabel] = organization
	}
	return exoscalev1.Bucket{
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: namespace,
			Labels:    labels,
		},
		Spec: exoscalev1.BucketSpec{
			ForProvider: exoscalev1.BucketParameters{
				BucketName: name,
			},
		},
	}
}

func createAggregated(key db.Key, organization string, size float64) db.Aggregated {
	return db.Aggregated{
		Key:          key,
		Organization: organization,
		Value:        size,
	}
}

func createBucketDetail(bucketName, namespace, organization string) BucketDetail {
	return BucketDetail{
		Organization: organization,
		BucketName:   bucketName,
		Namespace:    namespace,
	}
}

func createSosBucketUsage(bucketName string, size int) oapi.SosBucketUsage {
	date := time.Now()
	actualSize := int64(size)
	zone := oapi.ZoneName("ch-gva-2")
	return oapi.SosBucketUsage{
		CreatedAt: &date,
		Name:      &bucketName,
		Size:      &actualSize,
		ZoneName:  &zone,
	}
}
