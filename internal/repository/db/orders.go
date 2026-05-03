package db

import (
	"context"
	"errors"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/scouser-122/gophermart/internal/models"
)

// PostgresOrderStorage implements OrderStorage interface to store orders data in Postgres DB
type PostgresOrderStorage struct {
	Database *PostgresDatabase
}

// AddOrder adds specified order,
// returns error if order with specified ID already exists or add process failed
func (storage *PostgresOrderStorage) AddOrder(ctx context.Context, orderID string, userLogin string) (*models.Order, error) {
	var dbOrder models.Order
	err := storage.Database.Select(ctx, &dbOrder, "SELECT * FROM orders WHERE id = $1 AND user_login = $2", orderID, userLogin)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			dbOrder.ID = orderID
			dbOrder.UploadedAt = time.Now()
			dbOrder.Status = models.NewOrder
			dbOrder.UserLogin = userLogin
			_, err = storage.Database.Exec(
				ctx,
				"INSERT INTO orders (id,status,uploaded_at,user_login) VALUES ($1,$2,$3,$4)",
				dbOrder.ID, dbOrder.Status, dbOrder.UploadedAt, dbOrder.UserLogin,
			)
			if err != nil {
				return nil, err
			}
			return &dbOrder, nil
		} else {
			return nil, err
		}
	}
	return nil, &models.CustomErr{Code: models.CustomErrOrderAlreadyUploaded}
}
