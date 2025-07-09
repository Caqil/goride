package websocket

import (
	"encoding/json"
	"log"
	"sync"
	"time"

	"go.mongodb.org/mongo-driver/bson/primitive"
)

type Hub struct {
	clients    map[*Client]bool
	broadcast  chan []byte
	register   chan *Client
	unregister chan *Client
	rooms      map[string]map[*Client]bool
	mutex      sync.RWMutex
}

type Message struct {
	Type      string                 `json:"type"`
	RoomID    string                 `json:"room_id,omitempty"`
	UserID    primitive.ObjectID     `json:"user_id"`
	Timestamp int64                  `json:"timestamp"`
	Data      map[string]interface{} `json:"data"`
}

func NewHub() *Hub {
	return &Hub{
		clients:    make(map[*Client]bool),
		broadcast:  make(chan []byte),
		register:   make(chan *Client),
		unregister: make(chan *Client),
		rooms:      make(map[string]map[*Client]bool),
	}
}

func (h *Hub) Run() {
	for {
		select {
		case client := <-h.register:
			h.registerClient(client)

		case client := <-h.unregister:
			h.unregisterClient(client)

		case message := <-h.broadcast:
			h.broadcastMessage(message)
		}
	}
}

func (h *Hub) registerClient(client *Client) {
	h.mutex.Lock()
	defer h.mutex.Unlock()

	h.clients[client] = true
	log.Printf("Client registered: %s", client.UserID.Hex())

	// Join user to their personal room
	personalRoom := "user_" + client.UserID.Hex()
	h.joinRoom(client, personalRoom)

	// Join driver to drivers room if applicable
	if client.UserType == "driver" {
		h.joinRoom(client, "drivers")
	}

	// Send welcome message
	welcomeMsg := Message{
		Type:      "welcome",
		UserID:    client.UserID,
		Timestamp: getCurrentTimestamp(),
		Data: map[string]interface{}{
			"message": "Connected successfully",
		},
	}

	h.sendToClient(client, welcomeMsg)
}

func (h *Hub) unregisterClient(client *Client) {
	h.mutex.Lock()
	defer h.mutex.Unlock()

	if _, ok := h.clients[client]; ok {
		delete(h.clients, client)
		close(client.send)

		// Remove from all rooms
		for roomID, room := range h.rooms {
			if _, exists := room[client]; exists {
				delete(room, client)
				if len(room) == 0 {
					delete(h.rooms, roomID)
				}
			}
		}

		log.Printf("Client unregistered: %s", client.UserID.Hex())
	}
}

func (h *Hub) broadcastMessage(message []byte) {
	var msg Message
	if err := json.Unmarshal(message, &msg); err != nil {
		log.Printf("Error unmarshaling message: %v", err)
		return
	}

	if msg.RoomID != "" {
		h.sendToRoom(msg.RoomID, msg)
	} else {
		h.sendToAll(msg)
	}
}

func (h *Hub) sendToAll(message Message) {
	h.mutex.RLock()
	defer h.mutex.RUnlock()

	data, _ := json.Marshal(message)
	for client := range h.clients {
		select {
		case client.send <- data:
		default:
			close(client.send)
			delete(h.clients, client)
		}
	}
}

func (h *Hub) sendToRoom(roomID string, message Message) {
	h.mutex.RLock()
	defer h.mutex.RUnlock()

	room, exists := h.rooms[roomID]
	if !exists {
		return
	}

	data, _ := json.Marshal(message)
	for client := range room {
		select {
		case client.send <- data:
		default:
			close(client.send)
			delete(h.clients, client)
			delete(room, client)
		}
	}
}

func (h *Hub) sendToClient(client *Client, message Message) {
	data, _ := json.Marshal(message)
	select {
	case client.send <- data:
	default:
		close(client.send)
		delete(h.clients, client)
	}
}

func (h *Hub) SendToUser(userID primitive.ObjectID, message Message) {
	roomID := "user_" + userID.Hex()
	h.sendToRoom(roomID, message)
}

func (h *Hub) SendRideUpdate(rideID primitive.ObjectID, message Message) {
	roomID := "ride_" + rideID.Hex()
	h.sendToRoom(roomID, message)
}

func (h *Hub) SendLocationUpdate(driverID primitive.ObjectID, location map[string]interface{}) {
	message := Message{
		Type:      "location_update",
		UserID:    driverID,
		Timestamp: getCurrentTimestamp(),
		Data:      location,
	}

	// Send to active rides room
	h.sendToRoom("active_rides", message)
}

func (h *Hub) joinRoom(client *Client, roomID string) {
	if h.rooms[roomID] == nil {
		h.rooms[roomID] = make(map[*Client]bool)
	}
	h.rooms[roomID][client] = true
	client.rooms[roomID] = true
}

func (h *Hub) LeaveRoom(client *Client, roomID string) {
	h.mutex.Lock()
	defer h.mutex.Unlock()

	if room, exists := h.rooms[roomID]; exists {
		delete(room, client)
		delete(client.rooms, roomID)

		if len(room) == 0 {
			delete(h.rooms, roomID)
		}
	}
}

func (h *Hub) JoinRide(client *Client, rideID primitive.ObjectID) {
	h.mutex.Lock()
	defer h.mutex.Unlock()

	roomID := "ride_" + rideID.Hex()
	h.joinRoom(client, roomID)
}

func getCurrentTimestamp() int64 {
	return time.Now().Unix()
}
