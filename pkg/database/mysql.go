package database

import (
	"database/sql"

	"github.com/appuio/appuio-cloud-reporting/pkg/db"
)

// MysqlDBaaSType represents mysql DBaaS type
const MysqlDBaaSType ObjectType = "mysql"

// Available plans for MySQL
var mysqlProductDBaaS = []ProductDBaaS{
	{Plan: "hobbyist-2", Target: "1412", Amount: 0.06683},
	{Plan: "startup-4", Target: "1412", Amount: 0.15731},
	{Plan: "startup-8", Target: "1412", Amount: 0.30889},
	{Plan: "startup-16", Target: "1412", Amount: 0.60507},
	{Plan: "startup-32", Target: "1412", Amount: 1.10238},
	{Plan: "startup-64", Target: "1412", Amount: 2.02408},
	{Plan: "startup-128", Target: "1412", Amount: 3.58055},
	{Plan: "startup-225", Target: "1412", Amount: 5.65519},
	{Plan: "business-4", Target: "1412", Amount: 0.30787},
	{Plan: "business-8", Target: "1412", Amount: 0.60525},
	{Plan: "business-16", Target: "1412", Amount: 1.17123},
	{Plan: "business-32", Target: "1412", Amount: 2.1285},
	{Plan: "business-64", Target: "1412", Amount: 3.80662},
	{Plan: "business-128", Target: "1412", Amount: 7.30291},
	{Plan: "business-225", Target: "1412", Amount: 9.97887},
	{Plan: "premium-4", Target: "1412", Amount: 0.44811},
	{Plan: "premium-8", Target: "1412", Amount: 0.86957},
	{Plan: "premium-16", Target: "1412", Amount: 1.72469},
	{Plan: "premium-32", Target: "1412", Amount: 3.14683},
	{Plan: "premium-64", Target: "1412", Amount: 5.64105},
	{Plan: "premium-128", Target: "1412", Amount: 9.49136},
	{Plan: "premium-225", Target: "1412", Amount: 14.84892},
}

func generateMysqlProducts() []*db.Product {
	products := make([]*db.Product, 0, len(mysqlProductDBaaS))
	for _, p := range mysqlProductDBaaS {
		s := dbaasSourceString{
			Query:        queryDBaaSMysql,
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
