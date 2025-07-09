package mongodb

import (
	"context"
	"fmt"
	"time"

	"goride/internal/models"
	"goride/internal/repositories/interfaces"
	"goride/internal/services"
	"goride/internal/utils"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

type paymentRepository struct {
	collection *mongo.Collection
	cache      services.CacheService
}

func NewPaymentRepository(db *mongo.Database, cache services.CacheService) interfaces.PaymentRepository {
	return &paymentRepository{
		collection: db.Collection("payments"),
		cache:      cache,
	}
}

// Basic CRUD operations
func (r *paymentRepository) Create(ctx context.Context, payment *models.Payment) error {
	payment.ID = primitive.NewObjectID()
	payment.CreatedAt = time.Now()
	payment.UpdatedAt = time.Now()

	_, err := r.collection.InsertOne(ctx, payment)
	if err != nil {
		return fmt.Errorf("failed to create payment: %w", err)
	}

	// Cache completed payments for quick access
	if payment.Status == models.PaymentStatusCompleted {
		r.cachePayment(ctx, payment)
	}

	return nil
}

func (r *paymentRepository) GetByID(ctx context.Context, id primitive.ObjectID) (*models.Payment, error) {
	// Try cache first
	if payment := r.getPaymentFromCache(ctx, id.Hex()); payment != nil {
		return payment, nil
	}

	var payment models.Payment
	err := r.collection.FindOne(ctx, bson.M{"_id": id}).Decode(&payment)
	if err != nil {
		if err == mongo.ErrNoDocuments {
			return nil, fmt.Errorf("payment not found")
		}
		return nil, fmt.Errorf("failed to get payment: %w", err)
	}

	// Cache completed payments
	if payment.Status == models.PaymentStatusCompleted {
		r.cachePayment(ctx, &payment)
	}

	return &payment, nil
}

func (r *paymentRepository) Update(ctx context.Context, id primitive.ObjectID, updates map[string]interface{}) error {
	updates["updated_at"] = time.Now()

	_, err := r.collection.UpdateOne(
		ctx,
		bson.M{"_id": id},
		bson.M{"$set": updates},
	)
	if err != nil {
		return fmt.Errorf("failed to update payment: %w", err)
	}

	// Invalidate cache
	r.invalidatePaymentCache(ctx, id.Hex())

	return nil
}

func (r *paymentRepository) Delete(ctx context.Context, id primitive.ObjectID) error {
	_, err := r.collection.DeleteOne(ctx, bson.M{"_id": id})
	if err != nil {
		return fmt.Errorf("failed to delete payment: %w", err)
	}

	// Invalidate cache
	r.invalidatePaymentCache(ctx, id.Hex())

	return nil
}

// Transaction operations
func (r *paymentRepository) GetByTransactionID(ctx context.Context, transactionID string) (*models.Payment, error) {
	// Try cache first
	cacheKey := fmt.Sprintf("payment_txn_%s", transactionID)
	if r.cache != nil {
		var payment models.Payment
		if err := r.cache.Get(ctx, cacheKey, &payment); err == nil {
			return &payment, nil
		}
	}

	var payment models.Payment
	err := r.collection.FindOne(ctx, bson.M{"transaction_id": transactionID}).Decode(&payment)
	if err != nil {
		if err == mongo.ErrNoDocuments {
			return nil, fmt.Errorf("payment not found with transaction ID")
		}
		return nil, fmt.Errorf("failed to get payment by transaction ID: %w", err)
	}

	// Cache the result
	if payment.Status == models.PaymentStatusCompleted {
		r.cache.Set(ctx, cacheKey, payment, 30*time.Minute)
	}

	return &payment, nil
}

func (r *paymentRepository) GetByExternalID(ctx context.Context, externalID string) (*models.Payment, error) {
	var payment models.Payment
	err := r.collection.FindOne(ctx, bson.M{"external_payment_id": externalID}).Decode(&payment)
	if err != nil {
		if err == mongo.ErrNoDocuments {
			return nil, fmt.Errorf("payment not found with external ID")
		}
		return nil, fmt.Errorf("failed to get payment by external ID: %w", err)
	}

	return &payment, nil
}

func (r *paymentRepository) UpdateStatus(ctx context.Context, id primitive.ObjectID, status models.PaymentStatus) error {
	updates := map[string]interface{}{
		"status": status,
	}

	switch status {
	case models.PaymentStatusCompleted:
		updates["processed_at"] = time.Now()
	case models.PaymentStatusFailed:
		updates["failed_at"] = time.Now()
	case models.PaymentStatusRefunded:
		updates["refunded_at"] = time.Now()
	}

	return r.Update(ctx, id, updates)
}

// Ride association
func (r *paymentRepository) GetByRideID(ctx context.Context, rideID primitive.ObjectID) ([]*models.Payment, error) {
	filter := bson.M{"ride_id": rideID}

	cursor, err := r.collection.Find(ctx, filter, options.Find().SetSort(bson.D{{Key: "created_at", Value: -1}}))
	if err != nil {
		return nil, fmt.Errorf("failed to find payments by ride ID: %w", err)
	}
	defer cursor.Close(ctx)

	var payments []*models.Payment
	for cursor.Next(ctx) {
		var payment models.Payment
		if err := cursor.Decode(&payment); err != nil {
			return nil, fmt.Errorf("failed to decode payment: %w", err)
		}
		payments = append(payments, &payment)
	}

	return payments, nil
}

func (r *paymentRepository) GetPaymentForRide(ctx context.Context, rideID primitive.ObjectID) (*models.Payment, error) {
	filter := bson.M{
		"ride_id":      rideID,
		"payment_type": models.PaymentTypeRide,
		"status": bson.M{"$in": []models.PaymentStatus{
			models.PaymentStatusCompleted,
			models.PaymentStatusPending,
		}},
	}

	var payment models.Payment
	err := r.collection.FindOne(ctx, filter, options.FindOne().SetSort(bson.D{{Key: "created_at", Value: -1}})).Decode(&payment)
	if err != nil {
		if err == mongo.ErrNoDocuments {
			return nil, fmt.Errorf("payment not found for ride")
		}
		return nil, fmt.Errorf("failed to get payment for ride: %w", err)
	}

	return &payment, nil
}

// User payments
func (r *paymentRepository) GetByPayerID(ctx context.Context, payerID primitive.ObjectID, params *utils.PaginationParams) ([]*models.Payment, int64, error) {
	filter := bson.M{"payer_id": payerID}
	return r.findPaymentsWithFilter(ctx, filter, params)
}

func (r *paymentRepository) GetByPayeeID(ctx context.Context, payeeID primitive.ObjectID, params *utils.PaginationParams) ([]*models.Payment, int64, error) {
	filter := bson.M{"payee_id": payeeID}
	return r.findPaymentsWithFilter(ctx, filter, params)
}

// Payment method filtering
func (r *paymentRepository) GetByPaymentMethod(ctx context.Context, method models.PaymentMethod, params *utils.PaginationParams) ([]*models.Payment, int64, error) {
	filter := bson.M{"payment_method": method}
	return r.findPaymentsWithFilter(ctx, filter, params)
}

func (r *paymentRepository) GetByPaymentType(ctx context.Context, paymentType models.PaymentType, params *utils.PaginationParams) ([]*models.Payment, int64, error) {
	filter := bson.M{"payment_type": paymentType}
	return r.findPaymentsWithFilter(ctx, filter, params)
}

// Status filtering
func (r *paymentRepository) GetByStatus(ctx context.Context, status models.PaymentStatus, params *utils.PaginationParams) ([]*models.Payment, int64, error) {
	filter := bson.M{"status": status}
	return r.findPaymentsWithFilter(ctx, filter, params)
}

func (r *paymentRepository) GetPendingPayments(ctx context.Context) ([]*models.Payment, error) {
	filter := bson.M{"status": models.PaymentStatusPending}

	cursor, err := r.collection.Find(ctx, filter, options.Find().SetSort(bson.D{{Key: "created_at", Value: 1}}))
	if err != nil {
		return nil, fmt.Errorf("failed to find pending payments: %w", err)
	}
	defer cursor.Close(ctx)

	var payments []*models.Payment
	for cursor.Next(ctx) {
		var payment models.Payment
		if err := cursor.Decode(&payment); err != nil {
			return nil, fmt.Errorf("failed to decode payment: %w", err)
		}
		payments = append(payments, &payment)
	}

	return payments, nil
}

func (r *paymentRepository) GetFailedPayments(ctx context.Context, params *utils.PaginationParams) ([]*models.Payment, int64, error) {
	filter := bson.M{"status": models.PaymentStatusFailed}
	return r.findPaymentsWithFilter(ctx, filter, params)
}

// Time-based queries
func (r *paymentRepository) GetPaymentsByDateRange(ctx context.Context, startDate, endDate time.Time, params *utils.PaginationParams) ([]*models.Payment, int64, error) {
	filter := bson.M{
		"created_at": bson.M{
			"$gte": startDate,
			"$lte": endDate,
		},
	}
	return r.findPaymentsWithFilter(ctx, filter, params)
}

func (r *paymentRepository) GetDailyPayments(ctx context.Context, date time.Time) ([]*models.Payment, error) {
	startOfDay := time.Date(date.Year(), date.Month(), date.Day(), 0, 0, 0, 0, date.Location())
	endOfDay := startOfDay.Add(24 * time.Hour)

	filter := bson.M{
		"created_at": bson.M{
			"$gte": startOfDay,
			"$lt":  endOfDay,
		},
	}

	cursor, err := r.collection.Find(ctx, filter, options.Find().SetSort(bson.D{{Key: "created_at", Value: -1}}))
	if err != nil {
		return nil, fmt.Errorf("failed to find daily payments: %w", err)
	}
	defer cursor.Close(ctx)

	var payments []*models.Payment
	for cursor.Next(ctx) {
		var payment models.Payment
		if err := cursor.Decode(&payment); err != nil {
			return nil, fmt.Errorf("failed to decode payment: %w", err)
		}
		payments = append(payments, &payment)
	}

	return payments, nil
}

// Refund operations
func (r *paymentRepository) ProcessRefund(ctx context.Context, id primitive.ObjectID, refundAmount float64, reason string) error {
	updates := map[string]interface{}{
		"status":         models.PaymentStatusRefunded,
		"refund_amount":  refundAmount,
		"failure_reason": reason,
		"refunded_at":    time.Now(),
	}

	return r.Update(ctx, id, updates)
}

func (r *paymentRepository) GetRefunds(ctx context.Context, params *utils.PaginationParams) ([]*models.Payment, int64, error) {
	filter := bson.M{
		"status":        models.PaymentStatusRefunded,
		"refund_amount": bson.M{"$gt": 0},
	}
	return r.findPaymentsWithFilter(ctx, filter, params)
}

// Financial reporting
func (r *paymentRepository) GetRevenueStats(ctx context.Context, startDate, endDate time.Time) (map[string]interface{}, error) {
	pipeline := mongo.Pipeline{
		{{"$match", bson.M{
			"status": models.PaymentStatusCompleted,
			"created_at": bson.M{
				"$gte": startDate,
				"$lte": endDate,
			},
		}}},
		{{"$group", bson.M{
			"_id":                   nil,
			"total_revenue":         bson.M{"$sum": "$total_amount"},
			"platform_fees":         bson.M{"$sum": "$platform_fee"},
			"driver_earnings":       bson.M{"$sum": "$driver_earnings"},
			"tax_collected":         bson.M{"$sum": "$tax_amount"},
			"total_transactions":    bson.M{"$sum": 1},
			"avg_transaction_value": bson.M{"$avg": "$total_amount"},
		}}},
	}

	cursor, err := r.collection.Aggregate(ctx, pipeline)
	if err != nil {
		return nil, fmt.Errorf("failed to get revenue stats: %w", err)
	}
	defer cursor.Close(ctx)

	var result struct {
		TotalRevenue        float64 `bson:"total_revenue"`
		PlatformFees        float64 `bson:"platform_fees"`
		DriverEarnings      float64 `bson:"driver_earnings"`
		TaxCollected        float64 `bson:"tax_collected"`
		TotalTransactions   int64   `bson:"total_transactions"`
		AvgTransactionValue float64 `bson:"avg_transaction_value"`
	}

	if cursor.Next(ctx) {
		if err := cursor.Decode(&result); err != nil {
			return nil, fmt.Errorf("failed to decode revenue stats: %w", err)
		}
	}

	return map[string]interface{}{
		"total_revenue":         result.TotalRevenue,
		"platform_fees":         result.PlatformFees,
		"driver_earnings":       result.DriverEarnings,
		"tax_collected":         result.TaxCollected,
		"total_transactions":    result.TotalTransactions,
		"avg_transaction_value": result.AvgTransactionValue,
		"start_date":            startDate,
		"end_date":              endDate,
	}, nil
}

func (r *paymentRepository) GetPaymentStats(ctx context.Context, startDate, endDate time.Time) (map[string]interface{}, error) {
	pipeline := mongo.Pipeline{
		{{"$match", bson.M{
			"created_at": bson.M{
				"$gte": startDate,
				"$lte": endDate,
			},
		}}},
		{{"$group", bson.M{
			"_id":          "$status",
			"count":        bson.M{"$sum": 1},
			"total_amount": bson.M{"$sum": "$total_amount"},
		}}},
	}

	cursor, err := r.collection.Aggregate(ctx, pipeline)
	if err != nil {
		return nil, fmt.Errorf("failed to get payment stats: %w", err)
	}
	defer cursor.Close(ctx)

	stats := make(map[string]interface{})
	var totalCount int64
	var totalAmount float64

	for cursor.Next(ctx) {
		var result struct {
			ID          models.PaymentStatus `bson:"_id"`
			Count       int64                `bson:"count"`
			TotalAmount float64              `bson:"total_amount"`
		}

		if err := cursor.Decode(&result); err != nil {
			return nil, fmt.Errorf("failed to decode payment stats: %w", err)
		}

		stats[string(result.ID)] = map[string]interface{}{
			"count":        result.Count,
			"total_amount": result.TotalAmount,
		}

		totalCount += result.Count
		totalAmount += result.TotalAmount
	}

	// Calculate success rate
	completedCount := int64(0)
	if completed, exists := stats[string(models.PaymentStatusCompleted)]; exists {
		if countMap, ok := completed.(map[string]interface{}); ok {
			if count, ok := countMap["count"].(int64); ok {
				completedCount = count
			}
		}
	}

	successRate := float64(0)
	if totalCount > 0 {
		successRate = float64(completedCount) / float64(totalCount) * 100
	}

	stats["summary"] = map[string]interface{}{
		"total_count":  totalCount,
		"total_amount": totalAmount,
		"success_rate": successRate,
		"start_date":   startDate,
		"end_date":     endDate,
	}

	return stats, nil
}

func (r *paymentRepository) GetDriverEarnings(ctx context.Context, driverID primitive.ObjectID, startDate, endDate time.Time) (map[string]interface{}, error) {
	pipeline := mongo.Pipeline{
		{{"$match", bson.M{
			"payee_id": driverID,
			"status":   models.PaymentStatusCompleted,
			"created_at": bson.M{
				"$gte": startDate,
				"$lte": endDate,
			},
		}}},
		{{"$group", bson.M{
			"_id":                   nil,
			"total_earnings":        bson.M{"$sum": "$driver_earnings"},
			"total_rides":           bson.M{"$sum": 1},
			"total_tips":            bson.M{"$sum": "$tip_amount"},
			"avg_earnings_per_ride": bson.M{"$avg": "$driver_earnings"},
		}}},
	}

	cursor, err := r.collection.Aggregate(ctx, pipeline)
	if err != nil {
		return nil, fmt.Errorf("failed to get driver earnings: %w", err)
	}
	defer cursor.Close(ctx)

	var result struct {
		TotalEarnings      float64 `bson:"total_earnings"`
		TotalRides         int64   `bson:"total_rides"`
		TotalTips          float64 `bson:"total_tips"`
		AvgEarningsPerRide float64 `bson:"avg_earnings_per_ride"`
	}

	if cursor.Next(ctx) {
		if err := cursor.Decode(&result); err != nil {
			return nil, fmt.Errorf("failed to decode driver earnings: %w", err)
		}
	}

	return map[string]interface{}{
		"driver_id":             driverID,
		"total_earnings":        result.TotalEarnings,
		"total_rides":           result.TotalRides,
		"total_tips":            result.TotalTips,
		"avg_earnings_per_ride": result.AvgEarningsPerRide,
		"start_date":            startDate,
		"end_date":              endDate,
	}, nil
}

// Analytics
func (r *paymentRepository) GetTotalRevenue(ctx context.Context, startDate, endDate time.Time) (float64, error) {
	pipeline := mongo.Pipeline{
		{{"$match", bson.M{
			"status": models.PaymentStatusCompleted,
			"created_at": bson.M{
				"$gte": startDate,
				"$lte": endDate,
			},
		}}},
		{{"$group", bson.M{
			"_id":           nil,
			"total_revenue": bson.M{"$sum": "$total_amount"},
		}}},
	}

	cursor, err := r.collection.Aggregate(ctx, pipeline)
	if err != nil {
		return 0, fmt.Errorf("failed to get total revenue: %w", err)
	}
	defer cursor.Close(ctx)

	var result struct {
		TotalRevenue float64 `bson:"total_revenue"`
	}

	if cursor.Next(ctx) {
		if err := cursor.Decode(&result); err != nil {
			return 0, fmt.Errorf("failed to decode total revenue: %w", err)
		}
	}

	return result.TotalRevenue, nil
}

func (r *paymentRepository) GetAverageRideValue(ctx context.Context, days int) (float64, error) {
	startDate := time.Now().AddDate(0, 0, -days)

	pipeline := mongo.Pipeline{
		{{"$match", bson.M{
			"status":       models.PaymentStatusCompleted,
			"payment_type": models.PaymentTypeRide,
			"created_at":   bson.M{"$gte": startDate},
		}}},
		{{"$group", bson.M{
			"_id":       nil,
			"avg_value": bson.M{"$avg": "$total_amount"},
		}}},
	}

	cursor, err := r.collection.Aggregate(ctx, pipeline)
	if err != nil {
		return 0, fmt.Errorf("failed to get average ride value: %w", err)
	}
	defer cursor.Close(ctx)

	var result struct {
		AvgValue float64 `bson:"avg_value"`
	}

	if cursor.Next(ctx) {
		if err := cursor.Decode(&result); err != nil {
			return 0, fmt.Errorf("failed to decode average ride value: %w", err)
		}
	}

	return result.AvgValue, nil
}

func (r *paymentRepository) GetPaymentMethodStats(ctx context.Context, days int) (map[string]int64, error) {
	startDate := time.Now().AddDate(0, 0, -days)

	pipeline := mongo.Pipeline{
		{{"$match", bson.M{
			"status":     models.PaymentStatusCompleted,
			"created_at": bson.M{"$gte": startDate},
		}}},
		{{"$group", bson.M{
			"_id":   "$payment_method",
			"count": bson.M{"$sum": 1},
		}}},
	}

	cursor, err := r.collection.Aggregate(ctx, pipeline)
	if err != nil {
		return nil, fmt.Errorf("failed to get payment method stats: %w", err)
	}
	defer cursor.Close(ctx)

	stats := make(map[string]int64)

	for cursor.Next(ctx) {
		var result struct {
			ID    models.PaymentMethod `bson:"_id"`
			Count int64                `bson:"count"`
		}

		if err := cursor.Decode(&result); err != nil {
			return nil, fmt.Errorf("failed to decode payment method stats: %w", err)
		}

		stats[string(result.ID)] = result.Count
	}

	return stats, nil
}

// Helper methods
func (r *paymentRepository) findPaymentsWithFilter(ctx context.Context, filter bson.M, params *utils.PaginationParams) ([]*models.Payment, int64, error) {
	// Add search filter if provided
	if params.Search != "" {
		searchFields := []string{"transaction_id", "external_payment_id", "promo_code"}
		searchFilter := params.GetSearchFilter(searchFields)
		if len(searchFilter) > 0 {
			filter = bson.M{
				"$and": []bson.M{filter, searchFilter},
			}
		}
	}

	// Get total count
	total, err := r.collection.CountDocuments(ctx, filter)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to count payments: %w", err)
	}

	// Get paginated results
	opts := params.GetSortOptions()
	cursor, err := r.collection.Find(ctx, filter, opts)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to find payments: %w", err)
	}
	defer cursor.Close(ctx)

	var payments []*models.Payment
	for cursor.Next(ctx) {
		var payment models.Payment
		if err := cursor.Decode(&payment); err != nil {
			return nil, 0, fmt.Errorf("failed to decode payment: %w", err)
		}
		payments = append(payments, &payment)
	}

	return payments, total, nil
}

// Cache operations
func (r *paymentRepository) cachePayment(ctx context.Context, payment *models.Payment) {
	if r.cache != nil && payment.Status == models.PaymentStatusCompleted {
		cacheKey := fmt.Sprintf("payment:%s", payment.ID.Hex())
		r.cache.Set(ctx, cacheKey, payment, 30*time.Minute)

		// Also cache by transaction ID
		if payment.TransactionID != "" {
			txnKey := fmt.Sprintf("payment_txn_%s", payment.TransactionID)
			r.cache.Set(ctx, txnKey, payment, 30*time.Minute)
		}
	}
}

func (r *paymentRepository) getPaymentFromCache(ctx context.Context, paymentID string) *models.Payment {
	if r.cache == nil {
		return nil
	}

	cacheKey := fmt.Sprintf("payment:%s", paymentID)
	var payment models.Payment
	err := r.cache.Get(ctx, cacheKey, &payment)
	if err != nil {
		return nil
	}

	return &payment
}

func (r *paymentRepository) invalidatePaymentCache(ctx context.Context, paymentID string) {
	if r.cache != nil {
		cacheKey := fmt.Sprintf("payment:%s", paymentID)
		r.cache.Delete(ctx, cacheKey)

		// Note: We can't easily invalidate the transaction ID cache without knowing the transaction ID
		// This is a trade-off for performance vs cache consistency
	}
}
