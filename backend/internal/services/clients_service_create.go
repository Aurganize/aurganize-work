package services

import (
	"context"
	"errors"
	"strings"

	"git.aurganize.com/Aurganize/aurganize-work-backend/internal/domain"
	"git.aurganize.com/Aurganize/aurganize-work-backend/internal/storage/gen"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgconn"
)

// Create inserts a new client for the tenant in the current context.
// Returns ErrConflict if a client with the same name (case-insensitive)
// already exists.
func (s *ClientService) Create(
	ctx context.Context,
	q *gen.Queries,
	tenantId uuid.UUID,
	in CreateClientInput,
) (*gen.Clients, error) {
	in.Name = strings.TrimSpace(in.Name)
	if in.Name == "" {
		return nil, domain.ErrInvalidInput("client name required", nil)
	}

	billingAddress, err := jsonOrEmpty(in.BillingAddress)
	if err != nil {
		return nil, domain.ErrInvalidInput("invalid billing_address", err)
	}

	client, err := q.CreateClient(ctx, gen.CreateClientParams{
		TenantID:       tenantId,
		Name:           in.Name,
		ContactName:    in.ContactName,
		ContactEmail:   in.ContactEmail,
		ContactPhone:   in.ContactPhone,
		BillingAddress: billingAddress,
		Gstin:          in.GSTIN,
		Pan:            in.PAN,
		StateCode:      in.StateCode,
		Notes:          in.Notes,
	})

	if err != nil {
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
	return &client, nil
}
