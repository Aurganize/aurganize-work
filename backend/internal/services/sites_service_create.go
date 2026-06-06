package services

import (
	"context"
	"errors"
	"strconv"
	"strings"

	"git.aurganize.com/Aurganize/aurganize-work-backend/internal/domain"
	"git.aurganize.com/Aurganize/aurganize-work-backend/internal/storage/gen"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgtype"
)

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
