package handlers

import (
	"strconv"
	"time"

	"git.aurganize.com/Aurganize/aurganize-work-backend/internal/api/middleware"
	"git.aurganize.com/Aurganize/aurganize-work-backend/internal/api/response"
	"git.aurganize.com/Aurganize/aurganize-work-backend/internal/domain"
	"git.aurganize.com/Aurganize/aurganize-work-backend/internal/services"
	"git.aurganize.com/Aurganize/aurganize-work-backend/internal/storage/gen"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

type ProjectHandler struct {
	svc *services.ProjectService
}

func NewProjectHandler(s *services.ProjectService) *ProjectHandler {
	return &ProjectHandler{
		svc: s,
	}
}

type CreateProjectRequest struct {
	ClientID             uuid.UUID   `json:"client_id"               binding:"required"`
	Name                 string      `json:"name"                    binding:"required,min=2,max=200"`
	Description          *string     `json:"description"             binding:"omitempty,max=4000"`
	OwnerUserID          *uuid.UUID  `json:"owner_user_id"`
	EstimatedStartAt     *time.Time  `json:"estimated_start_at"`
	EstimatedEndAt       *time.Time  `json:"estimated_end_at"`
	EstimatedBudgetPaise *int64      `json:"estimated_budget_paise"  binding:"omitempty,gte=0"`
	SiteIDs              []uuid.UUID `json:"site_ids"`
}

type UpdateProjectRequest struct {
	Name                 *string    `json:"name"                    binding:"omitempty,min=2,max=200"`
	Description          *string    `json:"description"             binding:"omitempty,max=4000"`
	Status               *string    `json:"status"                  binding:"omitempty,oneof=active paused cancelled"`
	OwnerUserID          *uuid.UUID `json:"owner_user_id"`
	EstimatedStartAt     *time.Time `json:"estimated_start_at"`
	EstimatedEndAt       *time.Time `json:"estimated_end_at"`
	ActualStartAt        *time.Time `json:"actual_start_at"`
	ActualEndAt          *time.Time `json:"actual_end_at"`
	EstimatedBudgetPaise *int64     `json:"estimated_budget_paise"  binding:"omitempty,gte=0"`
}

type SetProjectSitesRequest struct {
	SiteIDs []uuid.UUID `json:"site_ids" binding:"required"`
}

func (h *ProjectHandler) Create(c *gin.Context) {
	a := middleware.MustAuth(c.Request.Context())
	tx := middleware.GetDBtx(c.Request.Context())
	if tx == nil {
		response.RenderError(c, domain.ErrInternal(nil))
		return
	}

	var req CreateProjectRequest
	if err := c.Bind(&req); err != nil {
		response.RenderValidationErrors(c, parseValidationError(err))
		return
	}

	proj, err := h.svc.Create(c.Request.Context(), gen.New(tx), a.TenantId, services.CreateProjectInput{
		ClientID:             req.ClientID,
		Name:                 req.Name,
		Description:          req.Description,
		OwnerUserID:          req.OwnerUserID,
		EstimatedStartAt:     req.EstimatedStartAt,
		EstimatedEndAt:       req.EstimatedEndAt,
		EstimatedBudgetPaise: req.EstimatedBudgetPaise,
		SiteIDs:              req.SiteIDs,
	})
	if err != nil {
		response.RenderError(c, err)
		return
	}

	response.Created(c, proj)
}

func (h *ProjectHandler) Get(c *gin.Context) {
	_ = middleware.MustAuth(c.Request.Context())
	tx := middleware.GetDBtx(c.Request.Context())
	if tx == nil {
		response.RenderError(c, domain.ErrInternal(nil))
	}

	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		response.RenderError(c, domain.ErrInvalidInput("invalid id", nil))
		return
	}

	proj, err := h.svc.Get(c.Request.Context(), gen.New(tx), id)
	if err != nil {
		response.RenderError(c, err)
		return
	}

	response.OK(c, proj)
}

func (h *ProjectHandler) List(c *gin.Context) {
	_ = middleware.MustAuth(c.Request.Context())
	tx := middleware.GetDBtx(c.Request.Context())
	if tx == nil {
		response.RenderError(c, domain.ErrInternal(nil))
		return
	}

	limit := int32(25)
	if l := c.Query("limit"); l != "" {
		n, perr := strconv.Atoi(l)
		if perr != nil {
			response.RenderError(c, domain.ErrInvalidInput("limit must be 1-100", nil))
			return
		}
		limit = int32(n)
	}

	in := services.ListProjectsInput{Limit: limit}
	if s := c.Query("stage"); s != "" {
		in.Stage = &s
	}

	if s := c.Query("status"); s != "" {
		in.Status = &s
	}

	if s := c.Query("client_id"); s != "" {
		parsed, perr := uuid.Parse(s)
		if perr != nil {
			response.RenderError(c, domain.ErrInvalidInput("invalid client_id", perr))
			return
		}
		in.ClientID = &parsed
	}

	if s := c.Query("owner_user_id"); s != "" {
		parsed, perr := uuid.Parse(s)
		if perr != nil {
			response.RenderError(c, domain.ErrInvalidInput("invalid owner_user_id", perr))
		}
		in.OwnerUserID = &parsed
	}

	page, err := h.svc.List(c.Request.Context(), gen.New(tx), in)
	if err != nil {
		response.RenderError(c, err)
		return
	}

	response.OK(c, page)
}

func (h *ProjectHandler) Update(c *gin.Context) {
	_ = middleware.MustAuth(c.Request.Context())
	tx := middleware.GetDBtx(c.Request.Context())

	if tx == nil {
		response.RenderError(c, domain.ErrInternal(nil))
	}
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		response.RenderError(c, domain.ErrInvalidInput("invalid id", err))
		return
	}

	var req UpdateProjectRequest
	if err := c.Bind(&req); err != nil {
		response.RenderValidationErrors(c, parseValidationError(err))
		return
	}

	proj, err := h.svc.Update(c.Request.Context(), gen.New(tx), id, services.UpdateProjectInput{
		Name:                 req.Name,
		Description:          req.Description,
		Status:               req.Status,
		OwnerUserID:          req.OwnerUserID,
		EstimatedStartAt:     req.EstimatedStartAt,
		EstimatedEndAt:       req.EstimatedStartAt,
		ActualStartAt:        req.ActualStartAt,
		ActualEndAt:          req.ActualEndAt,
		EstimatedBudgetPaise: req.EstimatedBudgetPaise,
	})
	if err != nil {
		response.RenderError(c, err)
		return
	}

	response.OK(c, proj)
}

func (h *ProjectHandler) Delete(c *gin.Context) {
	_ = middleware.MustAuth(c.Request.Context())
	tx := middleware.GetDBtx(c.Request.Context())
	if tx == nil {
		response.RenderError(c, domain.ErrInternal(nil))
		return
	}

	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		response.RenderError(c, domain.ErrInvalidInput("invalid id", err))
		return
	}

	if err := h.svc.Delete(c.Request.Context(), gen.New(tx), id); err != nil {
		response.RenderError(c, err)
		return
	}

	response.NoContent(c)
}

func (h *ProjectHandler) SetSites(c *gin.Context) {
	a := middleware.MustAuth(c.Request.Context())
	tx := middleware.GetDBtx(c.Request.Context())
	if tx == nil {
		response.RenderError(c, domain.ErrInternal(nil))
		return
	}

	id, err := uuid.Parse("id")
	if err != nil {
		response.RenderError(c, domain.ErrInvalidInput("invalid id", nil))
		return
	}

	var req SetProjectSitesRequest
	if err := c.Bind(&req); err != nil {
		response.RenderValidationErrors(c, parseValidationError(err))
		return
	}

	sites, err := h.svc.SetSites(c.Request.Context(), gen.New(tx), a.TenantId, id, req.SiteIDs)
	if err != nil {
		response.RenderError(c, err)
		return
	}

	response.OK(c, sites)
}
