package service

import (
	"context"
	"crypto/sha256"
	"fmt"

	"github.com/scouser-122/gophermart/internal/models"
	"github.com/scouser-122/gophermart/internal/repository"
	"github.com/scouser-122/gophermart/internal/repository/db"
)

type UsersService struct {
	UserStorage repository.UserStorage
}

func NewUsersService(database *db.PostgresDatabase) (*UsersService, error) {
	service := UsersService{}
	if err := database.Ping(context.Background()); err != nil {
		return nil, fmt.Errorf("cannot connect to DB")
	}
	dbStorage := db.PostgresUserStorage{
		Database: database,
	}
	service.UserStorage = &dbStorage
	return &service, nil
}

func (service *UsersService) Register(ctx context.Context, user *models.User) (*models.User, error) {
	if user.Login == "" {
		return nil, &models.CustomErr{
			Code: models.CustomErrLoginInvalidFormat,
		}
	}
	hash := sha256.New()
	_, err := hash.Write([]byte(user.Password))
	if err != nil {
		return nil, err
	}
	user.Password = fmt.Sprintf("%x", hash.Sum(nil))
	newUser, err := service.UserStorage.AddUser(ctx, user)
	if err != nil {
		return nil, err
	}
	return newUser, nil
}
