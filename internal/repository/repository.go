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

type IRepository interface {
	User() IUserRepository
	Request() IRequestRepository
}

type Repository struct {
	user    IUserRepository
	request IRequestRepository
}

func NewRepository(db *sql.DB) *Repository {
	return &Repository{
		user:    NewUserRepository(db),
		request: NewRequestRepository(db),
	}
}

func (r *Repository) User() IUserRepository {
	return r.user
}

func (r *Repository) Request() IRequestRepository {
	return r.request
}
