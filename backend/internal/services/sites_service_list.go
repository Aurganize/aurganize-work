package services

import (
	"context"

	"git.aurganize.com/Aurganize/aurganize-work-backend/internal/api/response"
	"git.aurganize.com/Aurganize/aurganize-work-backend/internal/domain"
	"git.aurganize.com/Aurganize/aurganize-work-backend/internal/storage/gen"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgtype"
)

func (s *SitesService) List(
	ctx context.Context,
	q *gen.Queries,
	in ListSitesInput,
) (*SitePage, error) {
	if in.Limit <= 0 || in.Limit > 100 {
		in.Limit = 50
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

	var items []gen.Sites
	var err error

	if in.ClientID != nil {
		items, err = q.ListSitesByClient(ctx, gen.ListSitesByClientParams{
			Limit:          in.Limit,
			ClientID:       *in.ClientID,
			AfterCreatedAt: afterCreatedAt,
			AfterID:        afterId,
		})
	} else {
		items, err = q.ListSites(ctx, gen.ListSitesParams{
			Limit:          in.Limit,
			AfterCreatedAt: afterCreatedAt,
			AfterID:        afterId,
		})
	}
	if err != nil {
		return nil, domain.ErrInternal(err)
	}
	out := &SitePage{Items: items}

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
