package memory

import (
	"context"
	"fmt"
	"slices"
	"time"

	"github.com/scouser-122/gophermart/internal/models"
)

type UserMemoryStorage struct {
	Users []*models.User
}

func (storage *UserMemoryStorage) AddUser(ctx context.Context, user *models.User) (*models.User, error) {
	alreadyExist := slices.ContainsFunc(storage.Users, func(storageUser *models.User) bool {
		return storageUser.Login == user.Login
	})
	if alreadyExist {
		return nil, models.ErrorLoginAlreadyTaken{}
	}
	user.CreatedAt = time.Now()
	storage.Users = append(storage.Users, user)
	return user, nil
}

func (storage *UserMemoryStorage) GetUser(ctx context.Context, login string) (*models.User, error) {
	index := slices.IndexFunc(storage.Users, func(user *models.User) bool {
		return user.Login == login
	})
	if index != -1 {
		return storage.Users[index], nil
	}
	return nil, fmt.Errorf("user not found")
}
