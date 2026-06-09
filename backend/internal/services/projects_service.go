package services

import (
	"time"

	"git.aurganize.com/Aurganize/aurganize-work-backend/internal/api/response"
	"git.aurganize.com/Aurganize/aurganize-work-backend/internal/storage/gen"
	"github.com/google/uuid"
)

// ProjectsService holds the counter dependency for code allocation.
// Stage transition logic lives in the StagesService (file 09).
type ProjectService struct {
	counterSvc *CounterService
}

func NewProjectService(counter *CounterService) *ProjectService {
	return &ProjectService{
		counterSvc: counter,
	}
}

// === Inputs / outputs ===

type CreateProjectInput struct {
	ClientID             uuid.UUID
	Name                 string
	Description          *string
	OwnerUserID          *uuid.UUID
	EstimatedStartAt     *time.Time
	EstimatedEndAt       *time.Time
	EstimatedBudgetPaise *int64
	SiteIDs              []uuid.UUID // optional initial sites
}

type UpdateProjectInput struct {
	Name                 *string
	Description          *string
	Status               *string
	OwnerUserID          *uuid.UUID
	EstimatedStartAt     *time.Time
	EstimatedEndAt       *time.Time
	ActualStartAt        *time.Time
	ActualEndAt          *time.Time
	EstimatedBudgetPaise *int64
}

type ListProjectsInput struct {
	Limit       int32
	Cursor      *response.Cursor
	Stage       *string
	Status      *string
	ClientID    *uuid.UUID
	OwnerUserID *uuid.UUID
}

type ProjectPage struct {
	Items      []gen.Projects `json:"items"`
	NextCursor string         `json:"next_cursor,omitempty"`
}

// ProjectWithSites is a Project plus its associated sites. Returned by
// Get so the client can render the full picture in one round trip.
type ProjectWithSites struct {
	gen.Projects
	Sites []gen.Sites
}
