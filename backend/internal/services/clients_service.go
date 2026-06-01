package services

import (
	"context"
	"encoding/json"
	"errors"
	"strings"

	"git.aurganize.com/Aurganize/aurganize-work-backend/internal/api/response"
	"git.aurganize.com/Aurganize/aurganize-work-backend/internal/domain"
	"git.aurganize.com/Aurganize/aurganize-work-backend/internal/storage/gen"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgtype"
)

type ClientService struct{}

func NewClientService() *ClientService {
	return &ClientService{}
}

type CreateClientInput struct {
	Name           string
	ContactName    *string
	ContactEmail   *string
	ContactPhone   *string
	BillingAddress map[string]any
	GSTIN          *string
	PAN            *string
	StateCode      *string
	Notes          *string
}

type UpdateClientInput struct {
	Name           *string
	ContactName    *string
	ContactEmail   *string
	ContactPhone   *string
	BillingAddress map[string]any
	GSTIN          *string
	PAN            *string
	StateCode      *string
	Notes          *string
}

type ListClientsInput struct {
	Limit  int32
	Cursor *response.Cursor
	Query  string // optional case-insensitive name search
}

type ClientPage struct {
	Items      []gen.Clients `json:"items"`
	NextCursor string        `json:"next_cursor,omitemtpy"`
}

// === Operations ===

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

func jsonOrEmpty(m map[string]any) ([]byte, error) {
	if m == nil {
		return []byte("{}"), nil
	}
	return json.Marshal(m)
}

// Get fetches one client by ID. Returns NotFound if it doesn't exist —
// or if RLS hides it because it belongs to another tenant.
//
// IMPORTANT: the indistinguishability of "doesn't exist" vs "exists but in
// another tenant" is intentional and is the cross-tenant safety property.
// Never differentiate these in the response.
func (s *ClientService) Get(
	ctx context.Context,
	q *gen.Queries,
	id uuid.UUID,
) (*gen.Clients, error) {
	client, err := q.GetClient(ctx, id)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, domain.ErrNotFound("client", err)
		}
		return nil, domain.ErrInternal(err)
	}
	return &client, nil
}

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

func (s *ClientService) Delete(
	ctx context.Context,
	q *gen.Queries,
	id uuid.UUID,
) error {
	if _, err := q.GetClient(ctx, id); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return domain.ErrNotFound("client", err)
		}
		return domain.ErrInternal(err)
	}

}
