package reporting

import (
	"context"
	"fmt"
	"time"

	"github.com/appuio/appuio-cloud-reporting/pkg/db"
	"github.com/jmoiron/sqlx"
)

func fetchDateTime(ctx context.Context, tx *sqlx.Tx, timestamp time.Time) (*db.DateTime, error) {
	var dateTimes []db.DateTime
	err := sqlx.SelectContext(ctx, tx, &dateTimes, `SELECT date_times.* FROM date_times WHERE timestamp = $1`, timestamp)
	if err != nil {
		return nil, fmt.Errorf("cannot get timestamps by timestamp %s: %w", timestamp, err)
	}
	if len(dateTimes) == 0 {
		return nil, nil
	}
	return &dateTimes[0], nil
}

func EnsureDateTime(ctx context.Context, tx *sqlx.Tx, dt *db.DateTime) (*db.DateTime, error) {
	dateTime, err := fetchDateTime(ctx, tx, dt.Timestamp)
	if err != nil {
		return nil, err
	}
	if dateTime == nil {
		return createDateTime(tx, dt)
	}
	return dateTime, nil
}

func createDateTime(p db.NamedPreparer, in *db.DateTime) (*db.DateTime, error) {
	var dateTime db.DateTime
	err := db.GetNamed(p, &dateTime,
		"INSERT INTO date_times (timestamp, year, month, day, hour) VALUES (:timestamp, :year, :month, :day, :hour) RETURNING *", in)
	if err != nil {
		err = fmt.Errorf("cannot create datetime %v: %w", in, err)
	}
	return &dateTime, err
}

func NewDateTime(timestamp time.Time) *db.DateTime {
	timestamp = timestamp.In(time.UTC)
	return &db.DateTime{
		Timestamp: timestamp,
		Year:      timestamp.Year(),
		Month:     int(timestamp.Month()),
		Day:       timestamp.Day(),
		Hour:      timestamp.Hour(),
	}
}
