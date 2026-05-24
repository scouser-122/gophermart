package db

import (
	"context"
	"errors"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/scouser-122/gophermart/internal/config"
	"github.com/scouser-122/gophermart/internal/logger"
	"github.com/scouser-122/gophermart/internal/models"
	"github.com/scouser-122/gophermart/internal/utils"
	"go.uber.org/zap"
)

// PostgresOrderStorage implements OrderStorage interface to store orders data in Postgres DB
type PostgresOrderStorage struct {
	Database *PostgresDatabase
}

// AddOrder adds specified order, links it with user, and increments user's balance if accrual present,
// returns error if order with specified ID already exists or add process failed
func (storage *PostgresOrderStorage) AddOrder(ctx context.Context, order *models.Order, userLogin string) error {
	logger := logger.GetLoggerFromContext(ctx)
	var dbOrder models.Order
	err := storage.Database.Select(ctx, &dbOrder, "SELECT * FROM orders WHERE id = $1", order.ID)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			order.UploadedAt = time.Now()
			order.UserLogin = userLogin
			tx, err := storage.Database.Begin(ctx)
			if err != nil {
				logger.Sugar().Error(zap.Error(err))
				return err
			}
			defer tx.Rollback(ctx)
			_, err = tx.Exec(
				ctx,
				"INSERT INTO orders (id,status,uploaded_at,accrual,user_login) VALUES ($1,$2,$3,$4,$5)",
				order.ID, order.Status, order.UploadedAt, order.Accrual, order.UserLogin,
			)
			if err != nil {
				logger.Sugar().Error(zap.Error(err))
				return err
			}
			if order.Accrual != nil {
				_, err := tx.Exec(ctx, "UPDATE users SET balance = balance + $1 WHERE login = $2", order.Accrual, order.UserLogin)
				if err != nil {
					logger.Sugar().Error(zap.Error(err))
					return err
				}
			}
			err = config.DataBaseRequestRetry(
				ctx,
				storage.Database.Config.RetryConfig,
				func() error {
					return tx.Commit(ctx)
				},
			)
			if err != nil {
				logger.Sugar().Error(zap.Error(err))
				return err
			}
			logger.Sugar().Info("order stored in DB", zap.Any("order", *order))
			return nil
		} else {
			logger.Sugar().Error(zap.Error(err))
			return err
		}
	}
	if dbOrder.UserLogin == userLogin {
		err = &models.CustomErr{Code: models.CustomErrOrderAlreadyUploaded}
		logger.Sugar().Error(err)
		return err
	} else {
		err = &models.CustomErr{Code: models.CustomErrOrderAlreadyUploadedByAnotherUser}
		logger.Sugar().Error(err)
		return err
	}
}

// GetUserOrders returns slice of orders for user with specified login
func (storage *PostgresOrderStorage) GetUserOrders(ctx context.Context, userLogin string) ([]*models.Order, error) {
	result := []*models.Order{}
	page := 0
	limit := 10
	logger := logger.GetLoggerFromContext(ctx)

	for {
		offset := page * limit
		rows, err := storage.Database.Query(
			ctx,
			"SELECT id, status, uploaded_at, accrual, user_login FROM orders WHERE user_login = $1 ORDER BY uploaded_at DESC LIMIT $2 OFFSET $3",
			userLogin, limit, offset,
		)
		if err != nil {
			if errors.Is(err, pgx.ErrNoRows) {
				break
			}
			logger.Error(err.Error())
			return []*models.Order{}, err
		}
		count := 0
		for rows.Next() {
			dbOrder := models.Order{}
			err = rows.Scan(&dbOrder.ID, &dbOrder.Status, &dbOrder.UploadedAt, &dbOrder.Accrual, &dbOrder.UserLogin)
			if err != nil {
				if errors.Is(err, pgx.ErrNoRows) {
					break
				}
				logger.Error(err.Error())
				rows.Close()
				return []*models.Order{}, err
			}
			result = append(result, &dbOrder)
			count++
		}
		err = rows.Err()
		if err != nil {
			logger.Error(err.Error())
			rows.Close()
			return []*models.Order{}, err
		}
		rows.Close()
		if count == 0 {
			break
		}
		page++
	}

	return result, nil
}

// GetOrder returns order for specified ID or nil id order was not uploaded
func (storage *PostgresOrderStorage) GetOrder(ctx context.Context, orderID string) (*models.Order, error) {
	logger := logger.GetLoggerFromContext(ctx)
	var dbOrder models.Order
	err := storage.Database.Select(ctx, &dbOrder, "SELECT * FROM orders WHERE id = $1", orderID)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, nil
		}
		logger.Sugar().Error(err)
		return nil, err
	}
	return &dbOrder, nil
}

// UpdateOrder updates order data and increments user's balance if accrual present
func (storage *PostgresOrderStorage) UpdateOrder(ctx context.Context, order *models.Order) (*models.Order, error) {
	logger := logger.GetLoggerFromContext(ctx)
	var dbOrder models.Order
	err := storage.Database.Select(ctx, &dbOrder, "SELECT * FROM orders WHERE id = $1", order.ID)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			err = &models.CustomErr{Code: models.CustomErrOrderNotFoundForWithdraw}
			logger.Sugar().Error(err)
			return nil, err
		}
		logger.Sugar().Error(err)
		return nil, err
	}
	if dbOrder.Status == order.Status && utils.EqualFloat32Ptr(dbOrder.Accrual, order.Accrual, 1e-6) {
		return &dbOrder, nil
	}
	dbOrder.Status = order.Status
	dbOrder.Accrual = order.Accrual
	tx, err := storage.Database.Begin(ctx)
	if err != nil {
		logger.Sugar().Error(zap.Error(err))
		return nil, err
	}
	defer tx.Rollback(ctx)
	_, err = tx.Exec(
		ctx,
		"UPDATE orders SET status = $1, accrual = $2 WHERE id = $3",
		dbOrder.Status, dbOrder.Accrual, dbOrder.ID,
	)
	if err != nil {
		logger.Sugar().Error(zap.Error(err))
		return nil, err
	}
	if order.Accrual != nil {
		_, err := tx.Exec(ctx, "UPDATE users SET balance = balance + $1 WHERE login = $2", dbOrder.Accrual, dbOrder.UserLogin)
		if err != nil {
			logger.Sugar().Error(zap.Error(err))
			return nil, err
		}
	}
	err = config.DataBaseRequestRetry(
		ctx,
		storage.Database.Config.RetryConfig,
		func() error {
			return tx.Commit(ctx)
		},
	)
	if err != nil {
		logger.Sugar().Error(zap.Error(err))
		return nil, err
	}
	logger.Sugar().Info("order updated in DB", zap.Any("order", dbOrder))
	return &dbOrder, nil
}

// GetWithdrawnForUser return total withdrawn points from all orders for specified user
func (storage *PostgresOrderStorage) GetWithdrawnForUser(ctx context.Context, login string) (float32, error) {
	logger := logger.GetLoggerFromContext(ctx)
	var result *float32
	row := storage.Database.QueryRow(ctx, "SELECT SUM(withdrawn) FROM orders WHERE user_login = $1", login)
	err := config.DataBaseRequestRetry(
		ctx,
		storage.Database.Config.RetryConfig,
		func() error {
			return row.Scan(&result)
		},
	)
	if err != nil {
		logger.Sugar().Error(err)
		return 0.0, err
	}
	if result == nil {
		return 0.0, nil
	}
	return *result, nil
}

// WithdrawBalanceForOrder withdraw user's loyalty points from balance for order with specified ID
func (storage *PostgresOrderStorage) WithdrawBalanceForOrder(ctx context.Context, orderID string, login string, sum float32) error {
	logger := logger.GetLoggerFromContext(ctx)
	tx, err := storage.Database.Begin(ctx)
	if err != nil {
		logger.Sugar().Error(err)
		return err
	}
	defer tx.Rollback(ctx)
	commandTag, err := tx.Exec(
		ctx,
		"UPDATE users SET balance = balance - $1 WHERE login = $2 AND balance >= $1",
		sum, login,
	)
	if err != nil {
		logger.Sugar().Error(err)
		return err
	}
	if commandTag.RowsAffected() == 0 {
		err = &models.CustomErr{Code: models.CustomErrUserBalanceNotEnough}
		logger.Sugar().Error(err)
		return err
	}
	commandTag, err = tx.Exec(
		ctx,
		"UPDATE orders SET withdrawn = COALESCE(withdrawn, 0.0) + $1, processed_at = CURRENT_TIMESTAMP WHERE id = $2 AND user_login = $3 AND (status = 'NEW' OR status = 'PROCESSING')",
		sum, orderID, login,
	)
	if err != nil {
		logger.Sugar().Error(err)
		return err
	}
	if commandTag.RowsAffected() == 0 {
		err = &models.CustomErr{Code: models.CustomErrOrderNotFoundForWithdraw}
		logger.Sugar().Error(err)
		return err
	}
	err = config.DataBaseRequestRetry(
		ctx,
		storage.Database.Config.RetryConfig,
		func() error {
			return tx.Commit(ctx)
		},
	)
	if err != nil {
		logger.Sugar().Error(err)
		return err
	}
	logger.Sugar().Info("balance withdrawed for order", zap.String("order", orderID), zap.Float32("sum", sum))
	return nil
}

// WithdrawalsForUser returns slice of withdrawals data for specified user
func (storage *PostgresOrderStorage) WithdrawalsForUser(ctx context.Context, login string) ([]models.WithdrawalResponse, error) {
	logger := logger.GetLoggerFromContext(ctx)
	result := []models.WithdrawalResponse{}

	page := 0
	limit := 10
	for {
		offset := page * limit
		rows, err := storage.Database.Query(
			ctx,
			"SELECT id, withdrawn, processed_at FROM orders WHERE user_login = $1 AND withdrawn IS NOT NULL ORDER BY processed_at DESC LIMIT $2 OFFSET $3",
			login, limit, offset,
		)
		if err != nil {
			if errors.Is(err, pgx.ErrNoRows) {
				break
			}
			logger.Error(err.Error())
			return []models.WithdrawalResponse{}, err
		}
		count := 0
		for rows.Next() {
			withdrawal := models.WithdrawalResponse{}
			err = rows.Scan(&withdrawal.Order, &withdrawal.Sum, &withdrawal.ProcessedAt)
			if err != nil {
				if errors.Is(err, pgx.ErrNoRows) {
					break
				}
				logger.Error(err.Error())
				rows.Close()
				return []models.WithdrawalResponse{}, err
			}
			result = append(result, withdrawal)
			count++
		}
		err = rows.Err()
		if err != nil {
			logger.Error(err.Error())
			rows.Close()
			return []models.WithdrawalResponse{}, err
		}
		rows.Close()
		if count == 0 {
			break
		}
		page++
	}

	return result, nil
}
