package tenantsmodel

import (
	"context"
	"fmt"
	"github.com/appuio/appuio-cloud-reporting/pkg/db"
	"github.com/jmoiron/sqlx"
)

func GetBySource(ctx context.Context, tx *sqlx.Tx, source string) (*db.Tenant, error) {
	var tenants []db.Tenant
	err := sqlx.SelectContext(ctx, tx, &tenants, `SELECT tenants.* FROM tenants WHERE source = $1 limit 1`, source)
	if err != nil {
		return nil, fmt.Errorf("cannot get tenants by source %s: %w", source, err)
	}
	if len(tenants) == 0 {
		return nil, nil
	}
	return &tenants[0], nil
}

func Ensure(ctx context.Context, tx *sqlx.Tx, ensureTenant *db.Tenant) (*db.Tenant, error) {
	tenant, err := GetBySource(ctx, tx, ensureTenant.Source)
	if err != nil {
		return nil, err
	}
	if tenant == nil {
		tenant, err = Create(tx, ensureTenant)
		if err != nil {
			return nil, err
		}
	}
	return tenant, nil
}

func Create(p db.NamedPreparer, in *db.Tenant) (*db.Tenant, error) {
	var tenant db.Tenant
	err := db.GetNamed(p, &tenant,
		"INSERT INTO tenants (source,target) VALUES (:source,:target) RETURNING *", in)
	if err != nil {
		err = fmt.Errorf("cannot create tenant %v: %w", in, err)
	}
	return &tenant, err
}
