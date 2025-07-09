package websocket

import (
	"log"
	"net/http"

	"github.com/gin-gonic/gin"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

type Handler struct {
	hub *Hub
}

func NewHandler() *Handler {
	hub := NewHub()
	go hub.Run()

	return &Handler{
		hub: hub,
	}
}

func (h *Handler) HandleWebSocket(c *gin.Context) {
	// Extract user info from JWT token (implement based on your auth middleware)
	userID, exists := c.Get("user_id")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
		return
	}

	userType, exists := c.Get("user_type")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "User type not found"})
		return
	}

	userObjectID, ok := userID.(primitive.ObjectID)
	if !ok {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid user ID"})
		return
	}

	userTypeStr, ok := userType.(string)
	if !ok {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid user type"})
		return
	}

	conn, err := upgrader.Upgrade(c.Writer, c.Request, nil)
	if err != nil {
		log.Printf("WebSocket upgrade failed: %v", err)
		return
	}

	client := NewClient(h.hub, conn, userObjectID, userTypeStr)
	h.hub.register <- client

	go client.writePump()
	go client.readPump()
}

func (h *Handler) SendRideUpdate(rideID primitive.ObjectID, updateType string, data map[string]interface{}) {
	message := Message{
		Type:      updateType,
		RoomID:    "ride_" + rideID.Hex(),
		Timestamp: getCurrentTimestamp(),
		Data:      data,
	}

	h.hub.SendRideUpdate(rideID, message)
}

func (h *Handler) SendUserNotification(userID primitive.ObjectID, notificationType string, data map[string]interface{}) {
	message := Message{
		Type:      notificationType,
		UserID:    userID,
		Timestamp: getCurrentTimestamp(),
		Data:      data,
	}

	h.hub.SendToUser(userID, message)
}

func (h *Handler) GetHub() *Hub {
	return h.hub
}
