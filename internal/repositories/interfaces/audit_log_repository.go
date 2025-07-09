package interfaces

import (
	"context"
	"time"

	"github.com/uber-clone/internal/models"
	"github.com/uber-clone/internal/utils"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

type AuditLogRepository interface {
	// Basic operations
	Create(ctx context.Context, auditLog *models.AuditLog) error
	GetByID(ctx context.Context, id primitive.ObjectID) (*models.AuditLog, error)
	
	// User actions
	GetByUserID(ctx context.Context, userID primitive.ObjectID, params *utils.PaginationParams) ([]*models.AuditLog, int64, error)
	GetUserActions(ctx context.Context, userID primitive.ObjectID, action models.AuditAction, params *utils.PaginationParams) ([]*models.AuditLog, int64, error)
	
	// Resource tracking
	GetByResource(ctx context.Context, resource string, params *utils.PaginationParams) ([]*models.AuditLog, int64, error)
	GetByResourceAndAction(ctx context.Context, resource string, action models.AuditAction, params *utils.PaginationParams) ([]*models.AuditLog, int64, error)
	GetResourceHistory(ctx context.Context, resource, resourceID string, params *utils.PaginationParams) ([]*models.AuditLog, int64, error)
	
	// Action filtering
	GetByAction(ctx context.Context, action models.AuditAction, params *utils.PaginationParams) ([]*models.AuditLog, int64, error)
	
	// Time-based queries
	GetByDateRange(ctx context.Context, startDate, endDate time.Time, params *utils.PaginationParams) ([]*models.AuditLog, int64, error)
	GetRecentLogs(ctx context.Context, hours int, params *utils.PaginationParams) ([]*models.AuditLog, int64, error)
	
	// Security monitoring
	GetSecurityEvents(ctx context.Context, params *utils.PaginationParams) ([]*models.AuditLog, int64, error)
	GetFailedLoginAttempts(ctx context.Context, ipAddress string, hours int) ([]*models.AuditLog, error)
	GetSuspiciousActivities(ctx context.Context, params *utils.PaginationParams) ([]*models.AuditLog, int64, error)
	
	// IP and location tracking
	GetByIPAddress(ctx context.Context, ipAddress string, params *utils.PaginationParams) ([]*models.AuditLog, int64, error)
	GetLoginsByLocation(ctx context.Context, userID primitive.ObjectID, days int) ([]map[string]interface{}, error)
	
	// Data compliance
	GetDataAccessLogs(ctx context.Context, resource string, days int) ([]*models.AuditLog, error)
	GetDataModificationLogs(ctx context.Context, startDate, endDate time.Time, params *utils.PaginationParams) ([]*models.AuditLog, int64, error)
	
	// Cleanup operations
	DeleteOldLogs(ctx context.Context, days int) error
	ArchiveLogs(ctx context.Context, beforeDate time.Time) error
	
	// Analytics
	GetAuditStats(ctx context.Context, days int) (map[string]interface{}, error)
	GetUserActivitySummary(ctx context.Context, userID primitive.ObjectID, days int) (map[string]interface{}, error)
	GetSystemUsageStats(ctx context.Context, days int) (map[string]interface{}, error)
}
