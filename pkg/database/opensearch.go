package database

import (
	"database/sql"

	"github.com/appuio/appuio-cloud-reporting/pkg/db"
)

// OpensearchDBaaSType represents opensearch DBaaS type
const OpensearchDBaaSType ObjectType = "opensearch"

// Available plans for OpenSearch
var opensearchProductDBaaS = []ProductDBaaS{
	{Plan: "hobbyist-2", Target: "1414", Amount: 0.07875},
	{Plan: "startup-4", Target: "1414", Amount: 0.18937},
	{Plan: "startup-8", Target: "1414", Amount: 0.37583},
	{Plan: "startup-16", Target: "1414", Amount: 0.73911},
	{Plan: "startup-32", Target: "1414", Amount: 1.43845},
	{Plan: "business-4", Target: "1414", Amount: 0.54946},
	{Plan: "business-8", Target: "1414", Amount: 1.08649},
	{Plan: "business-16", Target: "1414", Amount: 2.16514},
	{Plan: "business-32", Target: "1414", Amount: 4.1663},
	{Plan: "premium-6x-8", Target: "1414", Amount: 2.17296},
	{Plan: "premium-6x-16", Target: "1414", Amount: 4.18124},
	{Plan: "premium-6x-32", Target: "1414", Amount: 7.82587},
	{Plan: "premium-9x-8", Target: "1414", Amount: 3.14767},
	{Plan: "premium-9x-16", Target: "1414", Amount: 5.8918},
	{Plan: "premium-9x-32", Target: "1414", Amount: 11.7388},
	{Plan: "premium-15x-16", Target: "1414", Amount: 9.81967},
	{Plan: "premium-15x-32", Target: "1414", Amount: 17.55262},
	{Plan: "premium-30x-16", Target: "1414", Amount: 17.62729},
	{Plan: "premium-30x-32", Target: "1414", Amount: 35.10523},
}

func generateOpensearchProducts() []*db.Product {
	products := make([]*db.Product, 0, len(opensearchProductDBaaS))
	for _, p := range opensearchProductDBaaS {
		s := dbaasSourceString{
			Query:        queryDBaaSOpensearch,
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
