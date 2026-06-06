package services

import (
	"context"
	"errors"
	"strconv"

	"git.aurganize.com/Aurganize/aurganize-work-backend/internal/domain"
	"git.aurganize.com/Aurganize/aurganize-work-backend/internal/storage/gen"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgtype"
)

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
