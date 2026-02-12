package handler

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"

	"github.com/Kilat-Pet-Delivery/lib-common/auth"
	"github.com/Kilat-Pet-Delivery/lib-common/middleware"
	"github.com/Kilat-Pet-Delivery/lib-common/response"
	"github.com/Kilat-Pet-Delivery/service-tracking/internal/application"
)

// ShareHandler handles HTTP requests for trip sharing.
type ShareHandler struct {
	service *application.ShareService
}

// NewShareHandler creates a new ShareHandler.
func NewShareHandler(service *application.ShareService) *ShareHandler {
	return &ShareHandler{service: service}
}

// RegisterRoutes registers authenticated share routes.
func (h *ShareHandler) RegisterRoutes(r *gin.RouterGroup, jwtManager *auth.JWTManager) {
	authMW := middleware.AuthMiddleware(jwtManager)

	tracking := r.Group("/tracking")
	tracking.POST("/:bookingId/share", authMW, h.CreateShareLink)

	// Public route — no auth required
	tracking.GET("/shared/:token", h.GetSharedTracking)
}

// CreateShareLink handles POST /api/v1/tracking/:bookingId/share.
func (h *ShareHandler) CreateShareLink(c *gin.Context) {
	bookingID, err := uuid.Parse(c.Param("bookingId"))
	if err != nil {
		response.BadRequest(c, "invalid booking ID")
		return
	}

	result, err := h.service.CreateShareLink(c.Request.Context(), bookingID)
	if err != nil {
		response.Error(c, err)
		return
	}

	response.Created(c, result)
}

// GetSharedTracking handles GET /api/v1/tracking/shared/:token (public, no auth).
func (h *ShareHandler) GetSharedTracking(c *gin.Context) {
	token := c.Param("token")
	if token == "" {
		response.BadRequest(c, "token is required")
		return
	}

	result, err := h.service.GetSharedTracking(c.Request.Context(), token)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": err.Error(), "success": false})
		return
	}

	response.Success(c, result)
}
