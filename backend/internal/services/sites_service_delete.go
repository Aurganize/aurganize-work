package services

import (
	"context"
	"errors"

	"git.aurganize.com/Aurganize/aurganize-work-backend/internal/domain"
	"git.aurganize.com/Aurganize/aurganize-work-backend/internal/storage/gen"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
)

func (s *SitesService) Delete(
	ctx context.Context,
	q *gen.Queries,
	id uuid.UUID,
) error {
	if _, err := q.GetSite(ctx, id); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return domain.ErrNotFound("site", err)
		}
		return domain.ErrInternal(err)
	}

	if err := q.SoftDeleteSite(ctx, id); err != nil {
		return domain.ErrInternal(err)
	}

	return nil
}
