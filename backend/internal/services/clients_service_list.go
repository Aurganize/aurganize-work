package services

import (
	"context"
	"strings"

	"git.aurganize.com/Aurganize/aurganize-work-backend/internal/api/response"
	"git.aurganize.com/Aurganize/aurganize-work-backend/internal/domain"
	"git.aurganize.com/Aurganize/aurganize-work-backend/internal/storage/gen"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgtype"
)

// List returns a page of clients. Caller decodes the cursor; service
// returns next cursor (or empty when end reached).
func (s *ClientService) List(
	ctx context.Context,
	q *gen.Queries,
	in ListClientsInput,
) (*ClientPage, error) {
	if in.Limit <= 0 || in.Limit > 100 {
		in.Limit = 25
	}

	var afterCreatedAt pgtype.Timestamptz
	var afterID *uuid.UUID
	if in.Cursor != nil {
		t := in.Cursor.CreatedAt
		id := in.Cursor.ID
		afterCreatedAt = pgtype.Timestamptz{
			Time:  t,
			Valid: true,
		}
		afterID = &id
	}

	var items []gen.Clients
	var err error
	in.Query = strings.TrimSpace(in.Query)
	if in.Query != "" {
		items, err = q.ListClientsByQuery(ctx, gen.ListClientsByQueryParams{
			Limit:          in.Limit,
			Query:          in.Query,
			AfterCreatedAt: afterCreatedAt,
			AfterID:        afterID,
		})
	} else {
		items, err = q.ListClients(ctx, gen.ListClientsParams{
			Limit:          in.Limit,
			AfterCreatedAt: afterCreatedAt,
			AfterID:        afterID,
		})
	}

	if err != nil {
		return nil, domain.ErrInternal(err)
	}

	out := &ClientPage{
		Items: items,
	}

	if int32(len(items)) == in.Limit {
		// More may exist — emit a cursor pointing at the last row.
		last := items[len(items)-1]
		nextCursor := response.Cursor{
			CreatedAt: last.CreatedAt,
			ID:        last.ID,
		}
		out.NextCursor = nextCursor.EncodeCursor()
	}
	return out, nil
}
