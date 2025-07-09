package interfaces

import (
	"context"
	"time"

	"github.com/uber-clone/internal/models"
	"github.com/uber-clone/internal/utils"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

type NotificationRepository interface {
	// Basic CRUD operations
	Create(ctx context.Context, notification *models.Notification) error
	GetByID(ctx context.Context, id primitive.ObjectID) (*models.Notification, error)
	Update(ctx context.Context, id primitive.ObjectID, updates map[string]interface{}) error
	Delete(ctx context.Context, id primitive.ObjectID) error
	
	// User notifications
	GetByUserID(ctx context.Context, userID primitive.ObjectID, params *utils.PaginationParams) ([]*models.Notification, int64, error)
	GetUnreadByUserID(ctx context.Context, userID primitive.ObjectID) ([]*models.Notification, error)
	GetUnreadCount(ctx context.Context, userID primitive.ObjectID) (int64, error)
	
	// Status operations
	MarkAsRead(ctx context.Context, id primitive.ObjectID) error
	MarkAllAsRead(ctx context.Context, userID primitive.ObjectID) error
	UpdateStatus(ctx context.Context, id primitive.ObjectID, status models.NotificationStatus) error
	
	// Type filtering
	GetByType(ctx context.Context, notificationType models.NotificationType, params *utils.PaginationParams) ([]*models.Notification, int64, error)
	GetByUserAndType(ctx context.Context, userID primitive.ObjectID, notificationType models.NotificationType, params *utils.PaginationParams) ([]*models.Notification, int64, error)
	
	// Time-based operations
	GetExpiredNotifications(ctx context.Context) ([]*models.Notification, error)
	DeleteExpiredNotifications(ctx context.Context) error
	GetNotificationsByDateRange(ctx context.Context, startDate, endDate time.Time, params *utils.PaginationParams) ([]*models.Notification, int64, error)
	
	// Bulk operations
	CreateBatch(ctx context.Context, notifications []*models.Notification) error
	DeleteByUserID(ctx context.Context, userID primitive.ObjectID) error
	DeleteOldNotifications(ctx context.Context, days int) error
	
	// Analytics
	GetNotificationStats(ctx context.Context, days int) (map[string]interface{}, error)
	GetDeliveryStats(ctx context.Context, days int) (map[string]interface{}, error)
	GetEngagementStats(ctx context.Context, userID primitive.ObjectID, days int) (map[string]interface{}, error)
}