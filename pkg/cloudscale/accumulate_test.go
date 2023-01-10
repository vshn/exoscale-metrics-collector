package cloudscale

import (
	"fmt"
	"testing"
	"time"

	"github.com/cloudscale-ch/cloudscale-go-sdk/v2"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// assertEqualfUint64 implements the functionality of assert.Equalf for uint64, because assert.Equalf cannot print uint64 correctly.
// See https://github.com/stretchr/testify/issues/400
func assertEqualfUint64(t *testing.T, expected uint64, actual uint64, msg string, args ...interface{}) bool {
	if expected != actual {
		return assert.Fail(t, fmt.Sprintf("Not equal: \n"+
			"expected: %d\n"+
			"actual  : %d", expected, actual))
	}
	return true
}

func TestAccumulateBucketMetricsForObjectsUser(t *testing.T) {
	zone := "cloudscale"
	organization := "inity"
	namespace := "testnamespace"

	now := time.Now().In(location)
	date := time.Date(now.Year(), now.Month(), now.Day()-1, 0, 0, 0, 0, now.Location())

	// build input data structure
	bucketMetricsInterval := []cloudscale.BucketMetricsInterval{
		{
			Start: date,
			End:   date,
			Usage: cloudscale.BucketMetricsIntervalUsage{
				Requests:     5,
				StorageBytes: 1000000,
				SentBytes:    2000000,
			},
		},
	}
	bucketMetricsData := cloudscale.BucketMetricsData{
		TimeSeries: bucketMetricsInterval,
	}

	accumulated := make(map[AccumulateKey]uint64)
	assert.NoError(t, accumulateBucketMetricsForObjectsUser(accumulated, bucketMetricsData, organization, namespace))

	require.Len(t, accumulated, 3, "incorrect amount of values 'accumulated'")

	key := AccumulateKey{
		Zone:      zone,
		Tenant:    organization,
		Namespace: namespace,
		Start:     date,
	}

	key.Query = "object-storage-requests"
	assertEqualfUint64(t, uint64(5), accumulated[key], "incorrect value in %s", key)

	key.Query = "object-storage-storage"
	assertEqualfUint64(t, uint64(1000000), accumulated[key], "incorrect value in %s", key)

	key.Query = "object-storage-traffic-out"
	assertEqualfUint64(t, uint64(2000000), accumulated[key], "incorrect value in %s", key)
}
