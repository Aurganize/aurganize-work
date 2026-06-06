package services

import (
	"context"
	"errors"

	"git.aurganize.com/Aurganize/aurganize-work-backend/internal/domain"
	"git.aurganize.com/Aurganize/aurganize-work-backend/internal/storage/gen"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
)

// Delete is a soft delete. Calling Delete on an already-deleted client is
// idempotent — returns nil.
func (s *ClientService) Delete(
	ctx context.Context,
	q *gen.Queries,
	id uuid.UUID,
) error {
	// First check it exists (else we get a silent no-op which the API caller
	// can't distinguish from a wrong-tenant attempt).
	if _, err := q.GetClient(ctx, id); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return domain.ErrNotFound("client", err)
		}
		return domain.ErrInternal(err)
	}

	// Block delete if the client has active sites — would orphan them.
	// (In Batch 2 file 09 we'll layer business-rule errors more thoroughly.)
	n, err := q.CountSitesByClient(ctx, id)
	if err != nil {
		return domain.ErrInternal(err)
	}

	if n > 0 {
		return domain.ErrBusinessRuleViolation("client has active sites; delete sites first", nil)
	}

	if err := q.SoftDeleteClient(ctx, id); err != nil {
		return domain.ErrInternal(err)
	}

	return nil

}
