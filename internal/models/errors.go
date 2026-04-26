package models

import (
	"errors"

	"github.com/jackc/pgerrcode"
	"github.com/jackc/pgx/v5/pgconn"
)

type ErrorLoginAlreadyTaken struct {
	Message string
	Err     error
}

func (e ErrorLoginAlreadyTaken) Error() string {
	return e.Message
}

var (
	ErrLoginAlreadyTaken = ErrorLoginAlreadyTaken{}
)

type ErrorClassification int

const (
	NonRetryable ErrorClassification = iota
	Retryable
)

func ClassifyPostgreSQLError(err error) ErrorClassification {
	var pgErr *pgconn.PgError
	if errors.As(err, &pgErr) {
		switch pgErr.Code {
		case pgerrcode.ConnectionException,
			pgerrcode.ConnectionDoesNotExist,
			pgerrcode.ConnectionFailure:
			return Retryable
		}
	}
	var pgConnectErr *pgconn.ConnectError
	if errors.As(err, &pgConnectErr) {
		return Retryable
	}
	return NonRetryable
}
