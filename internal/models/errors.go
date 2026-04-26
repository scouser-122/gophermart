package models

import (
	"errors"
	"fmt"

	"github.com/jackc/pgerrcode"
	"github.com/jackc/pgx/v5/pgconn"
)

type CustomErr struct {
	Code    int
	Message string
}

func (e *CustomErr) Error() string {
	return fmt.Sprintf("code: %d", e.Code)
}

const (
	CustomErrRegisterFailed = iota
	CustomErrLoginBusy
	CustomErrLoginInvalidFormat
	CustomErrUserNotFound
)

const UnexpectedErrorMessage = "unexpected error happen"

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
