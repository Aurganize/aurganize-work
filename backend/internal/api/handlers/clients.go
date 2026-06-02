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

type ClientHandler struct {
	svc *services.ClientService
}

func NewClientHandler(svc *services.ClientService) *ClientHandler {
	return &ClientHandler{
		svc: svc,
	}
}

// === Request shapes ===
//
// We keep request struct names suffixed by the operation (CreateClientRequest)
// rather than reusing one shape for both create and update. This lets the
// validator distinguish "required on create" from "optional on update."

type CreateClientRequest struct {
	Name           string         `json:"name"           binding:"required,min=2,max=200"`
	ContactName    *string        `json:"contact_name"   binding:"omitempty,max=200"`
	ContactEmail   *string        `json:"contact_email"  binding:"omitempty,email"`
	ContactPhone   *string        `json:"contact_phone"  binding:"omitempty,max=40"`
	BillingAddress map[string]any `json:"billing_address"`
	GSTIN          *string        `json:"gstin"          binding:"omitempty,len=15"`
	PAN            *string        `json:"pan"            binding:"omitempty,len=10"`
	StateCode      *string        `json:"state_code"     binding:"omitempty,len=2"`
	Notes          *string        `json:"notes"          binding:"omitempty,max=2000"`
}

type UpdateClientRequest struct {
	Name           *string        `json:"name"           binding:"omitempty,min=2,max=200"`
	ContactName    *string        `json:"contact_name"   binding:"omitempty,max=200"`
	ContactEmail   *string        `json:"contact_email"  binding:"omitempty,email"`
	ContactPhone   *string        `json:"contact_phone"  binding:"omitempty,max=40"`
	BillingAddress map[string]any `json:"billing_address"`
	GSTIN          *string        `json:"gstin"          binding:"omitempty,len=15"`
	PAN            *string        `json:"pan"            binding:"omitempty,len=10"`
	StateCode      *string        `json:"state_code"     binding:"omitempty,len=2"`
	Notes          *string        `json:"notes"          binding:"omitempty,max=2000"`
}

func (h *ClientHandler) Create(c *gin.Context) {
	a := middleware.MustAuth(c.Request.Context())
	tx := middleware.GetDBtx(c.Request.Context())
	if tx == nil {
		response.RenderError(c, domain.ErrInternal(nil))
		return
	}

	var req CreateClientRequest
	if err := c.Bind(&req); err != nil {
		response.RenderValidationErrors(c, parseValidationError(err))
		return
	}

	client, err := h.svc.Create(c.Request.Context(), gen.New(tx), a.TenantId, services.CreateClientInput{
		Name:           req.Name,
		ContactName:    req.ContactName,
		ContactEmail:   req.ContactEmail,
		ContactPhone:   req.ContactPhone,
		BillingAddress: req.BillingAddress,
		GSTIN:          req.GSTIN,
		PAN:            req.PAN,
		StateCode:      req.StateCode,
		Notes:          req.Notes,
	})
	if err != nil {
		response.RenderError(c, err)
		return
	}

	response.Created(c, client)

}

func (h *ClientHandler) Get(c *gin.Context) {
	_ = middleware.MustAuth(c.Request.Context())
	tx := middleware.GetDBtx(c.Request.Context())

	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		response.RenderError(c, err)
		return
	}

	client, err := h.svc.Get(c.Request.Context(), gen.New(tx), id)
	if err != nil {
		response.RenderError(c, err)
		return
	}

	response.OK(c, client)

}

func (h *ClientHandler) List(c *gin.Context) {
	_ = middleware.MustAuth(c.Request.Context())
	tx := middleware.GetDBtx(c.Request.Context())

	cursor, err := response.DecodeCursor(c.Query("cursor"))
	if err != nil {
		response.RenderError(c, err)
		return
	}

	limit := int32(25)
	if l := c.Query("limit"); l != "" {
		n, parseErr := strconv.Atoi(l)
		if parseErr != nil || n <= 0 || n > 100 {
			response.RenderError(c, domain.ErrInvalidInput("limit must be 1-100", nil))
			return
		}
		limit = int32(n)
	}

	page, err := h.svc.List(c.Request.Context(), gen.New(tx), services.ListClientsInput{
		Limit:  limit,
		Cursor: cursor,
		Query:  c.Query("query"),
	})
	if err != nil {
		response.RenderError(c, err)
		return
	}
	response.OK(c, page)
}

func (h *ClientHandler) Update(c *gin.Context) {
	_ = middleware.MustAuth(c.Request.Context())
	tx := middleware.GetDBtx(c.Request.Context())
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		response.RenderError(c, err)
		return
	}

	var req UpdateClientRequest
	if err := c.Bind(&req); err != nil {
		response.RenderValidationErrors(c, parseValidationError(err))
		return
	}

	client, err := h.svc.Update(c.Request.Context(), gen.New(tx), id, services.UpdateClientInput{
		Name:           req.Name,
		ContactName:    req.ContactName,
		ContactEmail:   req.ContactEmail,
		ContactPhone:   req.ContactPhone,
		BillingAddress: req.BillingAddress,
		GSTIN:          req.GSTIN,
		PAN:            req.PAN,
		StateCode:      req.StateCode,
		Notes:          req.Notes,
	})
	if err != nil {
		response.RenderError(c, err)
		return
	}

	response.OK(c, client)
}

func (h *ClientHandler) Delete(c *gin.Context) {
	_ = middleware.MustAuth(c.Request.Context())
	tx := middleware.GetDBtx(c.Request.Context())
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		response.RenderError(c, err)
		return
	}

	if err := h.svc.Delete(c.Request.Context(), gen.New(tx), id); err != nil {
		response.RenderError(c, err)
		return
	}

	response.NoContent(c)
}
