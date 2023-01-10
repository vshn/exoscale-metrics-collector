package cloudscale

import (
	"database/sql"

	"github.com/appuio/appuio-cloud-reporting/pkg/db"
)

const (
	// source format: 'query:zone:tenant:namespace' or 'query:zone:tenant:namespace:class'
	// We do not have real (prometheus) queries here, just random hardcoded strings.
	sourceQueryStorage    = "object-storage-storage"
	sourceQueryTrafficOut = "object-storage-traffic-out"
	sourceQueryRequests   = "object-storage-requests"
)

var (
	// SourceZone represents the zone of the bucket, not of the cluster where the request for the bucket originated.
	// All the zones we use here must be known to the appuio-odoo-adapter as well.
	sourceZones = []string{"cloudscale"}
)

var (
	ensureProducts = []*db.Product{
		{
			Source: sourceQueryStorage + ":" + sourceZones[0],
			Target: sql.NullString{String: "1401", Valid: true},
			Amount: 0.0033,  // this is per DAY, equals 0.099 per GB per month
			Unit:   "GBDay", // SI GB according to cloudscale
			During: db.InfiniteRange(),
		},
		{
			Source: sourceQueryTrafficOut + ":" + sourceZones[0],
			Target: sql.NullString{String: "1403", Valid: true},
			Amount: 0.022,
			Unit:   "GB", // SI GB according to cloudscale
			During: db.InfiniteRange(),
		},
		{
			Source: sourceQueryRequests + ":" + sourceZones[0],
			Target: sql.NullString{String: "1405", Valid: true},
			Amount: 0.0055,
			Unit:   "KReq",
			During: db.InfiniteRange(),
		},
	}
)
var (
	ensureDiscounts = []*db.Discount{
		{
			Source:   sourceQueryStorage,
			Discount: 0,
			During:   db.InfiniteRange(),
		},
		{
			Source:   sourceQueryTrafficOut,
			Discount: 0,
			During:   db.InfiniteRange(),
		},
		{
			Source:   sourceQueryRequests,
			Discount: 0,
			During:   db.InfiniteRange(),
		},
	}
)

var (
	ensureQueries = []*db.Query{
		{
			Name:        sourceQueryStorage + ":" + sourceZones[0],
			Description: "Object Storage - Storage (cloudscale.ch)",
			Query:       "",
			Unit:        "GBDay",
			During:      db.InfiniteRange(),
		},
		{
			Name:        sourceQueryTrafficOut + ":" + sourceZones[0],
			Description: "Object Storage - Traffic Out (cloudscale.ch)",
			Query:       "",
			Unit:        "GB",
			During:      db.InfiniteRange(),
		},
		{
			Name:        sourceQueryRequests + ":" + sourceZones[0],
			Description: "Object Storage - Requests (cloudscale.ch)",
			Query:       "",
			Unit:        "KReq",
			During:      db.InfiniteRange(),
		},
	}
)
