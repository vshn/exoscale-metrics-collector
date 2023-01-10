package reporting

import (
	"context"
	"fmt"
	"reflect"
	"time"

	"github.com/appuio/appuio-cloud-reporting/pkg/db"
	"github.com/jmoiron/sqlx"
)

func fetchDiscount(ctx context.Context, tx *sqlx.Tx, source string) (*db.Discount, error) {
	var discounts []db.Discount
	err := sqlx.SelectContext(ctx, tx, &discounts, `SELECT discounts.* FROM discounts WHERE source = $1`, source)
	if err != nil {
		return nil, fmt.Errorf("cannot get discounts by source %s: %w", source, err)
	}
	if len(discounts) == 0 {
		return nil, nil
	}
	return &discounts[0], nil
}

func fetchDiscountBySourceQueryAndTime(ctx context.Context, tx *sqlx.Tx, sourceQuery string, timestamp time.Time) ([]db.Discount, error) {
	var discounts []db.Discount
	err := sqlx.SelectContext(ctx, tx, &discounts,
		`SELECT discounts.* FROM discounts 
                  WHERE (source = $1 OR source LIKE $2)
                  AND during @> $3::timestamptz`,
		sourceQuery, sourceQuery+":%", timestamp)
	if err != nil {
		return nil, fmt.Errorf("cannot get discounts by sourceQuery %s and timestamp %s: %w", sourceQuery, timestamp, err)
	}
	return discounts, nil
}

func GetBestMatchingDiscount(ctx context.Context, tx *sqlx.Tx, source string, timestamp time.Time) (*db.Discount, error) {
	tokenizedSource := NewTokenizedSource(source)
	candidateDiscounts, err := fetchDiscountBySourceQueryAndTime(ctx, tx, tokenizedSource.Tokens[0], timestamp)
	if err != nil {
		return nil, err
	}

	candidateSourcePatterns := make([]*TokenizedSource, len(candidateDiscounts))
	for i, candidateDiscount := range candidateDiscounts {
		candidateSourcePatterns[i] = NewTokenizedSource(candidateDiscount.Source)
	}

	match := FindBestMatchingTokenizedSource(tokenizedSource, candidateSourcePatterns)

	for _, candidateDiscount := range candidateDiscounts {
		if candidateDiscount.Source == match.String() {
			return &candidateDiscount, nil
		}
	}

	return nil, nil
}

func EnsureDiscount(ctx context.Context, tx *sqlx.Tx, ensureDiscount *db.Discount) (*db.Discount, error) {
	discount, err := fetchDiscount(ctx, tx, ensureDiscount.Source)
	if err != nil {
		return nil, err
	}
	if discount == nil {
		discount, err = createDiscount(tx, ensureDiscount)
		if err != nil {
			return nil, err
		}
	} else {
		ensureDiscount.Id = discount.Id
		if !reflect.DeepEqual(discount, ensureDiscount) {
			fmt.Printf("updating discount\n") // FIXME(mw): logger
			err = updateDiscount(tx, ensureDiscount)
			if err != nil {
				return nil, err
			}
		}
	}
	return discount, nil
}

func createDiscount(p db.NamedPreparer, in *db.Discount) (*db.Discount, error) {
	var discount db.Discount
	err := db.GetNamed(p, &discount,
		"INSERT INTO discounts (source,discount,during) VALUES (:source,:discount,:during) RETURNING *", in)
	if err != nil {
		err = fmt.Errorf("cannot create discount %v: %w", in, err)
	}
	return &discount, err
}

func updateDiscount(p db.NamedPreparer, in *db.Discount) error {
	var discount db.Discount
	err := db.GetNamed(p, &discount,
		"UPDATE discounts SET source=:source, discount=:target, during=:during WHERE id=:id RETURNING *", in)
	if err != nil {
		err = fmt.Errorf("cannot update discount %v: %w", in, err)
	}
	return err
}
