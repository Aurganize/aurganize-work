package services

import (
	"context"
	"errors"

	"git.aurganize.com/Aurganize/aurganize-work-backend/internal/domain"
	"git.aurganize.com/Aurganize/aurganize-work-backend/internal/storage/gen"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
)

// Get fetches one client by ID. Returns NotFound if it doesn't exist —
// or if RLS hides it because it belongs to another tenant.
//
// IMPORTANT: the indistinguishability of "doesn't exist" vs "exists but in
// another tenant" is intentional and is the cross-tenant safety property.
// Never differentiate these in the response.
func (s *ClientService) Get(
	ctx context.Context,
	q *gen.Queries,
	id uuid.UUID,
) (*gen.Clients, error) {
	client, err := q.GetClient(ctx, id)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, domain.ErrNotFound("client", err)
		}
		return nil, domain.ErrInternal(err)
	}
	return &client, nil
}
