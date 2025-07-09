package interfaces

import (
	"context"
	"time"

	"goride/internal/models"
	"goride/internal/utils"

	"go.mongodb.org/mongo-driver/bson/primitive"
)

type ChatRepository interface {
	// Chat CRUD operations
	CreateChat(ctx context.Context, chat *models.Chat) error
	GetChatByID(ctx context.Context, id primitive.ObjectID) (*models.Chat, error)
	UpdateChat(ctx context.Context, id primitive.ObjectID, updates map[string]interface{}) error
	DeleteChat(ctx context.Context, id primitive.ObjectID) error

	// Chat management
	GetChatByRideID(ctx context.Context, rideID primitive.ObjectID) (*models.Chat, error)
	GetChatsByParticipant(ctx context.Context, participantID primitive.ObjectID, params *utils.PaginationParams) ([]*models.Chat, int64, error)
	CloseChat(ctx context.Context, id primitive.ObjectID) error

	// Message CRUD operations
	CreateMessage(ctx context.Context, message *models.Message) error
	GetMessageByID(ctx context.Context, id primitive.ObjectID) (*models.Message, error)
	UpdateMessage(ctx context.Context, id primitive.ObjectID, updates map[string]interface{}) error
	DeleteMessage(ctx context.Context, id primitive.ObjectID) error

	// Message retrieval
	GetMessagesByChatID(ctx context.Context, chatID primitive.ObjectID, params *utils.PaginationParams) ([]*models.Message, int64, error)
	GetUnreadMessages(ctx context.Context, chatID primitive.ObjectID, userID primitive.ObjectID) ([]*models.Message, error)
	GetMessagesByType(ctx context.Context, chatID primitive.ObjectID, messageType models.MessageType) ([]*models.Message, error)

	// Message status
	UpdateMessageStatus(ctx context.Context, id primitive.ObjectID, status models.MessageStatus) error
	MarkMessageAsRead(ctx context.Context, messageID primitive.ObjectID, userID primitive.ObjectID) error
	MarkAllMessagesAsRead(ctx context.Context, chatID primitive.ObjectID, userID primitive.ObjectID) error

	// Search and filtering
	SearchMessages(ctx context.Context, chatID primitive.ObjectID, query string, params *utils.PaginationParams) ([]*models.Message, int64, error)
	GetMessagesByDateRange(ctx context.Context, chatID primitive.ObjectID, startDate, endDate time.Time, params *utils.PaginationParams) ([]*models.Message, int64, error)

	// Media messages
	GetMediaMessages(ctx context.Context, chatID primitive.ObjectID, mediaType models.MessageType) ([]*models.Message, error)

	// Analytics
	GetChatStats(ctx context.Context, chatID primitive.ObjectID) (map[string]interface{}, error)
	GetMessageStats(ctx context.Context, days int) (map[string]interface{}, error)
	GetUserChatActivity(ctx context.Context, userID primitive.ObjectID, days int) (map[string]interface{}, error)
}
