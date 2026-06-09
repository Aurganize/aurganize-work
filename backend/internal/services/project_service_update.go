package services

import (
	"context"
	"errors"

	"git.aurganize.com/Aurganize/aurganize-work-backend/internal/domain"
	"git.aurganize.com/Aurganize/aurganize-work-backend/internal/storage/gen"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgtype"
)

func (s *ProjectService) Update(
	ctx context.Context,
	q *gen.Queries,
	id uuid.UUID,
	in UpdateProjectInput,
) (*gen.Projects, error) {
	// Owner change requires user validation.
	if in.OwnerUserID != nil {
		user, err := q.GetUserByID(ctx, *in.OwnerUserID)
		if err != nil {
			if errors.Is(err, pgx.ErrNoRows) {
				return nil, domain.ErrNotFound("owner user", nil)
			}
			return nil, domain.ErrInternal(err)
		}

		if !user.IsActive {
			return nil, domain.ErrInvalidInput("owner user is inactive", nil)
		}
	}

	var statusEnum gen.NullProjectStatus
	if in.Status != nil {
		v, err := parseProjectStatus(*in.Status)
		if err != nil {
			return nil, err
		}
		statusEnum = v
	}

	var estimatedStartAt pgtype.Timestamptz
	var estimatedEndAt pgtype.Timestamptz
	var actualStartAt pgtype.Timestamptz
	var actualEndAt pgtype.Timestamptz

	// might be re-written later based on business requirement
	if in.EstimatedStartAt == nil {
		estimatedStartAt = pgtype.Timestamptz{
			Valid: false,
		}
	} else {
		estimatedStartAt = pgtype.Timestamptz{
			Time:  *in.EstimatedStartAt,
			Valid: true,
		}
	}

	if in.EstimatedEndAt == nil {
		estimatedEndAt = pgtype.Timestamptz{
			Valid: false,
		}
	} else {
		estimatedEndAt = pgtype.Timestamptz{
			Time:  *in.EstimatedEndAt,
			Valid: true,
		}
	}
	if in.ActualStartAt == nil {
		actualStartAt = pgtype.Timestamptz{
			Valid: false,
		}
	} else {
		actualStartAt = pgtype.Timestamptz{
			Time:  *in.ActualStartAt,
			Valid: true,
		}
	}

	if in.ActualEndAt == nil {
		actualEndAt = pgtype.Timestamptz{
			Valid: false,
		}
	} else {
		actualEndAt = pgtype.Timestamptz{
			Time:  *in.ActualEndAt,
			Valid: true,
		}
	}

	proj, err := q.UpdateProject(ctx, gen.UpdateProjectParams{
		ID:                   id,
		Name:                 in.Name,
		Description:          in.Description,
		Status:               statusEnum,
		OwnerUserID:          in.OwnerUserID,
		EstimatedStartAt:     estimatedStartAt,
		EstimatedEndAt:       estimatedEndAt,
		ActualStartAt:        actualStartAt,
		ActualEndAt:          actualEndAt,
		EstimatedBudgetPaise: in.EstimatedBudgetPaise,
	})

	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, domain.ErrNotFound("project", err)
		}

		var pgError *pgconn.PgError
		if errors.As(err, &pgError) {
			return nil, domain.ErrInvalidInput("invalid dates or budget", nil)
		}
		return nil, domain.ErrInternal(err)
	}

	return &proj, nil
}
