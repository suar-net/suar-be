package repository

import (
	"context"
	"database/sql"

	"github.com/suar-net/suar-be/internal/model"
)

type IUserRepository interface {
	Create(ctx context.Context, user *model.User) (int, error)
	GetByEmail(ctx context.Context, email string) (*model.User, error)
}

type IRequestRepository interface {
	Create(ctx context.Context, request *model.Request) error
	GetByUserID(ctx context.Context, userID int) ([]*model.Request, error)
}

type Repository struct {
	userRepo    IUserRepository
	requestRepo IRequestRepository
}

func NewRepository(db *sql.DB) *Repository {
	return &Repository{
		userRepo:    NewUserRepository(db),
		requestRepo: NewRequestRepository(db),
	}
}

func (r *Repository) UserRepo() IUserRepository {
	return r.userRepo
}

func (r *Repository) RequestRepo() IRequestRepository {
	return r.requestRepo
}
