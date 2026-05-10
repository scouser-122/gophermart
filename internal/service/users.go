package service

import (
	"context"

	"github.com/scouser-122/gophermart/internal/models"
	"github.com/scouser-122/gophermart/internal/repository"
	"github.com/scouser-122/gophermart/internal/utils"
)

// UsersService service to work with users
type UsersService struct {
	userStorage   repository.UserStorage
	ordersService *OrdersService
}

// NewUsersService creates new UsersService instance
func NewUsersService(userStorage repository.UserStorage, ordersService *OrdersService) *UsersService {
	service := UsersService{}
	service.userStorage = userStorage
	service.ordersService = ordersService
	return &service
}

// Register runs registration process for specified user
func (service *UsersService) Register(ctx context.Context, user *models.User) (*models.User, error) {
	if user.Login == "" {
		return nil, &models.CustomErr{
			Code: models.CustomErrUserLoginInvalidFormat,
		}
	}
	if user.Password == "" {
		return nil, &models.CustomErr{Code: models.CustomErrUserPasswordInvalidFormat}
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
		return &models.CustomErr{Code: models.CustomErrUserLoginInvalidFormat}
	}
	if user.Password == "" {
		return &models.CustomErr{Code: models.CustomErrUserPasswordInvalidFormat}
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
		return &models.CustomErr{Code: models.CustomErrUserLoginFailed}
	}
	return nil
}

// GetUserBalance return user's loyalty points balance and total withdrawn points
func (service *UsersService) GetUserBalance(ctx context.Context, login string) (*models.UserBalanceResponse, error) {
	result := models.UserBalanceResponse{}
	var err error
	result.Current, err = service.userStorage.GetUserBalance(ctx, login)
	if err != nil {
		return nil, err
	}
	result.Withdrawn, err = service.ordersService.GetWithdrawnForUser(ctx, login)
	if err != nil {
		return nil, err
	}
	return &result, nil
}
