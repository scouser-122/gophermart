package service

import (
	"context"
	"fmt"

	"github.com/scouser-122/gophermart/internal/models"
	"github.com/scouser-122/gophermart/internal/repository"
	"github.com/scouser-122/gophermart/internal/repository/db"
	"github.com/scouser-122/gophermart/internal/utils"
)

// UsersService service to work with users
type UsersService struct {
	userStorage repository.UserStorage
}

// NewUsersService creates new UsersService instance
func NewUsersService(database *db.PostgresDatabase) (*UsersService, error) {
	service := UsersService{}
	if err := database.Ping(context.Background()); err != nil {
		return nil, fmt.Errorf("cannot connect to DB")
	}
	dbStorage := db.PostgresUserStorage{
		Database: database,
	}
	service.userStorage = &dbStorage
	return &service, nil
}

// Register runs registration process for specified user
func (service *UsersService) Register(ctx context.Context, user *models.User) (*models.User, error) {
	if user.Login == "" {
		return nil, &models.CustomErr{
			Code: models.CustomErrLoginInvalidFormat,
		}
	}
	if user.Password == "" {
		return nil, &models.CustomErr{Code: models.CustomErrPasswordInvalidFormat}
	}
	passwordHash, err := utils.CountSha256Sum(user.Password)
	if err != nil {
		return nil, err
	}
	user.Password = passwordHash
	newUser, err := service.userStorage.AddUser(ctx, user)
	if err != nil {
		return nil, err
	}
	return newUser, nil
}

// Login runs login process for specified user
func (service *UsersService) Login(ctx context.Context, user *models.User) error {
	if user.Login == "" {
		return &models.CustomErr{Code: models.CustomErrLoginInvalidFormat}
	}
	if user.Password == "" {
		return &models.CustomErr{Code: models.CustomErrPasswordInvalidFormat}
	}
	storedUser, err := service.userStorage.GetUser(ctx, user.Login)
	if err != nil {
		return err
	}
	passwordHash, err := utils.CountSha256Sum(user.Password)
	if err != nil {
		return err
	}
	if passwordHash != storedUser.Password {
		return &models.CustomErr{Code: models.CustomErrLoginFailed}
	}
	return nil
}
