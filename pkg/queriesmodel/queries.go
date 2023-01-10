package queriesmodel

import (
	"context"
	"fmt"
	"github.com/appuio/appuio-cloud-reporting/pkg/db"
	"github.com/jmoiron/sqlx"
	"reflect"
)

func GetByName(ctx context.Context, tx *sqlx.Tx, name string) (*db.Query, error) {
	var queries []db.Query
	err := sqlx.SelectContext(ctx, tx, &queries, `SELECT queries.* FROM queries WHERE name = $1`, name)
	if err != nil {
		return nil, fmt.Errorf("cannot get queries by name %s: %w", name, err)
	}
	if len(queries) == 0 {
		return nil, nil
	}
	return &queries[0], nil
}

func Ensure(ctx context.Context, tx *sqlx.Tx, ensureQuery *db.Query) (*db.Query, error) {
	query, err := GetByName(ctx, tx, ensureQuery.Name)
	if err != nil {
		return nil, err
	}
	if query == nil {
		query, err = Create(tx, ensureQuery)
		if err != nil {
			return nil, err
		}
	} else {
		ensureQuery.Id = query.Id
		if !reflect.DeepEqual(query, ensureQuery) {
			fmt.Printf("updating query\n")
			err = Update(tx, ensureQuery)
			if err != nil {
				return nil, err
			}
		}
	}
	return query, nil
}

func Create(p db.NamedPreparer, in *db.Query) (*db.Query, error) {
	var query db.Query
	err := db.GetNamed(p, &query,
		"INSERT INTO queries (parent_id, name, description, query, unit, during) VALUES (:parent_id, :name, :description, :query, :unit, :during) RETURNING *", in)
	if err != nil {
		err = fmt.Errorf("cannot create query %v: %w", in, err)
	}
	return &query, err
}

func Update(p db.NamedPreparer, in *db.Query) error {
	var query db.Query
	err := db.GetNamed(p, &query,
		"UPDATE queries SET name=:name, description=:description, query=:query, unit=:unit, during=:during, parent_id=:parent_id WHERE id=:id RETURNING *", in)
	if err != nil {
		err = fmt.Errorf("cannot update query %v: %w", in, err)
	}
	return err
}
