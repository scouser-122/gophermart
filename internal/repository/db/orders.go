package db

import (
	"context"
	"errors"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/scouser-122/gophermart/internal/logger"
	"github.com/scouser-122/gophermart/internal/models"
)

// PostgresOrderStorage implements OrderStorage interface to work with orders data in Postgres DB
type PostgresOrderStorage struct {
	Database *PostgresDatabase
	repo     *GenericRepository[models.Order]
}

const ordersPageSize = 10

// NewPostgresOrderStorage creates Postgres order storage
func NewPostgresOrderStorage(db *PostgresDatabase) *PostgresOrderStorage {
	mapper := func(row pgx.Row) (*models.Order, error) {
		var order models.Order
		err := row.Scan(
			&order.ID,
			&order.Status,
			&order.UploadedAt,
			&order.Accrual,
			&order.Withdrawn,
			&order.ProcessedAt,
			&order.UserLogin,
		)
		if err != nil {
			return nil, err
		}
		return &order, err
	}
	return &PostgresOrderStorage{
		Database: db,
		repo:     NewGenericRepository(db, "orders", "id", mapper),
	}
}

// Create creates new order,
// returns error if order with specified ID already exists or process failed
func (s *PostgresOrderStorage) Create(
	ctx context.Context,
	ID string,
	status models.OrderStatus,
	accrual *float32,
	userLogin string,
) (*models.Order, error) {
	logger := logger.GetSlogLoggerFromContext(ctx)
	repo := s.repo
	tx := models.GetTransactionFromContext(ctx)
	if tx != nil {
		repo = s.repo.WithTx(tx.(pgx.Tx))
	}
	order, err := repo.Create(ctx, "id,status,uploaded_at,accrual,user_login", ID, status, time.Now(), accrual, userLogin)
	if err != nil {
		logger.Error(err.Error())
		return nil, err
	}
	return order, nil
}

// Get obtains order from storage
func (s *PostgresOrderStorage) Get(ctx context.Context, ID string) (*models.Order, error) {
	logger := logger.GetSlogLoggerFromContext(ctx)
	order, err := s.repo.GetByID(ctx, ID)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			err = &models.CustomErr{Code: models.CustomErrOrderNotFound}
			return nil, err
		}
		logger.Error(err.Error())
		return nil, err
	}
	return order, nil
}

// Update updates order data
func (s *PostgresOrderStorage) Update(ctx context.Context, order *models.Order, userLogin string) error {
	logger := logger.GetSlogLoggerFromContext(ctx)
	repo := s.repo
	tx := models.GetTransactionFromContext(ctx)
	if tx != nil {
		repo = s.repo.WithTx(tx.(pgx.Tx))
	}
	_, err := repo.Update(ctx, "UPDATE orders SET status = $1, accrual = $2 WHERE id = $3", order.Status, order.Accrual, userLogin)
	if err != nil {
		logger.Error(err.Error())
		return err
	}
	return err
}

// GetAllForUser returns all orders for specified user
func (s *PostgresOrderStorage) GetAllForUser(ctx context.Context, userLogin string) ([]*models.Order, error) {
	logger := logger.GetSlogLoggerFromContext(ctx)
	result := []*models.Order{}
	for orders, err := range s.repo.GetAllConditional(
		ctx,
		"WHERE user_login = $1",
		[]any{userLogin},
		"uploaded_at DESC",
		ordersPageSize,
	) {
		if err != nil {
			logger.Error(err.Error())
			return []*models.Order{}, err
		}
		result = append(result, orders...)
	}
	return result, nil
}

// GetWithdrawnForUser return total withdrawn points from all orders for specified user
func (s *PostgresOrderStorage) GetWithdrawnForUser(ctx context.Context, login string) (float32, error) {
	logger := logger.GetSlogLoggerFromContext(ctx)
	var result *float32
	mapper := func(row pgx.Row) error {
		return row.Scan(&result)
	}
	err := s.repo.CustomQuery(ctx, mapper, "SELECT SUM(withdrawn) FROM orders WHERE user_login = $1", login)
	if err != nil {
		logger.Error(err.Error())
		return 0.0, err
	}
	if result == nil {
		return 0.0, nil
	}
	return *result, nil
}

// WithdrawSum set's withdrawn amount for specified order
func (s *PostgresOrderStorage) WithdrawSum(ctx context.Context, ID string, sum float32, login string) error {
	logger := logger.GetSlogLoggerFromContext(ctx)
	repo := s.repo
	tx := models.GetTransactionFromContext(ctx)
	if tx != nil {
		repo = s.repo.WithTx(tx.(pgx.Tx))
	}
	tag, err := repo.Update(
		ctx,
		"UPDATE orders SET withdrawn = COALESCE(withdrawn, 0.0) + $1, processed_at = CURRENT_TIMESTAMP WHERE id = $2 AND user_login = $3 AND (status = 'NEW' OR status = 'PROCESSING')",
		sum, ID, login,
	)
	if err != nil {
		logger.Error(err.Error())
		return err
	}
	if tag.RowsAffected() == 0 {
		err = &models.CustomErr{Code: models.CustomErrOrderNotFoundForWithdraw}
		logger.Error(err.Error())
		return err
	}
	return err
}

// WithdrawalsForUser returns slice of withdrawals data for specified user
func (s *PostgresOrderStorage) WithdrawalsForUser(ctx context.Context, userLogin string) ([]models.WithdrawalResponse, error) {
	logger := logger.GetSlogLoggerFromContext(ctx)
	result := []models.WithdrawalResponse{}
	for orders, err := range s.repo.GetAllConditional(
		ctx,
		"WHERE user_login = $1 AND withdrawn IS NOT NULL",
		[]any{userLogin},
		"processed_at DESC",
		ordersPageSize,
	) {
		if err != nil {
			logger.Error(err.Error())
			return []models.WithdrawalResponse{}, err
		}
		for _, o := range orders {
			result = append(result, models.WithdrawalResponse{
				Order:       o.ID,
				Sum:         *o.Withdrawn,
				ProcessedAt: *o.ProcessedAt,
			})
		}
	}
	return result, nil
}
