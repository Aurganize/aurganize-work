package services

import (
	"context"
	"fmt"

	"git.aurganize.com/Aurganize/aurganize-work-backend/internal/api/response"
	"git.aurganize.com/Aurganize/aurganize-work-backend/internal/domain"
	"git.aurganize.com/Aurganize/aurganize-work-backend/internal/storage/gen"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgtype"
)

func (s *ProjectService) List(
	ctx context.Context,
	q *gen.Queries,
	in ListProjectsInput,
) (*ProjectPage, error) {

	if in.Limit <= 0 || in.Limit >= 100 {
		in.Limit = 25
	}

	var afterCreatedAt pgtype.Timestamptz
	var afterId *uuid.UUID

	if in.Cursor != nil {
		t := in.Cursor.CreatedAt
		id := in.Cursor.ID
		afterCreatedAt = pgtype.Timestamptz{
			Time:  t,
			Valid: true,
		}
		afterId = &id
	}

	// Validate filter enum values (better error than the DB enum cast).
	var stage gen.NullProjectStage
	if in.Stage != nil {
		v, err := parseProjectStage(*in.Stage)
		if err != nil {
			return nil, err
		}
		stage = v
	}

	var status gen.NullProjectStatus
	if in.Status != nil {
		v, err := parseProjectStatus(*in.Status)
		if err != nil {
			return nil, err
		}
		status = v
	}

	items, err := q.ListProjects(ctx, gen.ListProjectsParams{
		Limit:          in.Limit,
		Stage:          stage,
		Status:         status,
		ClientID:       in.ClientID,
		OwnerUserID:    in.OwnerUserID,
		AfterCreatedAt: afterCreatedAt,
		AfterID:        afterId,
	})
	if err != nil {
		return nil, domain.ErrInternal(err)
	}

	out := &ProjectPage{Items: items}
	if int32(len(items)) == in.Limit {
		last := items[len(items)-1]
		nextCursor := response.Cursor{
			CreatedAt: last.CreatedAt,
			ID:        last.ID,
		}
		out.NextCursor = nextCursor.EncodeCursor()
	}

	return out, nil
}

func parseProjectStage(s string) (gen.NullProjectStage, error) {
	switch gen.ProjectStage(s) {
	case gen.ProjectStageLead, gen.ProjectStageSurvey, gen.ProjectStageDesign,
		gen.ProjectStageQuote, gen.ProjectStageProduction, gen.ProjectStageSignoff,
		gen.ProjectStageCompleted:
		return gen.NullProjectStage{
			ProjectStage: gen.ProjectStage(s),
			Valid:        true,
		}, nil
	}
	return gen.NullProjectStage{
			Valid: false,
		}, domain.ErrInvalidInput(
			fmt.Sprintf("invalid stage %v; must be one of lead, survey, design, quote, production, signoff, completed", s), nil,
		)
}

func parseProjectStatus(s string) (gen.NullProjectStatus, error) {
	switch gen.ProjectStatus(s) {
	case gen.ProjectStatusActive, gen.ProjectStatusPaused, gen.ProjectStatusCancelled:
		return gen.NullProjectStatus{
			ProjectStatus: gen.ProjectStatus(s),
			Valid:         true,
		}, nil
	}
	return gen.NullProjectStatus{
			Valid: false,
		}, domain.ErrInvalidInput(
			fmt.Sprintf("invalid status %v; must be one of active, pause, cancelled", s), nil,
		)
}
