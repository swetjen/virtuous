package db

import (
	"context"
	"errors"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
)

func NewTest() *Queries {
	return New(testDB{})
}

type testDB struct{}

type testRow struct{}

func (testRow) Scan(_ ...interface{}) error {
	return errors.New("db test row")
}

func (testDB) Exec(context.Context, string, ...interface{}) (pgconn.CommandTag, error) {
	return pgconn.CommandTag{}, errors.New("db test exec")
}

func (testDB) Query(context.Context, string, ...interface{}) (pgx.Rows, error) {
	return nil, errors.New("db test query")
}

func (testDB) QueryRow(context.Context, string, ...interface{}) pgx.Row {
	return testRow{}
}
