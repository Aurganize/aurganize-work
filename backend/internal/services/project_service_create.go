package services

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"git.aurganize.com/Aurganize/aurganize-work-backend/internal/domain"
	"git.aurganize.com/Aurganize/aurganize-work-backend/internal/storage/gen"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgtype"
)

// Create allocates a code, inserts the project, and (atomically) adds the
// initial sites. All within the request's transaction so a failed site
// link rolls back the project too.
func (s *ProjectService) Create(
	ctx context.Context,
	q *gen.Queries,
	tenantId uuid.UUID,
	in CreateProjectInput,
) (*gen.Projects, error) {
	in.Name = strings.TrimSpace(in.Name)
	if in.Name == "" {
		return nil, domain.ErrInvalidInput("project name required", nil)
	}

	var estimatedStartAt pgtype.Timestamptz
	var estimatedEndAt pgtype.Timestamptz

	// might be re-written later based on business requirement
	if in.EstimatedStartAt == nil || in.EstimatedEndAt == nil {
		return nil, domain.ErrInvalidInput("estimated start or estimated end is of invalid format or nil", nil)
	}
	estimatedStartAt = pgtype.Timestamptz{
		Time:  *in.EstimatedStartAt,
		Valid: true,
	}
	estimatedEndAt = pgtype.Timestamptz{
		Time:  *in.EstimatedEndAt,
		Valid: true,
	}

	// Verify the client exists in this tenant (RLS does cross-tenant check).
	if _, err := q.GetClient(ctx, in.ClientID); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, domain.ErrNotFound("client", err)
		}

		return nil, domain.ErrInternal(err)
	}

	// If owner provided, verify they're a real, active user in the tenant.
	if in.OwnerUserID != nil {
		user, err := q.GetUserByID(ctx, *in.OwnerUserID)
		if err != nil {
			if errors.Is(err, pgx.ErrNoRows) {
				return nil, domain.ErrNotFound("owner user", err)
			}
			return nil, domain.ErrInternal(err)
		}

		if !user.IsActive {
			return nil, domain.ErrInvalidInput("owner user is inactive", nil)
		}
	}

	// Allocate the next code: P/<year>/<NNN>.
	year := int32(time.Now().Year())
	seq, err := s.counterSvc.Allocate(ctx, q, tenantId, "project", year)
	if err != nil {
		return nil, err
	}

	code := fmt.Sprintf("P/%d/%03d", year, seq)

	// Insert the project.
	proj, err := q.CreateProject(ctx, gen.CreateProjectParams{
		TenantID:             tenantId,
		ClientID:             in.ClientID,
		Code:                 code,
		Name:                 in.Name,
		Description:          in.Description,
		Stage:                gen.ProjectStageLead,
		Status:               gen.ProjectStatusActive,
		OwnerUserID:          in.OwnerUserID,
		EstimatedStartAt:     estimatedStartAt,
		EstimatedEndAt:       estimatedEndAt,
		EstimatedBudgetPaise: in.EstimatedBudgetPaise,
	})

	if err != nil {
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) {
			switch pgErr.Code {
			case "23514": // check_violation_constraint
				return nil, domain.ErrInvalidInput("invalid dates or budget", err)
			case "23505": // unique_violation - extremely unlikey (we just allocated)
				return nil, domain.ErrConflict("project code collision; retry", err)
			}
		}

		return nil, domain.ErrInternal(err)
	}

	// Link initial sites, if any. Each site is verified to exist + belong
	// to the project's client (RLS hides cross-tenant; we also check
	// same-client below to avoid a designer accidentally linking a site
	// from a different client of the same tenant).
	for _, siteId := range in.SiteIDs {
		site, err := q.GetSite(ctx, siteId)
		if err != nil {
			if errors.Is(err, pgx.ErrNoRows) {
				return nil, domain.ErrNotFound("site", err)
			}
			return nil, domain.ErrInternal(err)
		}

		if site.ClientID != in.ClientID {
			return nil, domain.ErrInvalidInput(
				"site does not belong to the project's client", nil,
			)
		}

		if err := q.AddProjectSite(ctx, gen.AddProjectSiteParams{
			ProjectID: proj.ID,
			TenantID:  tenantId,
			SiteID:    site.ID,
			Notes:     nil,
		}); err != nil {
			return nil, domain.ErrInternal(err)
		}
	}

	return &proj, nil

}
