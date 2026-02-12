package handler

import (
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"

	"github.com/Kilat-Pet-Delivery/lib-common/auth"
	"github.com/Kilat-Pet-Delivery/lib-common/middleware"
	"github.com/Kilat-Pet-Delivery/lib-common/response"
	"github.com/Kilat-Pet-Delivery/service-tracking/internal/application"
)

// ChatHandler handles HTTP requests for chat operations.
type ChatHandler struct {
	service *application.ChatService
}

// NewChatHandler creates a new ChatHandler.
func NewChatHandler(service *application.ChatService) *ChatHandler {
	return &ChatHandler{service: service}
}

// RegisterRoutes registers chat routes on the given router group.
func (h *ChatHandler) RegisterRoutes(r *gin.RouterGroup, jwtManager *auth.JWTManager) {
	authMW := middleware.AuthMiddleware(jwtManager)

	chat := r.Group("/chat")
	chat.Use(authMW)
	{
		chat.POST("/:bookingId/messages", h.SendMessage)
		chat.GET("/:bookingId/messages", h.GetMessages)
	}
}

// SendMessage handles POST /api/v1/chat/:bookingId/messages.
func (h *ChatHandler) SendMessage(c *gin.Context) {
	bookingID, err := uuid.Parse(c.Param("bookingId"))
	if err != nil {
		response.BadRequest(c, "invalid booking ID")
		return
	}

	userID, ok := middleware.GetUserID(c)
	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
		return
	}

	role, ok := middleware.GetUserRole(c)
	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
		return
	}

	var req application.SendMessageRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, err.Error())
		return
	}

	result, err := h.service.SendMessage(c.Request.Context(), bookingID, userID, string(role), req)
	if err != nil {
		response.Error(c, err)
		return
	}

	response.Created(c, result)
}

// GetMessages handles GET /api/v1/chat/:bookingId/messages.
func (h *ChatHandler) GetMessages(c *gin.Context) {
	bookingID, err := uuid.Parse(c.Param("bookingId"))
	if err != nil {
		response.BadRequest(c, "invalid booking ID")
		return
	}

	page, limit := parseChatPagination(c)

	messages, total, err := h.service.GetMessages(c.Request.Context(), bookingID, page, limit)
	if err != nil {
		response.Error(c, err)
		return
	}

	response.Paginated(c, messages, total, page, limit)
}

func parseChatPagination(c *gin.Context) (int, int) {
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	limit, _ := strconv.Atoi(c.DefaultQuery("limit", "50"))
	if page < 1 {
		page = 1
	}
	if limit < 1 {
		limit = 50
	}
	if limit > 100 {
		limit = 100
	}
	return page, limit
}
