package service

import (
	"context"

	"github.com/suar-net/suar-be/internal/model"
	"github.com/suar-net/suar-be/internal/repository"
)

type IRequestService interface {
	ProcessRequest(ctx context.Context, dto *model.DTORequest) (*model.DTOResponse, error)
	GetHistory()
}

type IUserService interface {
	Register()
	Login()
}

type Service struct {
	requestService IRequestService
}

func NewService(r repository.Repository) *Service {
	return &Service{
		requestService: NewRequestService(r.RequestRepo()),
	}
}

func (s *Service) RequestService() IRequestService {
	return s.requestService
}
