package services

import (
	"context"
	"errors"

	"git.aurganize.com/Aurganize/aurganize-work-backend/internal/domain"
	"git.aurganize.com/Aurganize/aurganize-work-backend/internal/storage/gen"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
)

// Me returns the current user's profile. Runs inside the request's
// tenancy context (caller passes the tenant-scoped tx via gen.New).
func (s *AuthService) Me(ctx context.Context, querier *gen.Queries, userId uuid.UUID) (*MeResult, error) {
	user, err := querier.GetUserByID(ctx, userId)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, domain.ErrNotFound("user not found", err)
		}
		return nil, domain.ErrInternal(err)
	}

	return &MeResult{
		UserID:   user.ID,
		TenantID: user.TenantID,
		Email:    user.Email,
		Name:     user.Name,
		Role:     string(user.Role),
	}, nil
}
