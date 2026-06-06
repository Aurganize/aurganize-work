package handlers

import (
	"strconv"

	"git.aurganize.com/Aurganize/aurganize-work-backend/internal/api/middleware"
	"git.aurganize.com/Aurganize/aurganize-work-backend/internal/api/response"
	"git.aurganize.com/Aurganize/aurganize-work-backend/internal/domain"
	"git.aurganize.com/Aurganize/aurganize-work-backend/internal/services"
	"git.aurganize.com/Aurganize/aurganize-work-backend/internal/storage/gen"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

type SiteHandler struct {
	svc *services.SitesService
}

func NewSiteHanlder(siteService *services.SitesService) *SiteHandler {
	return &SiteHandler{
		svc: siteService,
	}
}

type CreateSiteRequest struct {
	ClientID           uuid.UUID      `json:"client_id"             binding:"required"`
	Name               string         `json:"name"                  binding:"required,min=1,max=200"`
	Address            map[string]any `json:"address"               binding:"required"`
	Latitude           *float64       `json:"latitude"              binding:"omitempty,gte=-90,lte=90"`
	Longitude          *float64       `json:"longitude"             binding:"omitempty,gte=-180,lte=180"`
	ContactOnSiteName  *string        `json:"contact_on_site_name"  binding:"omitempty,max=200"`
	ContactOnSitePhone *string        `json:"contact_on_site_phone" binding:"omitempty,max=40"`
	AccessNotes        *string        `json:"access_notes"          binding:"omitempty,max=2000"`
}

type UpdateSiteRequest struct {
	Name               *string        `json:"name"                  binding:"omitempty,min=1,max=200"`
	Address            map[string]any `json:"address"`
	Latitude           *float64       `json:"latitude"              binding:"omitempty,gte=-90,lte=90"`
	Longitude          *float64       `json:"longitude"             binding:"omitempty,gte=-180,lte=180"`
	ContactOnSiteName  *string        `json:"contact_on_site_name"  binding:"omitempty,max=200"`
	ContactOnSitePhone *string        `json:"contact_on_site_phone" binding:"omitempty,max=40"`
	AccessNotes        *string        `json:"access_notes"          binding:"omitempty,max=2000"`
}

func (h *SiteHandler) Create(c *gin.Context) {
	a := middleware.MustAuth(c.Request.Context())
	tx := middleware.GetDBtx(c.Request.Context())
	if tx == nil {
		response.RenderError(c, domain.ErrInternal(nil))
		return
	}

	var req CreateSiteRequest
	if err := c.Bind(&req); err != nil {
		response.RenderValidationErrors(c, parseValidationError(err))
		return
	}

	if (req.Latitude == nil) != (req.Longitude == nil) {
		response.RenderError(c, domain.ErrInvalidInput(
			"lattitude and longitude must both be present or both absent", nil))
		return
	}

	site, err := h.svc.Create(c.Request.Context(), gen.New(tx), a.TenantId, services.CreateSiteInput{
		ClientID:           req.ClientID,
		Name:               req.Name,
		Address:            req.Address,
		Latitude:           req.Latitude,
		Longitude:          req.Longitude,
		ContactOnSiteName:  req.ContactOnSiteName,
		ContactOnSitePhone: req.ContactOnSitePhone,
		AccessNotes:        req.AccessNotes,
	})

	if err != nil {
		response.RenderError(c, err)
		return
	}
	response.Created(c, site)
}

func (h *SiteHandler) Get(c *gin.Context) {
	_ = middleware.MustAuth(c.Request.Context())
	tx := middleware.GetDBtx(c.Request.Context())
	if tx == nil {
		response.RenderError(c, domain.ErrInternal(nil))
		return
	}

	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		response.RenderError(c, domain.ErrInternal(err))
		return
	}

	site, err := h.svc.Get(c.Request.Context(), gen.New(tx), id)
	if err != nil {
		response.RenderError(c, err)
		return
	}
	response.OK(c, site)
}

func (h *SiteHandler) List(c *gin.Context) {
	_ = middleware.MustAuth(c.Request.Context())
	tx := middleware.GetDBtx(c.Request.Context())
	if tx == nil {
		response.RenderError(c, domain.ErrInternal(nil))
		return
	}

	cursor, err := response.DecodeCursor(c.Query("cursor"))
	if err != nil {
		response.RenderError(c, domain.ErrInvalidInput("invalid cursor", err))
		return
	}

	limit := int32(25)
	if l := c.Query("limit"); l != "" {
		n, perr := strconv.Atoi(l)
		if perr != nil || n <= 0 || n > 100 {
			response.RenderError(c, domain.ErrInvalidInput("limit must be 1-100", nil))
			return
		}
		limit = int32(n)
	}

	var clientFilter *uuid.UUID
	if cid := c.Query("client_id"); cid != "" {
		parsed, err := uuid.Parse(cid)
		if err != nil {
			response.RenderError(c, domain.ErrInvalidInput("invalid client_id", err))
			return
		}
		clientFilter = &parsed
	}

	page, err := h.svc.List(c.Request.Context(), gen.New(tx), services.ListSitesInput{
		Limit:    limit,
		Cursor:   cursor,
		ClientID: clientFilter,
	})

	if err != nil {
		response.RenderError(c, err)
		return
	}
	response.OK(c, page)
}

func (h *SiteHandler) Update(c *gin.Context) {
	_ = middleware.MustAuth(c.Request.Context())
	tx := middleware.GetDBtx(c.Request.Context())
	if tx == nil {
		response.RenderError(c, domain.ErrInternal(nil))
		return
	}

	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		response.RenderError(c, domain.ErrInternal(err))
		return
	}

	var req UpdateSiteRequest
	if err := c.Bind(&req); err != nil {
		response.RenderValidationErrors(c, parseValidationError(err))
		return
	}

	site, err := h.svc.Update(c.Request.Context(), gen.New(tx), id, services.UpdateSiteInput{
		Name:               req.Name,
		Address:            req.Address,
		Latitude:           req.Latitude,
		Longitude:          req.Longitude,
		ContactOnSiteName:  req.ContactOnSiteName,
		ContactOnSitePhone: req.ContactOnSitePhone,
		AcessNotes:         req.AccessNotes,
	})
	if err != nil {
		response.RenderError(c, err)
		return
	}

	response.OK(c, site)
}

func (h *SiteHandler) Delete(c *gin.Context) {
	_ = middleware.MustAuth(c.Request.Context())
	tx := middleware.GetDBtx(c.Request.Context())
	if tx == nil {
		response.RenderError(c, domain.ErrInternal(nil))
		return
	}

	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		response.RenderError(c, domain.ErrInternal(err))
	}

	if err := h.svc.Delete(c.Request.Context(), gen.New(tx), id); err != nil {
		response.RenderError(c, err)
		return
	}
	response.NoContent(c)
}
