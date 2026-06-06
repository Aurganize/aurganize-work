package services

import (
	"git.aurganize.com/Aurganize/aurganize-work-backend/internal/api/response"
	"git.aurganize.com/Aurganize/aurganize-work-backend/internal/storage/gen"
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
