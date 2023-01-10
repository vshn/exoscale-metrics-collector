package database

import (
	"database/sql"

	"github.com/appuio/appuio-cloud-reporting/pkg/db"
)

// RedisDBaaSType represents redis DBaaS type
const RedisDBaaSType ObjectType = "redis"

// Available plans for Redis
var redisProductDBaaS = []ProductDBaaS{
	{Plan: "hobbyist-2", Target: "1413", Amount: 0.06779},
	{Plan: "startup-4", Target: "1413", Amount: 0.13357},
	{Plan: "startup-8", Target: "1413", Amount: 0.25920},
	{Plan: "startup-16", Target: "1413", Amount: 0.50718},
	{Plan: "startup-32", Target: "1413", Amount: 0.98935},
	{Plan: "startup-64", Target: "1413", Amount: 1.89367},
	{Plan: "startup-128", Target: "1413", Amount: 3.52751},
	{Plan: "startup-225", Target: "1413", Amount: 5.67208},
	{Plan: "business-1", Target: "1413", Amount: 0.07800},
	{Plan: "business-4", Target: "1413", Amount: 0.26140},
	{Plan: "business-8", Target: "1413", Amount: 0.50920},
	{Plan: "business-16", Target: "1413", Amount: 0.99136},
	{Plan: "business-32", Target: "1413", Amount: 1.89589},
	{Plan: "business-64", Target: "1413", Amount: 3.52973},
	{Plan: "business-128", Target: "1413", Amount: 7.05502},
	{Plan: "business-225", Target: "1413", Amount: 9.85634},
	{Plan: "premium-4", Target: "1413", Amount: 0.38519},
	{Plan: "premium-8", Target: "1413", Amount: 0.74655},
	{Plan: "premium-16", Target: "1413", Amount: 1.48704},
	{Plan: "premium-32", Target: "1413", Amount: 2.84384},
	{Plan: "premium-64", Target: "1413", Amount: 5.29459},
	{Plan: "premium-128", Target: "1413", Amount: 9.31293},
	{Plan: "premium-225", Target: "1413", Amount: 14.78450},
}

func generateRedisProducts() []*db.Product {
	products := make([]*db.Product, 0, len(redisProductDBaaS))
	for _, p := range redisProductDBaaS {
		s := dbaasSourceString{
			Query:        queryDBaaSRedis,
			Organization: "*",
			Namespace:    "*",
			Plan:         p.Plan,
		}
		product := db.Product{
			Source: s.getSourceString(),
			Target: sql.NullString{String: p.Target, Valid: true},
			Amount: p.Amount,
			Unit:   defaultUnitDBaaS,
			During: db.InfiniteRange(),
		}
		products = append(products, &product)
	}
	return products
}
