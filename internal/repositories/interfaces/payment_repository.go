package interfaces

import (
	"context"
	"time"

	"goride/internal/models"
	"goride/internal/utils"

	"go.mongodb.org/mongo-driver/bson/primitive"
)

type PaymentRepository interface {
	// Basic CRUD operations
	Create(ctx context.Context, payment *models.Payment) error
	GetByID(ctx context.Context, id primitive.ObjectID) (*models.Payment, error)
	Update(ctx context.Context, id primitive.ObjectID, updates map[string]interface{}) error
	Delete(ctx context.Context, id primitive.ObjectID) error

	// Transaction operations
	GetByTransactionID(ctx context.Context, transactionID string) (*models.Payment, error)
	GetByExternalID(ctx context.Context, externalID string) (*models.Payment, error)
	UpdateStatus(ctx context.Context, id primitive.ObjectID, status models.PaymentStatus) error

	// Ride association
	GetByRideID(ctx context.Context, rideID primitive.ObjectID) ([]*models.Payment, error)
	GetPaymentForRide(ctx context.Context, rideID primitive.ObjectID) (*models.Payment, error)

	// User payments
	GetByPayerID(ctx context.Context, payerID primitive.ObjectID, params *utils.PaginationParams) ([]*models.Payment, int64, error)
	GetByPayeeID(ctx context.Context, payeeID primitive.ObjectID, params *utils.PaginationParams) ([]*models.Payment, int64, error)

	// Payment method filtering
	GetByPaymentMethod(ctx context.Context, method models.PaymentMethod, params *utils.PaginationParams) ([]*models.Payment, int64, error)
	GetByPaymentType(ctx context.Context, paymentType models.PaymentType, params *utils.PaginationParams) ([]*models.Payment, int64, error)

	// Status filtering
	GetByStatus(ctx context.Context, status models.PaymentStatus, params *utils.PaginationParams) ([]*models.Payment, int64, error)
	GetPendingPayments(ctx context.Context) ([]*models.Payment, error)
	GetFailedPayments(ctx context.Context, params *utils.PaginationParams) ([]*models.Payment, int64, error)

	// Time-based queries
	GetPaymentsByDateRange(ctx context.Context, startDate, endDate time.Time, params *utils.PaginationParams) ([]*models.Payment, int64, error)
	GetDailyPayments(ctx context.Context, date time.Time) ([]*models.Payment, error)

	// Refund operations
	ProcessRefund(ctx context.Context, id primitive.ObjectID, refundAmount float64, reason string) error
	GetRefunds(ctx context.Context, params *utils.PaginationParams) ([]*models.Payment, int64, error)

	// Financial reporting
	GetRevenueStats(ctx context.Context, startDate, endDate time.Time) (map[string]interface{}, error)
	GetPaymentStats(ctx context.Context, startDate, endDate time.Time) (map[string]interface{}, error)
	GetDriverEarnings(ctx context.Context, driverID primitive.ObjectID, startDate, endDate time.Time) (map[string]interface{}, error)

	// Analytics
	GetTotalRevenue(ctx context.Context, startDate, endDate time.Time) (float64, error)
	GetAverageRideValue(ctx context.Context, days int) (float64, error)
	GetPaymentMethodStats(ctx context.Context, days int) (map[string]int64, error)
}
