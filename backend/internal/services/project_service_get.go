package services

import (
	"context"
	"errors"

	"git.aurganize.com/Aurganize/aurganize-work-backend/internal/domain"
	"git.aurganize.com/Aurganize/aurganize-work-backend/internal/storage/gen"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
)

// Get returns a project plus its linked sites in one call.
func (s *ProjectService) Get(
	ctx context.Context,
	q *gen.Queries,
	id uuid.UUID,
) (*ProjectWithSites, error) {
	proj, err := q.GetProject(ctx, id)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, domain.ErrNotFound("project", err)
		}
		return nil, domain.ErrInternal(err)
	}

	sites, err := q.ListSitesByProject(ctx, proj.ID)
	if err != nil {
		return nil, domain.ErrInternal(err)
	}

	return &ProjectWithSites{
		Projects: proj,
		Sites:    sites,
	}, nil
}
