package database

import (
	"database/sql"

	"github.com/appuio/appuio-cloud-reporting/pkg/db"
)

// KafkaDBaaSType represents kafka DBaaS type
const KafkaDBaaSType ObjectType = "kafka"

// Available plans for Kafka
var kafkaProductDBaaS = []ProductDBaaS{
	{Plan: "startup-2", Target: "1416", Amount: 0.34305},
	{Plan: "business-4", Target: "1416", Amount: 0.85131},
	{Plan: "business-8", Target: "1416", Amount: 1.64491},
	{Plan: "business-16", Target: "1416", Amount: 3.31770},
	{Plan: "business-32", Target: "1416", Amount: 6.33919},
	{Plan: "premium-6x-8", Target: "1416", Amount: 3.33140},
	{Plan: "premium-6x-16", Target: "1416", Amount: 6.35223},
	{Plan: "premium-6x-32", Target: "1416", Amount: 11.71154},
	{Plan: "premium-9x-8", Target: "1416", Amount: 4.78083},
	{Plan: "premium-9x-16", Target: "1416", Amount: 8.80127},
	{Plan: "premium-9x-32", Target: "1416", Amount: 17.56341},
	{Plan: "premium-15x-8", Target: "1416", Amount: 7.37187},
	{Plan: "premium-15x-16", Target: "1416", Amount: 14.67527},
	{Plan: "premium-15x-32", Target: "1416", Amount: 25.54634},
	{Plan: "premium-30x-8", Target: "1416", Amount: 14.74374},
	{Plan: "premium-30x-16", Target: "1416", Amount: 25.61155},
	{Plan: "premium-30x-32", Target: "1416", Amount: 51.09267},
}

func generateKafkaProducts() []*db.Product {
	products := make([]*db.Product, 0, len(kafkaProductDBaaS))
	for _, p := range kafkaProductDBaaS {
		s := dbaasSourceString{
			Query:        queryDBaaSKafka,
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
