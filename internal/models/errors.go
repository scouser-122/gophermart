package models

import (
	"errors"
	"fmt"

	"github.com/jackc/pgerrcode"
	"github.com/jackc/pgx/v5/pgconn"
)

// CustomErr specifies custom app error with code and message
type CustomErr struct {
	// Code error code
	Code int

	// Message error message
	Message string
}

// Error func to implement error interface
func (e *CustomErr) Error() string {
	return fmt.Sprintf("code: %d", e.Code)
}

// Error codes
const (
	CustomErrUserRegisterFailed                = 10001
	CustomErrUserLoginBusy                     = 10002
	CustomErrUserLoginInvalidFormat            = 10003
	CustomErrUserPasswordInvalidFormat         = 10004
	CustomErrUserNotFound                      = 10005
	CustomErrUserLoginPasswordNotMatch         = 10006
	CustomErrOrderAlreadyUploaded              = 10007
	CustomErrOrderAlreadyUploadedByAnotherUser = 10008
	CustomErrOrderIDInvalidFormat              = 10009
	CustomErrUserOrdersListEmpty               = 10010
	CustomErrAccrualOrderNotRegistered         = 10011
	CustomErrAccrualTooManyRequests            = 10012
	CustomErrAccrualInternalServerError        = 10013
	CustomErrOrderNotFoundForWithdraw          = 10014
	CustomErrUserBalanceNotEnough              = 10015
	CustomErrWithdrawalsListEmpty              = 10016
	CustomErrGetOrderFromAccrualService        = 10017
)

// UnexpectedErrorMessage message for unexpecter app errors, usualy leads to returning InternalServerError
const UnexpectedErrorMessage = "unexpected error happen"

// ErrorClassification defines error class for specific handling
type ErrorClassification int

const (
	// NonRetryable non retryable errors
	NonRetryable ErrorClassification = iota
	// Retryable errors
	Retryable
)

// ClassifyPostgreSQLError detects class of error which may happen during Postgres DB interaction
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
