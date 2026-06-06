package services

import (
	"git.aurganize.com/Aurganize/aurganize-work-backend/internal/api/response"
	"git.aurganize.com/Aurganize/aurganize-work-backend/internal/storage/gen"
	"github.com/google/uuid"
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
