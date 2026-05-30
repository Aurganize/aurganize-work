package handlers

import (
	"git.aurganize.com/Aurganize/aurganize-work-backend/internal/api/middleware"
	"git.aurganize.com/Aurganize/aurganize-work-backend/internal/api/response"
	"git.aurganize.com/Aurganize/aurganize-work-backend/internal/auth"
	"git.aurganize.com/Aurganize/aurganize-work-backend/internal/domain"
	"git.aurganize.com/Aurganize/aurganize-work-backend/internal/services"
	"git.aurganize.com/Aurganize/aurganize-work-backend/internal/storage/gen"
	"github.com/gin-gonic/gin"
)

type AuthHandler struct {
	svc *services.AuthService
}

func NewAuthHandler(svc *services.AuthService) *AuthHandler {
	return &AuthHandler{
		svc: svc,
	}
}

// === Signup ===

// SignupRequest is the JSON body of POST /auth/signup.
//
// validator tags do shape checking only. Business rules (slug uniqueness,
// password strength, etc.) live in the service.
type SignupRequest struct {
	TenantName string `json:"tenant_name" binding:"required,min=2,max=100"`
	TenantSlug string `json:"tenant_slug" binding:"required,min=3,max=40"`
	AdminName  string `json:"admin_name" binding:"required,min=2,max=80"`
	Email      string `json:"email" binding:"required,email"`
	Password   string `json:"password" binding:"required,min=8,max=72"`
	Client     string `json:"client" binding:"required,oneof=web mobile"`
}

func (h *AuthHandler) Signup(c *gin.Context) {
	var req SignupRequest
	if err := c.Bind(&req); err != nil {
		response.RenderValidationErrors(c, parseValidationError(err))
		return
	}

	tokenPair, err := h.svc.SignUp(c.Request.Context(), services.SignupInput{
		TenantName: req.TenantName,
		TenantSlug: req.TenantSlug,
		AdminName:  req.AdminName,
		Email:      req.Email,
		Password:   req.Password,
		Client:     auth.ClientType(req.Client),
		UserAgent:  c.GetHeader("User-Agent"),
		IPAddress:  c.ClientIP(),
	})

	if err != nil {
		response.RenderError(c, err)
		return
	}

	response.Created(c, tokenPair)
}

// === Login ===
type LoginRequest struct {
	Email    string `json:"email" binding:"required,email"`
	Password string `json:"password" binding:"required"`
	Client   string `json:"client" binding:"required,oneof=web mobile"`
}

func (h *AuthHandler) Login(c *gin.Context) {
	var req LoginRequest
	if err := c.Bind(&req); err != nil {
		response.RenderValidationErrors(c, parseValidationError(err))
		return
	}

	tokenPair, err := h.svc.Login(c.Request.Context(), services.LoginInput{
		Email:     req.Email,
		Password:  req.Password,
		Client:    auth.ClientType(req.Client),
		UserAgent: c.GetHeader("User-Agent"),
		IPAddress: c.ClientIP(),
	})

	if err != nil {
		response.RenderError(c, err)
		return
	}

	response.OK(c, tokenPair)
}

// === Refresh ===
type RefreshRequest struct {
	RefreshToken string `json:"refresh_token" binding:"required"`
	Client       string `json:"client" binding:"required,oneof=web mobile"`
}

func (h *AuthHandler) Refresh(c *gin.Context) {
	var req RefreshRequest
	if err := c.Bind(&req); err != nil {
		response.RenderValidationErrors(c, parseValidationError(err))
		return
	}

	tokenPair, err := h.svc.Refresh(c.Request.Context(), services.RefreshInput{
		RefreshToken: req.RefreshToken,
		Client:       auth.ClientType(req.Client),
		UserAgent:    c.GetHeader("User-Agent"),
		IPAddress:    c.ClientIP(),
	})

	if err != nil {
		response.RenderError(c, err)
		return
	}
	response.OK(c, tokenPair)
}

// === Logout ===
type LogoutRequest struct {
	RefreshToken string `json:"refresh_token" binding:"required"`
}

func (h *AuthHandler) Logout(c *gin.Context) {
	var req LogoutRequest
	if err := c.Bind(&req); err != nil {
		response.RenderValidationErrors(c, parseValidationError(err))
		return
	}

	if err := h.svc.Logout(c.Request.Context(), services.LogoutInput{
		RefreshToken: req.RefreshToken,
	}); err != nil {
		response.RenderError(c, err)
		return
	}

	response.NoContent(c)
}

// === Me ===

// Me returns the current user's profile. Behind AuthRequired + Tenancy.
func (h *AuthHandler) Me(c *gin.Context) {
	authContext := middleware.MustAuth(c.Request.Context())
	tx := middleware.GetDBtx(c.Request.Context())
	if tx == nil {
		response.RenderError(c, domain.ErrInternal(nil))
		return
	}

	querier := gen.New(tx)
	res, err := h.svc.Me(c.Request.Context(), querier, authContext.UserId)
	if err != nil {
		response.RenderError(c, err)
		return
	}
	response.OK(c, res)

}
