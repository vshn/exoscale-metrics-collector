package cloudscale

import (
	"context"
	"fmt"
	"os"
	"time"

	"github.com/cloudscale-ch/cloudscale-go-sdk/v2"
	cloudscalev1 "github.com/vshn/provider-cloudscale/apis/cloudscale/v1"
	corev1 "k8s.io/api/core/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

const (
	organizationLabel = "appuio.io/organization"
	namespaceLabel    = "crossplane.io/claim-namespace"
)

// AccumulateKey represents one data point ("fact") in the billing database.
// The actual value for the data point is not present in this type, as this type is just a map key, and the corresponding value is stored as a map value.
type AccumulateKey struct {
	Query     string
	Zone      string
	Tenant    string
	Namespace string
	Start     time.Time
}

// String returns the full "source" string as used by the appuio-cloud-reporting
func (k AccumulateKey) String() string {
	return k.Query + ":" + k.Zone + ":" + k.Tenant + ":" + k.Namespace
}

/*
accumulateBucketMetrics gets all the bucket metrics from cloudscale and puts them into a map. The map key is the "AccumulateKey",
and the value is the raw value of the data returned by cloudscale (e.g. bytes, requests). In order to construct the
correct AccumulateKey, this function needs to fetch the namespace and bucket custom resources, because that's where the tenant is stored.
This method is "accumulating" data because it collects data from possibly multiple ObjectsUsers under the same
AccumulateKey. This is because the billing system can't handle multiple ObjectsUsers per namespace.
*/
func accumulateBucketMetrics(ctx context.Context, date time.Time, cloudscaleClient *cloudscale.Client, k8sclient client.Client) (map[AccumulateKey]uint64, error) {
	bucketMetricsRequest := cloudscale.BucketMetricsRequest{Start: date, End: date}
	bucketMetrics, err := cloudscaleClient.Metrics.GetBucketMetrics(ctx, &bucketMetricsRequest)
	if err != nil {
		return nil, err
	}

	nsTenants, err := fetchNamespaces(ctx, k8sclient)
	if err != nil {
		return nil, err
	}

	buckets, err := fetchBuckets(ctx, k8sclient)
	if err != nil {
		return nil, err
	}

	accumulated := make(map[AccumulateKey]uint64)

	for _, bucketMetricsData := range bucketMetrics.Data {
		name := bucketMetricsData.Subject.BucketName
		ns, ok := buckets[name]
		if !ok {
			fmt.Fprintf(os.Stderr, "WARNING: Cannot sync bucket, bucket resource %q not found\n", name)
			continue
		}
		tenant, ok := nsTenants[ns]
		if !ok {
			fmt.Fprintf(os.Stderr, "WARNING: Cannot sync bucket, namespace %q not found in map\n", ns)
			continue
		}

		err = accumulateBucketMetricsForObjectsUser(accumulated, bucketMetricsData, tenant, ns)
		if err != nil {
			fmt.Fprintf(os.Stderr, "WARNING: Cannot sync bucket %s: %v\n", name, err)
			continue
		}
	}

	return accumulated, nil
}

func fetchBuckets(ctx context.Context, k8sclient client.Client) (map[string]string, error) {
	buckets := &cloudscalev1.BucketList{}
	if err := k8sclient.List(ctx, buckets, client.HasLabels{namespaceLabel}); err != nil {
		return nil, fmt.Errorf("bucket list: %w", err)
	}

	bucketNS := map[string]string{}
	for _, b := range buckets.Items {
		bucketNS[b.GetBucketName()] = b.Labels[namespaceLabel]
	}
	return bucketNS, nil
}

func fetchNamespaces(ctx context.Context, k8sclient client.Client) (map[string]string, error) {
	namespaces := &corev1.NamespaceList{}
	if err := k8sclient.List(ctx, namespaces, client.HasLabels{organizationLabel}); err != nil {
		return nil, fmt.Errorf("namespace list: %w", err)
	}

	nsTenants := map[string]string{}
	for _, ns := range namespaces.Items {
		nsTenants[ns.Name] = ns.Labels[organizationLabel]
	}
	return nsTenants, nil
}

func accumulateBucketMetricsForObjectsUser(accumulated map[AccumulateKey]uint64, bucketMetricsData cloudscale.BucketMetricsData, tenant, namespace string) error {
	if len(bucketMetricsData.TimeSeries) != 1 {
		return fmt.Errorf("there must be exactly one metrics data point, found %d", len(bucketMetricsData.TimeSeries))
	}

	// For now all the buckets have the same zone. This may change in the future if Cloudscale decides to have different
	// prices for different locations.
	zone := sourceZones[0]

	sourceStorage := AccumulateKey{
		Query:     sourceQueryStorage,
		Zone:      zone,
		Tenant:    tenant,
		Namespace: namespace,
		Start:     bucketMetricsData.TimeSeries[0].Start,
	}
	sourceTrafficOut := AccumulateKey{
		Query:     sourceQueryTrafficOut,
		Zone:      zone,
		Tenant:    tenant,
		Namespace: namespace,
		Start:     bucketMetricsData.TimeSeries[0].Start,
	}
	sourceRequests := AccumulateKey{
		Query:     sourceQueryRequests,
		Zone:      zone,
		Tenant:    tenant,
		Namespace: namespace,
		Start:     bucketMetricsData.TimeSeries[0].Start,
	}

	accumulated[sourceStorage] += uint64(bucketMetricsData.TimeSeries[0].Usage.StorageBytes)
	accumulated[sourceTrafficOut] += uint64(bucketMetricsData.TimeSeries[0].Usage.SentBytes)
	accumulated[sourceRequests] += uint64(bucketMetricsData.TimeSeries[0].Usage.Requests)

	return nil
}
