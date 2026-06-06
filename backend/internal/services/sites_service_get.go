package services

import (
	"context"
	"errors"

	"git.aurganize.com/Aurganize/aurganize-work-backend/internal/domain"
	"git.aurganize.com/Aurganize/aurganize-work-backend/internal/storage/gen"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
)

func (s *SitesService) Get(
	ctx context.Context,
	q *gen.Queries,
	id uuid.UUID,
) (*gen.Sites, error) {
	site, err := q.GetSite(ctx, id)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, domain.ErrNotFound("site", err)
		}
		return nil, domain.ErrInternal(err)
	}
	return &site, nil
}
