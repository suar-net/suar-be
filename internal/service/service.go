package service

import (
	"context"

	"github.com/suar-net/suar-be/internal/config"
	"github.com/suar-net/suar-be/internal/model"
	"github.com/suar-net/suar-be/internal/repository"
)

type IRequestService interface {
	ProcessRequest(ctx context.Context, dto *model.DTORequest) (*model.DTOResponse, error)
	GetHistory()
}

type IAuthService interface {
	Register(ctx context.Context, userReg *model.DTOUserRegisterRequest) (*model.User, error)
	Login(ctx context.Context, userLog *model.DTOLoginRequest) (*model.DTOLoginResponse, error)
	ValidateToken(ctx context.Context, tokenString string) (*model.Claims, error)
}

type Service struct {
	requestService IRequestService
	authService    IAuthService
}

func NewService(r repository.Repository, jwt config.JWTConfig) *Service {
	return &Service{
		requestService: NewRequestService(r.RequestRepo()),
		authService:    NewAuthService(r.UserRepo(), jwt),
	}
}

func (s *Service) RequestService() IRequestService {
	return s.requestService
}

func (s *Service) AuthService() IAuthService {
	return s.authService
}
