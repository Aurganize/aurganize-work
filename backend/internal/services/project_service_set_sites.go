package services

import (
	"context"
	"errors"

	"git.aurganize.com/Aurganize/aurganize-work-backend/internal/domain"
	"git.aurganize.com/Aurganize/aurganize-work-backend/internal/storage/gen"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
)

// SetSites replaces the project's site list atomically with the given list.
// Each site is validated to belong to the project's client; on any failure,
// the entire change rolls back (the request's tx).
func (s *ProjectService) SetSites(
	ctx context.Context,
	q *gen.Queries,
	tenantId uuid.UUID,
	projectId uuid.UUID,
	siteIds []uuid.UUID,
) ([]gen.Sites, error) {
	proj, err := q.GetProject(ctx, projectId)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, domain.ErrNotFound("project", err)
		}
		return nil, domain.ErrInternal(err)
	}

	// Verify every site exists and belongs to the project's client.
	for _, siteId := range siteIds {
		site, err := q.GetSite(ctx, siteId)
		if err != nil {
			if errors.Is(err, pgx.ErrNoRows) {
				return nil, domain.ErrNotFound("site", err)
			}
			return nil, domain.ErrInternal(err)
		}
		if site.ClientID != proj.ClientID {
			return nil, domain.ErrInvalidInput(
				"site does not belong to the project's client", nil,
			)
		}
	}

	// Atomic replacement: delete all existing links, then re-insert.
	// Cheap and simple; the alternative (compute diff, insert/delete deltas)
	// is more code with no observable benefit for a join table this small.
	if err := q.RemoveAllProjectSites(ctx, projectId); err != nil {
		return nil, domain.ErrInternal(err)
	}

	for _, siteId := range siteIds {
		if err := q.AddProjectSite(ctx, gen.AddProjectSiteParams{
			ProjectID: projectId,
			SiteID:    siteId,
			TenantID:  tenantId,
			Notes:     nil,
		}); err != nil {
			return nil, domain.ErrInternal(err)
		}
	}

	sites, err := q.ListSitesByProject(ctx, projectId)
	if err != nil {
		return nil, domain.ErrInternal(err)
	}

	return sites, nil
}
