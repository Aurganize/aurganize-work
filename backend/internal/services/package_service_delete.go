package services

import (
	"context"
	"errors"

	"git.aurganize.com/Aurganize/aurganize-work-backend/internal/domain"
	"git.aurganize.com/Aurganize/aurganize-work-backend/internal/storage/gen"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
)

func (s *ProjectService) Delete(
	ctx context.Context,
	q *gen.Queries,
	id uuid.UUID,
) error {
	proj, err := q.GetProject(ctx, id)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return domain.ErrNotFound("project", err)
		}
		return domain.ErrInternal(err)
	}

	// Business rule: can't delete an active project at production/signoff.
	// If the user really wants to, they must first cancel it.
	if proj.Status == gen.ProjectStatusActive &&
		(proj.Stage == gen.ProjectStageProduction || proj.Stage == gen.ProjectStageSignoff) {
		return domain.ErrBusinessRuleViolation(
			"cannot delete an active project at production or signoff stagel; cancle it first", nil)
	}

	err = q.SoftDeleteProject(ctx, id)
	if err != nil {
		return domain.ErrInternal(err)
	}
	return nil
}
