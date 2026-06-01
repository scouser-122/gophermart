package service

import (
	"errors"
	"time"

	"github.com/golang-jwt/jwt/v4"
	"github.com/scouser-122/gophermart/internal/config"
)

// JwtService specifies service to generate and parse JWT tokens
type JwtService struct {
	serverConfig *config.ServerConfig
}

// NewJwtService creates new JwtService instance
func NewJwtService(serverConfig *config.ServerConfig) *JwtService {
	return &JwtService{
		serverConfig: serverConfig,
	}
}

// GenerateJWT generates JWT for specified login
func (service *JwtService) GenerateJWT(login string) (string, error) {
	liveTime := time.Hour * time.Duration(service.serverConfig.JwtTokenExp)
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.RegisteredClaims{
		Subject:   login,
		ExpiresAt: jwt.NewNumericDate(time.Now().Add(liveTime)),
	})

	tokenString, err := token.SignedString([]byte(service.serverConfig.JwtSecretKey))
	if err != nil {
		return "", err
	}

	return tokenString, nil
}

// GetUserLoginFromJWT parses JWT and takes login from it's structure
func (service *JwtService) GetUserLoginFromJWT(jwtString string) (string, error) {
	claims := &jwt.RegisteredClaims{}

	token, err := jwt.ParseWithClaims(jwtString, claims, func(token *jwt.Token) (interface{}, error) {
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, errors.New("unexpected signing method")
		}
		return []byte(service.serverConfig.JwtSecretKey), nil
	})

	if err != nil {
		return "", err
	}

	if claims, ok := token.Claims.(*jwt.RegisteredClaims); ok && token.Valid {
		return claims.Subject, nil
	}

	return "", errors.New("invalid token")
}
