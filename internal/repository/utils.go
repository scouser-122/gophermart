package repository

import (
	"context"

	"github.com/scouser-122/gophermart/internal/models"
)

// RepositoryUtils declares interface with repository utils
type RepositoryUtils interface {
	// CreateTransaction creates transaction to be used in several operations
	CreateTransaction(ctx context.Context) (models.GenericTransaction, error)
}
