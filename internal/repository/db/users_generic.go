package db

import (
	"context"
	"errors"
	"time"

	"github.com/jackc/pgerrcode"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/scouser-122/gophermart/internal/logger"
	"github.com/scouser-122/gophermart/internal/models"
)

// GenericUserStorage implements UserStorage interface to store users data in Postgres DB
type GenericUserStorage struct {
	Database *PostgresDatabase
	repo     *GenericRepository[models.User]
}

// NewGenericUserStorage creates generic users storage
func NewGenericUserStorage(db *PostgresDatabase) *GenericUserStorage {
	mapper := func(row pgx.Row) (*models.User, error) {
		var user models.User
		err := row.Scan(
			&user.Login,
			&user.Password,
			&user.Balance,
			&user.CreatedAt,
		)
		if err != nil {
			return nil, err
		}
		return &user, err
	}
	return &GenericUserStorage{
		Database: db,
		repo:     NewGenericRepository(db, "users", "login", mapper),
	}
}

// Create creates new user,
// returns error if user with specified login already exists or process failed
func (s *GenericUserStorage) Create(ctx context.Context, login string, password string) (*models.User, error) {
	logger := logger.GetLoggerFromContext(ctx)
	user, err := s.repo.Create(ctx, "login,password,created_at", login, password, time.Now())
	if err != nil {
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) {
			if pgErr.Code == pgerrcode.UniqueViolation {
				err = &models.CustomErr{Code: models.CustomErrUserLoginBusy}
				logger.Sugar().Error(err)
				return nil, err
			}
		}
		logger.Sugar().Error(err)
		return nil, err
	}
	return user, nil
}

// Get obtains user from storage
func (s *GenericUserStorage) Get(ctx context.Context, login string) (*models.User, error) {
	logger := logger.GetLoggerFromContext(ctx)
	user, err := s.repo.GetByID(ctx, login)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			err = &models.CustomErr{Code: models.CustomErrUserNotFound}
			logger.Sugar().Error(err)
			return nil, err
		}
		logger.Sugar().Error(err)
		return nil, err
	}
	return user, nil
}

// AddBalance adds specified accrual to user's balance
func (s *GenericUserStorage) AddBalance(ctx context.Context, login string, accrual *float32) error {
	logger := logger.GetLoggerFromContext(ctx)
	repo := s.repo
	tx := models.GetTransactionFromContext(ctx)
	if tx != nil {
		repo = s.repo.WithTx(tx.(pgx.Tx))
	}
	_, err := repo.Update(ctx, "UPDATE users SET balance = balance + $1 WHERE login = $2", accrual, login)
	if err != nil {
		logger.Sugar().Error(err)
		return err
	}
	return err
}

// WithdrawBalance withdraws specified sum from user's balance
func (s *GenericUserStorage) WithdrawBalance(ctx context.Context, login string, sum float32) error {
	logger := logger.GetLoggerFromContext(ctx)
	repo := s.repo
	tx := models.GetTransactionFromContext(ctx)
	if tx != nil {
		repo = s.repo.WithTx(tx.(pgx.Tx))
	}
	tag, err := repo.Update(ctx, "UPDATE users SET balance = balance - $1 WHERE login = $2 AND balance >= $1", sum, login)
	if err != nil {
		logger.Sugar().Error(err)
		return err
	}
	if tag.RowsAffected() == 0 {
		err = &models.CustomErr{Code: models.CustomErrUserBalanceNotEnough}
		logger.Sugar().Error(err)
		return err
	}
	return err
}
