package db

import (
	"context"
	"errors"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/scouser-122/gophermart/internal/logger"
	"github.com/scouser-122/gophermart/internal/models"
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
			if order.Status == models.RegisteredOrder {
				order.Status = models.NewOrder
			}
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
			err = tx.Commit(ctx)
			if err != nil {
				logger.Sugar().Error(zap.Error(err))
				return err
			}
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

// GetWithdrawnForUser return total withdrawn points from all orders for specified user
func (storage *PostgresOrderStorage) GetWithdrawnForUser(ctx context.Context, login string) (float32, error) {
	logger := logger.GetLoggerFromContext(ctx)
	var result *float32
	row, err := storage.Database.QueryRow(ctx, "SELECT SUM(withdrawn) FROM orders WHERE user_login = $1", login)
	if err != nil {
		logger.Sugar().Error(err)
		return 0.0, err
	}
	err = row.Scan(&result)
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
	row := tx.QueryRow(ctx, "SELECT id FROM orders WHERE id = $1 AND user_login = $2", orderID, login)
	var foundOrder string
	err = row.Scan(&foundOrder)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			err = &models.CustomErr{Code: models.CustomErrOrderNotFoundForWithdraw}
			logger.Sugar().Error(err)
			return err
		} else {
			logger.Sugar().Error(err)
			return err
		}
	}
	_, err = tx.Exec(
		ctx,
		"UPDATE orders SET withdrawn = COALESCE(withdrawn, 0.0) + $1, processed_at = CURRENT_TIMESTAMP WHERE id = $2",
		sum, orderID,
	)
	if err != nil {
		logger.Sugar().Error(err)
		return err
	}
	err = tx.Commit(ctx)
	if err != nil {
		logger.Sugar().Error(err)
		return err
	}
	return nil
}
