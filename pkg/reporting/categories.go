package reporting

import (
	"context"
	"fmt"

	"github.com/appuio/appuio-cloud-reporting/pkg/db"
	"github.com/jmoiron/sqlx"
)

func fetchCategory(ctx context.Context, tx *sqlx.Tx, source string) (*db.Category, error) {
	var categories []db.Category
	err := sqlx.SelectContext(ctx, tx, &categories, `SELECT categories.* FROM categories WHERE source = $1`, source)
	if err != nil {
		return nil, fmt.Errorf("cannot get categories by source %s: %w", source, err)
	}
	if len(categories) == 0 {
		return nil, nil
	}
	return &categories[0], nil
}

func EnsureCategory(ctx context.Context, tx *sqlx.Tx, cat *db.Category) (*db.Category, error) {
	category, err := fetchCategory(ctx, tx, cat.Source)
	if err != nil {
		return nil, err
	}
	if category == nil {
		return createCategory(tx, cat)
	}
	return category, nil
}

func createCategory(p db.NamedPreparer, in *db.Category) (*db.Category, error) {
	var category db.Category
	err := db.GetNamed(p, &category,
		"INSERT INTO categories (source,target) VALUES (:source,:target) RETURNING *", in)
	if err != nil {
		err = fmt.Errorf("cannot create category %v: %w", in, err)
	}
	return &category, err
}
