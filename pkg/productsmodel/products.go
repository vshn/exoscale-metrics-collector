package productsmodel

import (
	"context"
	"fmt"
	"reflect"
	"time"

	"github.com/appuio/appuio-cloud-reporting/pkg/db"
	"github.com/jmoiron/sqlx"
	"github.com/vshn/exoscale-metrics-collector/pkg/tokenmatcher"
)

func GetBySource(ctx context.Context, tx *sqlx.Tx, source string) (*db.Product, error) {
	var products []db.Product
	err := sqlx.SelectContext(ctx, tx, &products, `SELECT products.* FROM products WHERE source = $1`, source)
	if err != nil {
		return nil, err
	}
	if len(products) == 0 {
		return nil, nil
	}
	return &products[0], nil
}

func getBySourceQueryAndTime(ctx context.Context, tx *sqlx.Tx, sourceQuery string, timestamp time.Time) ([]db.Product, error) {
	var products []db.Product
	err := sqlx.SelectContext(ctx, tx, &products,
		`SELECT products.* FROM products 
                  WHERE (source = $1 OR source LIKE $2)
                  AND during @> $3::timestamptz`,
		sourceQuery, sourceQuery+":%", timestamp)
	if err != nil {
		return nil, fmt.Errorf("cannot get products by sourceQuery %s and timestamp %s: %w", sourceQuery, timestamp, err)
	}
	return products, nil
}

func GetBestMatch(ctx context.Context, tx *sqlx.Tx, source string, timestamp time.Time) (*db.Product, error) {
	tokenizedSource := tokenmatcher.NewTokenizedSource(source)
	candidateProducts, err := getBySourceQueryAndTime(ctx, tx, tokenizedSource.Tokens[0], timestamp)
	if err != nil {
		return nil, err
	}

	candidateSourcePatterns := make([]*tokenmatcher.TokenizedSource, len(candidateProducts))
	for i, candidateProduct := range candidateProducts {
		candidateSourcePatterns[i] = tokenmatcher.NewTokenizedSource(candidateProduct.Source)
	}

	match := tokenmatcher.FindBestMatch(tokenizedSource, candidateSourcePatterns)

	for _, candidateProduct := range candidateProducts {
		if candidateProduct.Source == match.String() {
			return &candidateProduct, nil
		}
	}

	return nil, nil
}

func Ensure(ctx context.Context, tx *sqlx.Tx, ensureProduct *db.Product) (*db.Product, error) {
	product, err := GetBySource(ctx, tx, ensureProduct.Source)
	if err != nil {
		return nil, err
	}
	if product == nil {
		product, err = Create(tx, ensureProduct)
		if err != nil {
			return nil, err
		}
	} else {
		ensureProduct.Id = product.Id
		if !reflect.DeepEqual(product, ensureProduct) {
			fmt.Printf("updating product\n")
			err = Update(tx, ensureProduct)
			if err != nil {
				return nil, err
			}
		}
	}
	return product, nil
}

func Create(p db.NamedPreparer, in *db.Product) (*db.Product, error) {
	var product db.Product
	err := db.GetNamed(p, &product,
		"INSERT INTO products (source,target,amount,unit,during) VALUES (:source,:target,:amount,:unit,:during) RETURNING *", in)
	if err != nil {
		err = fmt.Errorf("cannot create product %v: %w", in, err)
	}
	return &product, err
}

func Update(p db.NamedPreparer, in *db.Product) error {
	var product db.Product
	err := db.GetNamed(p, &product,
		"UPDATE products SET source=:source, target=:target, amount=:amount, unit=:unit, during=:during WHERE id=:id RETURNING *", in)
	if err != nil {
		err = fmt.Errorf("cannot update product %v: %w", in, err)
	}
	return err
}
