package service

import (
	"context"

	"github.com/scouser-122/gophermart/internal/models"
	"github.com/scouser-122/gophermart/internal/repository"
	"github.com/scouser-122/gophermart/internal/repository/memory"
)

type UserService struct {
	UserStorage repository.UserStorage
}

func NewUserService() *UserService {
	service := UserService{}
	memStorage := memory.UserMemoryStorage{}
	service.UserStorage = &memStorage
	return &service
}

func (service *UserService) Register(ctx context.Context, user *models.User) (*models.User, error) {
	newUser, err := service.UserStorage.AddUser(ctx, user)
	if err != nil {
		return nil, err
	}
	return newUser, nil
}
