package datetimesmodel

import (
	"context"
	"fmt"
	"github.com/appuio/appuio-cloud-reporting/pkg/db"
	"github.com/jmoiron/sqlx"
	"time"
)

func GetByTimestamp(ctx context.Context, tx *sqlx.Tx, timestamp time.Time) (*db.DateTime, error) {
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

func Ensure(ctx context.Context, tx *sqlx.Tx, ensureDateTime *db.DateTime) (*db.DateTime, error) {
	dateTime, err := GetByTimestamp(ctx, tx, ensureDateTime.Timestamp)
	if err != nil {
		return nil, err
	}
	if dateTime == nil {
		dateTime, err = Create(tx, ensureDateTime)
		if err != nil {
			return nil, err
		}
	}
	return dateTime, nil
}

func Create(p db.NamedPreparer, in *db.DateTime) (*db.DateTime, error) {
	var dateTime db.DateTime
	err := db.GetNamed(p, &dateTime,
		"INSERT INTO date_times (timestamp, year, month, day, hour) VALUES (:timestamp, :year, :month, :day, :hour) RETURNING *", in)
	if err != nil {
		err = fmt.Errorf("cannot create datetime %v: %w", in, err)
	}
	return &dateTime, err
}

func New(timestamp time.Time) *db.DateTime {
	timestamp = timestamp.In(time.UTC)
	return &db.DateTime{
		Timestamp: timestamp,
		Year:      timestamp.Year(),
		Month:     int(timestamp.Month()),
		Day:       timestamp.Day(),
		Hour:      timestamp.Hour(),
	}
}
