package repository

import (
	"context"
	"database/sql"

	"github.com/suar-net/suar-be/internal/model"
)

// IRepository is the main interface that combines all repository interfaces.
// This is useful for dependency injection and mocking.
type IRepository interface {
	User() IUserRepository
	Request() IRequestRepository
}

// Repository is the struct that holds all repository implementations.
type Repository struct {
	user    IUserRepository
	request IRequestRepository
}

// NewRepository is the constructor to create a new Repository instance.
func NewRepository(db *sql.DB) *Repository {
	return &Repository{
		user:    NewUserRepository(db),
		request: NewRequestRepository(db),
	}
}

// User returns the IUserRepository implementation.
func (r *Repository) User() IUserRepository {
	return r.user
}

// Request returns the IRequestRepository implementation.
func (r *Repository) Request() IRequestRepository {
	return r.request
}

// IUserRepository defines the interface for data operations on users.
type IUserRepository interface {
	Create(ctx context.Context, user *model.User) (int, error)
	GetByEmail(ctx context.Context, email string) (*model.User, error)
}

// IRequestRepository defines the interface for data operations on request history.
type IRequestRepository interface {
	Create(ctx context.Context, request *model.Request) error
	GetByUserID(ctx context.Context, userID int) ([]*model.Request, error)
}
