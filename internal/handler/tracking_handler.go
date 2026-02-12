package handler

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/gorilla/websocket"
	"go.uber.org/zap"

	"github.com/Kilat-Pet-Delivery/lib-common/auth"
	"github.com/Kilat-Pet-Delivery/lib-common/middleware"
	"github.com/Kilat-Pet-Delivery/lib-common/response"
	"github.com/Kilat-Pet-Delivery/service-tracking/internal/application"
	"github.com/Kilat-Pet-Delivery/service-tracking/internal/ws"
)

var upgrader = websocket.Upgrader{
	ReadBufferSize:  1024,
	WriteBufferSize: 1024,
	CheckOrigin: func(r *http.Request) bool {
		// In production, restrict to specific origins.
		return true
	},
}

// TrackingHandler handles HTTP and WebSocket requests for tracking.
type TrackingHandler struct {
	service    *application.TrackingService
	hub        *ws.Hub
	jwtManager *auth.JWTManager
	logger     *zap.Logger
}

// NewTrackingHandler creates a new TrackingHandler.
func NewTrackingHandler(
	service *application.TrackingService,
	hub *ws.Hub,
	jwtManager *auth.JWTManager,
	logger *zap.Logger,
) *TrackingHandler {
	return &TrackingHandler{
		service:    service,
		hub:        hub,
		jwtManager: jwtManager,
		logger:     logger,
	}
}

// RegisterRoutes registers the REST API routes for tracking.
func (h *TrackingHandler) RegisterRoutes(r *gin.RouterGroup, jwtManager *auth.JWTManager) {
	tracking := r.Group("/tracking")
	tracking.Use(middleware.AuthMiddleware(jwtManager))
	{
		tracking.GET("/:bookingId", h.GetTracking)
		tracking.GET("/:bookingId/route", h.GetRouteGeoJSON)
	}
}

// RegisterWSRoute registers the WebSocket route on the engine.
func (h *TrackingHandler) RegisterWSRoute(r *gin.Engine, jwtManager *auth.JWTManager) {
	r.GET("/ws/tracking/:bookingId", h.HandleWebSocket)
}

// GetTracking returns the tracking data for a booking.
func (h *TrackingHandler) GetTracking(c *gin.Context) {
	bookingIDStr := c.Param("bookingId")
	bookingID, err := uuid.Parse(bookingIDStr)
	if err != nil {
		response.BadRequest(c, "invalid booking ID format")
		return
	}

	tracking, err := h.service.GetTracking(c.Request.Context(), bookingID)
	if err != nil {
		response.Error(c, err)
		return
	}

	response.Success(c, tracking)
}

// GetRouteGeoJSON returns the route as GeoJSON for a booking's trip.
func (h *TrackingHandler) GetRouteGeoJSON(c *gin.Context) {
	bookingIDStr := c.Param("bookingId")
	bookingID, err := uuid.Parse(bookingIDStr)
	if err != nil {
		response.BadRequest(c, "invalid booking ID format")
		return
	}

	geoJSON, err := h.service.GetRouteGeoJSON(c.Request.Context(), bookingID)
	if err != nil {
		response.Error(c, err)
		return
	}

	c.Data(http.StatusOK, "application/geo+json", []byte(geoJSON))
}

// HandleWebSocket upgrades the connection to WebSocket and subscribes to tracking updates.
func (h *TrackingHandler) HandleWebSocket(c *gin.Context) {
	// Validate JWT from query parameter.
	token := c.Query("token")
	if token == "" {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "token query parameter is required"})
		return
	}

	_, err := h.jwtManager.ValidateAccessToken(token)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "invalid or expired token"})
		return
	}

	// Parse booking ID.
	bookingIDStr := c.Param("bookingId")
	bookingID, err := uuid.Parse(bookingIDStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid booking ID format"})
		return
	}

	// Upgrade to WebSocket.
	conn, err := upgrader.Upgrade(c.Writer, c.Request, nil)
	if err != nil {
		h.logger.Error("failed to upgrade to websocket", zap.Error(err))
		return
	}

	client := &ws.Client{
		Conn:      conn,
		BookingID: bookingID,
		Send:      make(chan []byte, 256),
	}

	h.hub.Register(client)

	// Start read and write pumps in separate goroutines.
	go client.WritePump(h.hub)
	go client.ReadPump(h.hub)
}
