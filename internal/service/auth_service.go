package service

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/suar-net/suar-be/internal/config"
	"github.com/suar-net/suar-be/internal/model"
	"github.com/suar-net/suar-be/internal/repository"
	"golang.org/x/crypto/bcrypt"
)

type authService struct {
	userRepo  repository.IUserRepository
	jwtConfig config.JWTConfig
}

func NewAuthService(userRepo repository.IUserRepository, jwtConfig config.JWTConfig) IAuthService {
	return &authService{
		userRepo:  userRepo,
		jwtConfig: jwtConfig,
	}
}

func (s *authService) Register(ctx context.Context, userReg *model.DTOUserRegisterRequest) (*model.User, error) {
	existingUser, err := s.userRepo.GetByEmail(ctx, userReg.Email)
	if err != nil {
		return nil, fmt.Errorf("error checking for existing email: %w", err)
	}
	if existingUser != nil {
		return nil, ErrEmailTaken
	}

	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(userReg.Password), bcrypt.DefaultCost)
	if err != nil {
		return nil, fmt.Errorf("error hashing password: %w", err)
	}

	user := model.User{
		Username:     userReg.Username,
		Email:        userReg.Email,
		PasswordHash: string(hashedPassword),
	}

	newUserID, err := s.userRepo.Create(ctx, &user)
	if err != nil {
		return nil, fmt.Errorf("failed to create user: %w", err)
	}

	user.ID = newUserID
	return &user, nil
}

func (s *authService) Login(ctx context.Context, userLog *model.DTOLoginRequest) (*model.DTOLoginResponse, error) {
	user, err := s.userRepo.GetByEmail(ctx, userLog.Email)
	if err != nil {
		return nil, fmt.Errorf("database error: %w", err)
	}
	if user == nil {
		return nil, ErrInvalidCredentials
	}

	err = bcrypt.CompareHashAndPassword([]byte(user.PasswordHash), []byte(userLog.Password))
	if err != nil {
		return nil, ErrInvalidCredentials
	}

	expirationTime := time.Now().Add(s.jwtConfig.AccessTokenExpiresIn)
	claims := &model.Claims{
		ID:       user.ID,
		Username: user.Username,
		Email:    user.Email,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(expirationTime),
			IssuedAt:  jwt.NewNumericDate(time.Now()),
			Issuer:    "suar-be",
		},
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	tokenString, err := token.SignedString([]byte(s.jwtConfig.SecretKey))
	if err != nil {
		return nil, fmt.Errorf("failed to create token: %w", err)
	}

	return &model.DTOLoginResponse{
		AccessToken: tokenString,
		TokenType:   "Bearer",
	}, nil
}

func (s *authService) ValidateToken(ctx context.Context, tokenString string) (*model.Claims, error) {
	claims := &model.Claims{}
	token, err := jwt.ParseWithClaims(tokenString, claims, func(token *jwt.Token) (interface{}, error) {
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("unexpected signing method: %v", token.Header["alg"])
		}
		return []byte(s.jwtConfig.SecretKey), nil
	})

	if err != nil {
		if errors.Is(err, jwt.ErrTokenExpired) {
			return nil, ErrTokenExpired
		}
		return nil, ErrTokenInvalid
	}

	if !token.Valid {
		return nil, ErrTokenInvalid
	}

	return claims, nil
}
