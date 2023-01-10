package reporting

import (
	"context"
	"fmt"
	"reflect"
	"time"

	"github.com/appuio/appuio-cloud-reporting/pkg/db"
	"github.com/jmoiron/sqlx"
)

func getProductBySource(ctx context.Context, tx *sqlx.Tx, source string) (*db.Product, error) {
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

func getProductBySourceQueryAndTime(ctx context.Context, tx *sqlx.Tx, sourceQuery string, timestamp time.Time) ([]db.Product, error) {
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

func GetBestMatchingProduct(ctx context.Context, tx *sqlx.Tx, source string, timestamp time.Time) (*db.Product, error) {
	tokenizedSource := NewTokenizedSource(source)
	candidateProducts, err := getProductBySourceQueryAndTime(ctx, tx, tokenizedSource.Tokens[0], timestamp)
	if err != nil {
		return nil, err
	}

	candidateSourcePatterns := make([]*TokenizedSource, len(candidateProducts))
	for i, candidateProduct := range candidateProducts {
		candidateSourcePatterns[i] = NewTokenizedSource(candidateProduct.Source)
	}

	match := FindBestMatchingTokenizedSource(tokenizedSource, candidateSourcePatterns)

	for _, candidateProduct := range candidateProducts {
		if candidateProduct.Source == match.String() {
			return &candidateProduct, nil
		}
	}

	return nil, nil
}

func EnsureProduct(ctx context.Context, tx *sqlx.Tx, ensureProduct *db.Product) (*db.Product, error) {
	product, err := getProductBySource(ctx, tx, ensureProduct.Source)
	if err != nil {
		return nil, err
	}
	if product == nil {
		product, err = createProduct(tx, ensureProduct)
		if err != nil {
			return nil, err
		}
	} else {
		ensureProduct.Id = product.Id
		if !reflect.DeepEqual(product, ensureProduct) {
			fmt.Printf("updating product\n") // FIXME(mw): logger
			err = updateProduct(tx, ensureProduct)
			if err != nil {
				return nil, err
			}
		}
	}
	return product, nil
}

func createProduct(p db.NamedPreparer, in *db.Product) (*db.Product, error) {
	var product db.Product
	err := db.GetNamed(p, &product,
		"INSERT INTO products (source,target,amount,unit,during) VALUES (:source,:target,:amount,:unit,:during) RETURNING *", in)
	if err != nil {
		err = fmt.Errorf("cannot create product %v: %w", in, err)
	}
	return &product, err
}

func updateProduct(p db.NamedPreparer, in *db.Product) error {
	var product db.Product
	err := db.GetNamed(p, &product,
		"UPDATE products SET source=:source, target=:target, amount=:amount, unit=:unit, during=:during WHERE id=:id RETURNING *", in)
	if err != nil {
		err = fmt.Errorf("cannot update product %v: %w", in, err)
	}
	return err
}
