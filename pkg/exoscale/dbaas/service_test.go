package dbaas

import (
	"context"
	"reflect"
	"testing"

	egoscale "github.com/exoscale/egoscale/v2"
	"github.com/stretchr/testify/assert"
	db "github.com/vshn/exoscale-metrics-collector/pkg/database"
)

func TestDBaaS_aggregatedDBaaS(t *testing.T) {
	ctx := context.Background()

	key1 := db.NewKey("vshn-xyz", "hobbyist-2", string(db.PostgresDBaaSType))
	key2 := db.NewKey("vshn-abc", "business-128", string(db.PostgresDBaaSType))

	expectedAggregatedDBaaS := map[db.Key]db.Aggregated{
		key1: {
			Key:          key1,
			Organization: "org1",
			Value:        1,
		},
		key2: {
			Key:          key2,
			Organization: "org2",
			Value:        1,
		},
	}

	tests := map[string]struct {
		ctx                     Context
		expectedAggregatedDBaaS map[db.Key]db.Aggregated
	}{
		"given DBaaS details and Exoscale DBaasS, we should get the ExpectedAggregatedDBaasS": {
			ctx: Context{
				Context: ctx,
				dbaasDetails: []Detail{
					{
						Organization: "org1",
						DBName:       "postgres-abc",
						Namespace:    "vshn-xyz",
						Zone:         "ch-gva-2",
					},
					{
						Organization: "org2",
						DBName:       "postgres-def",
						Namespace:    "vshn-abc",
						Zone:         "ch-gva-2",
					},
				},
				exoscaleDBaasS: []*egoscale.DatabaseService{
					{
						Name: strToPointer("postgres-abc"),
						Type: strToPointer(string(db.PostgresDBaaSType)),
						Plan: strToPointer("hobbyist-2"),
					},
					{
						Name: strToPointer("postgres-def"),
						Type: strToPointer(string(db.PostgresDBaaSType)),
						Plan: strToPointer("business-128"),
					},
				},
				aggregatedDBaasS: map[db.Key]db.Aggregated{},
			},
			expectedAggregatedDBaaS: expectedAggregatedDBaaS,
		},
		"given DBaaS details and different names in Exoscale DBaasS, we should not get the ExpectedAggregatedDBaasS": {
			ctx: Context{
				Context: ctx,
				dbaasDetails: []Detail{
					{
						Organization: "org1",
						DBName:       "postgres-abc",
						Namespace:    "vshn-xyz",
						Zone:         "ch-gva-2",
					},
					{
						Organization: "org2",
						DBName:       "postgres-def",
						Namespace:    "vshn-abc",
						Zone:         "ch-gva-2",
					},
				},
				exoscaleDBaasS: []*egoscale.DatabaseService{
					{
						Name: strToPointer("postgres-123"),
						Type: strToPointer(string(db.PostgresDBaaSType)),
					},
					{
						Name: strToPointer("postgres-456"),
						Type: strToPointer(string(db.PostgresDBaaSType)),
					},
				},
				aggregatedDBaasS: map[db.Key]db.Aggregated{},
			},
			expectedAggregatedDBaaS: map[db.Key]db.Aggregated{},
		},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			err := aggregateDBaaS(&tc.ctx)
			assert.NoError(t, err)
			assert.True(t, reflect.DeepEqual(tc.expectedAggregatedDBaaS, tc.ctx.aggregatedDBaasS))
		})
	}
}

func strToPointer(s string) *string {
	return &s
}
