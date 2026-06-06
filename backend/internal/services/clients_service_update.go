package services

import (
	"context"
	"errors"

	"git.aurganize.com/Aurganize/aurganize-work-backend/internal/domain"
	"git.aurganize.com/Aurganize/aurganize-work-backend/internal/storage/gen"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
)

// Update applies partial changes. Nil fields are left untouched.
func (s *ClientService) Update(
	ctx context.Context,
	q *gen.Queries,
	id uuid.UUID,
	in UpdateClientInput,
) (*gen.Clients, error) {
	params := gen.UpdateClientParams{
		ID:           id,
		Name:         in.Name,
		ContactName:  in.ContactName,
		ContactEmail: in.ContactEmail,
		ContactPhone: in.ContactPhone,
		Gstin:        in.GSTIN,
		Pan:          in.PAN,
		StateCode:    in.StateCode,
		Notes:        in.Notes,
	}

	if in.BillingAddress != nil {
		raw, err := jsonOrEmpty(in.BillingAddress)
		if err != nil {
			return nil, domain.ErrInvalidInput("invalid billing_address", err)
		}
		params.BillingAddress = raw
	}

	updatedClient, err := q.UpdateClient(ctx, params)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, domain.ErrNotFound("client", err)
		}

		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) {
			switch pgErr.Code {
			case "23505": // unique_violation
				return nil, domain.ErrConflict("client already exists in this tenant", err)
			case "23514": // check_violation
				return nil, domain.ErrInvalidInput("invalid format (e.g., GSTIN)", err)
			}
		}
		return nil, domain.ErrInternal(err)
	}

	return &updatedClient, nil
}
