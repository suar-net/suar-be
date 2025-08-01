package repository

import (
	"context"
	"database/sql"

	"github.com/suar-net/suar-be/internal/model"
)

// requestRepository is the implementation of IRequestRepository.
type requestRepository struct {
	db *sql.DB
}

// NewRequestRepository is the constructor for requestRepository.
func NewRequestRepository(db *sql.DB) IRequestRepository {
	return &requestRepository{db: db}
}

// Create inserts a new request record into the database.
func (r *requestRepository) Create(ctx context.Context, request *model.Request) error {
	query := `
		INSERT INTO request_history (user_id, request_method, request_url, request_headers, request_body, response_status_code, response_headers, response_body, response_size, duration_ms)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10)`

	_, err := r.db.ExecContext(ctx, query,
		request.UserID,
		request.RequestMethod,
		request.RequestURL,
		request.RequestHeaders,
		request.RequestBody,
		request.ResponseStatusCode,
		request.ResponseHeaders,
		request.ResponseBody,
		request.ResponseSize,
		request.DurationMs,
	)

	return err
}

// GetByUserID retrieves all request history for a specific user.
func (r *requestRepository) GetByUserID(ctx context.Context, userID int) ([]*model.Request, error) {
	query := `
		SELECT id, user_id, executed_at, request_method, request_url, request_headers, request_body, response_status_code, response_headers, response_body, response_size, duration_ms
		FROM request_history
		WHERE user_id = $1
		ORDER BY executed_at DESC`

	rows, err := r.db.QueryContext(ctx, query, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var requests []*model.Request
	for rows.Next() {
		var req model.Request
		if err := rows.Scan(
			&req.ID,
			&req.UserID,
			&req.ExecutedAt,
			&req.RequestMethod,
			&req.RequestURL,
			&req.RequestHeaders,
			&req.RequestBody,
			&req.ResponseStatusCode,
			&req.ResponseHeaders,
			&req.ResponseBody,
			&req.ResponseSize,
			&req.DurationMs,
		); err != nil {
			return nil, err
		}
		requests = append(requests, &req)
	}

	return requests, nil
}
