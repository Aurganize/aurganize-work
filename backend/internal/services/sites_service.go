package services

import (
	"context"
	"errors"
	"strconv"
	"strings"

	"git.aurganize.com/Aurganize/aurganize-work-backend/internal/api/response"
	"git.aurganize.com/Aurganize/aurganize-work-backend/internal/domain"
	"git.aurganize.com/Aurganize/aurganize-work-backend/internal/storage/gen"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgtype"
)

type SitesService struct{}

func NewSitesService() *SitesService {
	return &SitesService{}
}

type CreateSiteInput struct {
	ClientID           uuid.UUID
	Name               string
	Address            map[string]any
	Latitude           *float64
	Longitude          *float64
	ContactOnSiteName  *string
	ContactOnSitePhone *string
	AccessNotes        *string
}

type UpdateSiteInput struct {
	Name               *string
	Address            map[string]any
	Latitude           *float64
	Longitude          *float64
	ContactOnSiteName  *string
	ContactOnSitePhone *string
	AcessNotes         *string
}

type ListSitesInput struct {
	Limit    int32
	Cursor   *response.Cursor
	ClientID *uuid.UUID
}

type SitePage struct {
	Items      []gen.Sites `json:"items"`
	NextCursor string      `json:"next_cursor,omitempty"`
}

// Create inserts a new site. Verifies the parent client exists and belongs
// to the same tenant — RLS does the second part automatically.
func (s *SitesService) Create(
	ctx context.Context,
	q *gen.Queries,
	tenantId uuid.UUID,
	in CreateSiteInput,
) (*gen.Sites, error) {
	in.Name = strings.TrimSpace(in.Name)
	if in.Name == "" {
		return nil, domain.ErrInvalidInput("site name required", nil)
	}
	if in.Address == nil || len(in.Address) == 0 {
		return nil, domain.ErrInvalidInput("address required", nil)
	}

	if _, err := q.GetClient(ctx, in.ClientID); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, domain.ErrNotFound("client", err)
		}
		return nil, domain.ErrInternal(err)
	}

	addr, err := jsonOrEmpty(in.Address)
	if err != nil {
		return nil, domain.ErrInvalidInput("invalid address", err)
	}
	var lattitude pgtype.Numeric
	var longitude pgtype.Numeric

	if err := lattitude.Scan(strconv.FormatFloat(*in.Latitude, 'f', 6, 64)); err != nil {
		return nil, domain.ErrInvalidInput("provided latitude is invalid", err)
	}
	if err := longitude.Scan(strconv.FormatFloat(*in.Longitude, 'f', 6, 64)); err != nil {
		return nil, domain.ErrInvalidInput("provided longitude is invalid", err)
	}

	site, err := q.CreateSite(ctx, gen.CreateSiteParams{
		TenantID:           tenantId,
		ClientID:           in.ClientID,
		Name:               in.Name,
		Address:            addr,
		Latitude:           lattitude,
		Longitude:          longitude,
		ContactOnSiteName:  in.ContactOnSiteName,
		ContactOnSitePhone: in.ContactOnSitePhone,
		AccessNotes:        in.AccessNotes,
	})

	if err != nil {
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) {
			switch pgErr.Code {
			case "23505":
				return nil, domain.ErrConflict("site name already exists for this client", err)
			case "23514":
				return nil, domain.ErrInvalidInput("invalid coordinates (lat/lng range or mismatched null)", err)
			case "23503":
				return nil, domain.ErrNotFound("client", err)
			}
		}
		return nil, domain.ErrInternal(err)
	}

	return &site, nil
}

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

func (s *SitesService) Update(
	ctx context.Context,
	q *gen.Queries,
	id uuid.UUID,
	in UpdateSiteInput,
) (*gen.Sites, error) {
	params := gen.UpdateSiteParams{
		ID:                 id,
		Name:               in.Name,
		ContactOnSiteName:  in.ContactOnSiteName,
		ContactOnSitePhone: in.ContactOnSitePhone,
		AccessNotes:        in.AcessNotes,
	}

	if in.Address != nil {
		raw, err := jsonOrEmpty(in.Address)
		if err != nil {
			return nil, domain.ErrInvalidInput("invalid address", err)
		}
		params.Address = raw
	}

	var lattitude pgtype.Numeric
	var longitude pgtype.Numeric
	if in.Latitude != nil {
		if err := lattitude.Scan(strconv.FormatFloat(*in.Latitude, 'f', 6, 64)); err != nil {
			return nil, domain.ErrInvalidInput("provided latitude is invalid", err)
		}
	}
	if in.Longitude != nil {
		if err := longitude.Scan(strconv.FormatFloat(*in.Longitude, 'f', 6, 64)); err != nil {
			return nil, domain.ErrInvalidInput("provided longitude is invalid", err)
		}
	}

	if (in.Latitude == nil) != (in.Longitude == nil) {
		return nil, domain.ErrInvalidInput(
			"latitude and longitude must both be provided or both omitted",
			nil,
		)
	}

	params.Latitude = lattitude
	params.Longitude = longitude

	site, err := q.UpdateSite(ctx, params)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, domain.ErrNotFound("site", err)
		}
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) {
			if pgErr.Code == "23505" {
				return nil, domain.ErrConflict("site name conflicts under this client", err)
			}
			if pgErr.Code == "23514" {
				return nil, domain.ErrInvalidInput("invalid coordinates", err)
			}
		}
		return nil, domain.ErrInternal(err)
	}
	return &site, err

}

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
