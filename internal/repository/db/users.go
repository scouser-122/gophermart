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

// PostgresUserStorage implements UserStorage interface to store users data in Postgres DB
type PostgresUserStorage struct {
	Database *PostgresDatabase
	repo     *GenericRepository[models.User]
}

// NewPostgresUserStorage creates Postgres users storage
func NewPostgresUserStorage(db *PostgresDatabase) *PostgresUserStorage {
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
	return &PostgresUserStorage{
		Database: db,
		repo:     NewGenericRepository(db, "users", "login", mapper),
	}
}

// Create creates new user,
// returns error if user with specified login already exists or process failed
func (s *PostgresUserStorage) Create(ctx context.Context, login string, password string) (*models.User, error) {
	logger := logger.GetSlogLoggerFromContext(ctx)
	user, err := s.repo.Create(ctx, "login,password,created_at", login, password, time.Now())
	if err != nil {
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) {
			if pgErr.Code == pgerrcode.UniqueViolation {
				err = &models.CustomErr{Code: models.CustomErrUserLoginBusy}
				logger.Error(err.Error())
				return nil, err
			}
		}
		logger.Error(err.Error())
		return nil, err
	}
	return user, nil
}

// Get obtains user from storage
func (s *PostgresUserStorage) Get(ctx context.Context, login string) (*models.User, error) {
	logger := logger.GetSlogLoggerFromContext(ctx)
	user, err := s.repo.GetByID(ctx, login)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			err = &models.CustomErr{Code: models.CustomErrUserNotFound}
			logger.Error(err.Error())
			return nil, err
		}
		logger.Error(err.Error())
		return nil, err
	}
	return user, nil
}

// AddBalance adds specified accrual to user's balance
func (s *PostgresUserStorage) AddBalance(ctx context.Context, login string, accrual *float32) error {
	logger := logger.GetSlogLoggerFromContext(ctx)
	repo := s.repo
	tx := models.GetTransactionFromContext(ctx)
	if tx != nil {
		repo = s.repo.WithTx(tx.(pgx.Tx))
	}
	_, err := repo.Update(ctx, "UPDATE users SET balance = balance + $1 WHERE login = $2", accrual, login)
	if err != nil {
		logger.Error(err.Error())
		return err
	}
	return err
}

// WithdrawBalance withdraws specified sum from user's balance
func (s *PostgresUserStorage) WithdrawBalance(ctx context.Context, login string, sum float32) error {
	logger := logger.GetSlogLoggerFromContext(ctx)
	repo := s.repo
	tx := models.GetTransactionFromContext(ctx)
	if tx != nil {
		repo = s.repo.WithTx(tx.(pgx.Tx))
	}
	tag, err := repo.Update(ctx, "UPDATE users SET balance = balance - $1 WHERE login = $2 AND balance >= $1", sum, login)
	if err != nil {
		logger.Error(err.Error())
		return err
	}
	if tag.RowsAffected() == 0 {
		err = &models.CustomErr{Code: models.CustomErrUserBalanceNotEnough}
		logger.Error(err.Error())
		return err
	}
	return err
}
