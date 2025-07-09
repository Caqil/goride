package mongodb

import (
	"context"
	"fmt"
	"time"

	"goride/internal/models"
	"goride/internal/repositories/interfaces"
	"goride/internal/utils"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

type paymentRepository struct {
	collection *mongo.Collection
	cache      CacheService
}

func NewPaymentRepository(db *mongo.Database, cache CacheService) interfaces.PaymentRepository {
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
	cacheKey := fmt.Sprintf("payment_tx_%s", transactionID)
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
	if r.cache != nil {
		r.cache.Set(ctx, cacheKey, &payment, 30*time.Minute)
	}

	return &payment, nil
}

func (r *paymentRepository) GetByExternalID(ctx context.Context, externalID string) (*models.Payment, error) {
	var payment models.Payment
	err := r.collection.FindOne(ctx, bson.M{"external_id": externalID}).Decode(&payment)
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

	// Add status-specific timestamps
	switch status {
	case models.PaymentStatusCompleted:
		updates["completed_at"] = time.Now()
	case models.PaymentStatusFailed:
		updates["failed_at"] = time.Now()
	case models.PaymentStatusCancelled:
		updates["cancelled_at"] = time.Now()
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
	var payment models.Payment
	err := r.collection.FindOne(
		ctx,
		bson.M{
			"ride_id": rideID,
			"type":    models.PaymentTypeRide,
		},
		options.FindOne().SetSort(bson.D{{Key: "created_at", Value: -1}}),
	).Decode(&payment)
	if err != nil {
		if err == mongo.ErrNoDocuments {
			return nil, fmt.Errorf("no payment found for ride")
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
	filter := bson.M{"type": paymentType}
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
	startOfDay := utils.GetStartOfDay(date)
	endOfDay := utils.GetEndOfDay(date)

	filter := bson.M{
		"created_at": bson.M{
			"$gte": startOfDay,
			"$lte": endOfDay,
		},
	}

	cursor, err := r.collection.Find(ctx, filter, options.Find().SetSort(bson.D{{Key: "created_at", Value: 1}}))
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
	// Start a transaction to ensure consistency
	session, err := r.collection.Database().Client().StartSession()
	if err != nil {
		return fmt.Errorf("failed to start session: %w", err)
	}
	defer session.EndSession(ctx)

	err = mongo.WithSession(ctx, session, func(sc mongo.SessionContext) error {
		// Get the original payment
		payment, err := r.GetByID(sc, id)
		if err != nil {
			return err
		}

		// Validate refund amount
		if refundAmount > payment.Amount {
			return fmt.Errorf("refund amount cannot be greater than original payment amount")
		}

		// Create refund record
		refund := &models.Payment{
			ID:            primitive.NewObjectID(),
			RideID:        payment.RideID,
			PayerID:       payment.PayeeID, // Company pays back to customer
			PayeeID:       payment.PayerID, // Customer receives refund
			Amount:        refundAmount,
			Currency:      payment.Currency,
			PaymentMethod: payment.PaymentMethod,
			PaymentType:          models.PaymentTypeRefund,
			Status:        models.PaymentStatusCompleted,
			Description:   fmt.Sprintf("Refund for payment %s: %s", payment.ID.Hex(), reason),
			Metadata: map[string]interface{}{
				"original_payment_id": payment.ID,
				"refund_reason":       reason,
			},
			CreatedAt:   time.Now(),
			UpdatedAt:   time.Now(),
			CompletedAt: &[]time.Time{time.Now()}[0],
		}

		// Insert refund record
		_, err = r.collection.InsertOne(sc, refund)
		if err != nil {
			return fmt.Errorf("failed to create refund record: %w", err)
		}

		// Update original payment status
		updates := map[string]interface{}{
			"status":        models.PaymentStatusRefunded,
			"refunded_at":   time.Now(),
			"refund_amount": refundAmount,
			"refund_reason": reason,
		}

		_, err = r.collection.UpdateOne(
			sc,
			bson.M{"_id": id},
			bson.M{"$set": updates},
		)
		if err != nil {
			return fmt.Errorf("failed to update original payment: %w", err)
		}

		return nil
	})

	if err != nil {
		return fmt.Errorf("failed to process refund: %w", err)
	}

	// Invalidate cache
	r.invalidatePaymentCache(ctx, id.Hex())

	return nil
}

func (r *paymentRepository) GetRefunds(ctx context.Context, params *utils.PaginationParams) ([]*models.Payment, int64, error) {
	filter := bson.M{"type": models.PaymentTypeRefund}
	return r.findPaymentsWithFilter(ctx, filter, params)
}

// Financial reporting
func (r *paymentRepository) GetRevenueStats(ctx context.Context, startDate, endDate time.Time) (map[string]interface{}, error) {
	pipeline := mongo.Pipeline{
		{{"$match", bson.M{
			"created_at": bson.M{
				"$gte": startDate,
				"$lte": endDate,
			},
			"status": models.PaymentStatusCompleted,
			"type":   bson.M{"$ne": models.PaymentTypeRefund}, // Exclude refunds
		}}},
		{{"$group", bson.M{
			"_id":            nil,
			"total_revenue":  bson.M{"$sum": "$amount"},
			"total_payments": bson.M{"$sum": 1},
			"avg_payment":    bson.M{"$avg": "$amount"},
			"min_payment":    bson.M{"$min": "$amount"},
			"max_payment":    bson.M{"$max": "$amount"},
		}}},
	}

	cursor, err := r.collection.Aggregate(ctx, pipeline)
	if err != nil {
		return nil, fmt.Errorf("failed to get revenue stats: %w", err)
	}
	defer cursor.Close(ctx)

	var result struct {
		TotalRevenue  float64 `bson:"total_revenue"`
		TotalPayments int64   `bson:"total_payments"`
		AvgPayment    float64 `bson:"avg_payment"`
		MinPayment    float64 `bson:"min_payment"`
		MaxPayment    float64 `bson:"max_payment"`
	}

	if cursor.Next(ctx) {
		if err := cursor.Decode(&result); err != nil {
			return nil, fmt.Errorf("failed to decode revenue stats: %w", err)
		}
	}

	// Get refunds for the same period
	refundPipeline := mongo.Pipeline{
		{{"$match", bson.M{
			"created_at": bson.M{
				"$gte": startDate,
				"$lte": endDate,
			},
			"type":   models.PaymentTypeRefund,
			"status": models.PaymentStatusCompleted,
		}}},
		{{"$group", bson.M{
			"_id":           nil,
			"total_refunds": bson.M{"$sum": "$amount"},
			"refund_count":  bson.M{"$sum": 1},
		}}},
	}

	cursor, err = r.collection.Aggregate(ctx, refundPipeline)
	if err != nil {
		return nil, fmt.Errorf("failed to get refund stats: %w", err)
	}
	defer cursor.Close(ctx)

	var refundResult struct {
		TotalRefunds float64 `bson:"total_refunds"`
		RefundCount  int64   `bson:"refund_count"`
	}

	if cursor.Next(ctx) {
		cursor.Decode(&refundResult)
	}

	netRevenue := result.TotalRevenue - refundResult.TotalRefunds

	return map[string]interface{}{
		"total_revenue":  result.TotalRevenue,
		"total_refunds":  refundResult.TotalRefunds,
		"net_revenue":    netRevenue,
		"total_payments": result.TotalPayments,
		"refund_count":   refundResult.RefundCount,
		"avg_payment":    result.AvgPayment,
		"min_payment":    result.MinPayment,
		"max_payment":    result.MaxPayment,
		"start_date":     startDate,
		"end_date":       endDate,
	}, nil
}

func (r *paymentRepository) GetPaymentStats(ctx context.Context, startDate, endDate time.Time) (map[string]interface{}, error) {
	// Payments by method
	methodPipeline := mongo.Pipeline{
		{{"$match", bson.M{
			"created_at": bson.M{
				"$gte": startDate,
				"$lte": endDate,
			},
			"status": models.PaymentStatusCompleted,
		}}},
		{{"$group", bson.M{
			"_id":          "$payment_method",
			"count":        bson.M{"$sum": 1},
			"total_amount": bson.M{"$sum": "$amount"},
		}}},
	}

	cursor, err := r.collection.Aggregate(ctx, methodPipeline)
	if err != nil {
		return nil, fmt.Errorf("failed to get payment method stats: %w", err)
	}
	defer cursor.Close(ctx)

	methodStats := make(map[string]map[string]interface{})
	for cursor.Next(ctx) {
		var result struct {
			Method      models.PaymentMethod `bson:"_id"`
			Count       int64                `bson:"count"`
			TotalAmount float64              `bson:"total_amount"`
		}

		if err := cursor.Decode(&result); err != nil {
			return nil, fmt.Errorf("failed to decode method stats: %w", err)
		}

		methodStats[string(result.Method)] = map[string]interface{}{
			"count":        result.Count,
			"total_amount": result.TotalAmount,
		}
	}

	// Payments by status
	statusPipeline := mongo.Pipeline{
		{{"$match", bson.M{
			"created_at": bson.M{
				"$gte": startDate,
				"$lte": endDate,
			},
		}}},
		{{"$group", bson.M{
			"_id":   "$status",
			"count": bson.M{"$sum": 1},
		}}},
	}

	cursor, err = r.collection.Aggregate(ctx, statusPipeline)
	if err != nil {
		return nil, fmt.Errorf("failed to get payment status stats: %w", err)
	}
	defer cursor.Close(ctx)

	statusStats := make(map[string]int64)
	for cursor.Next(ctx) {
		var result struct {
			Status models.PaymentStatus `bson:"_id"`
			Count  int64                `bson:"count"`
		}

		if err := cursor.Decode(&result); err != nil {
			return nil, fmt.Errorf("failed to decode status stats: %w", err)
		}

		statusStats[string(result.Status)] = result.Count
	}

	return map[string]interface{}{
		"method_stats": methodStats,
		"status_stats": statusStats,
		"start_date":   startDate,
		"end_date":     endDate,
	}, nil
}

func (r *paymentRepository) GetDriverEarnings(ctx context.Context, driverID primitive.ObjectID, startDate, endDate time.Time) (map[string]interface{}, error) {
	pipeline := mongo.Pipeline{
		{{"$match", bson.M{
			"payee_id": driverID,
			"created_at": bson.M{
				"$gte": startDate,
				"$lte": endDate,
			},
			"status": models.PaymentStatusCompleted,
			"type":   models.PaymentTypeDriverEarnings,
		}}},
		{{"$group", bson.M{
			"_id":                  nil,
			"total_earnings":       bson.M{"$sum": "$amount"},
			"total_rides":          bson.M{"$sum": 1},
			"avg_earning_per_ride": bson.M{"$avg": "$amount"},
		}}},
	}

	cursor, err := r.collection.Aggregate(ctx, pipeline)
	if err != nil {
		return nil, fmt.Errorf("failed to get driver earnings: %w", err)
	}
	defer cursor.Close(ctx)

	var result struct {
		TotalEarnings     float64 `bson:"total_earnings"`
		TotalRides        int64   `bson:"total_rides"`
		AvgEarningPerRide float64 `bson:"avg_earning_per_ride"`
	}

	if cursor.Next(ctx) {
		if err := cursor.Decode(&result); err != nil {
			return nil, fmt.Errorf("failed to decode driver earnings: %w", err)
		}
	}

	// Daily breakdown
	dailyPipeline := mongo.Pipeline{
		{{"$match", bson.M{
			"payee_id": driverID,
			"created_at": bson.M{
				"$gte": startDate,
				"$lte": endDate,
			},
			"status": models.PaymentStatusCompleted,
			"type":   models.PaymentTypeDriverEarnings,
		}}},
		{{"$group", bson.M{
			"_id": bson.M{
				"date": bson.M{"$dateToString": bson.M{
					"format": "%Y-%m-%d",
					"date":   "$created_at",
				}},
			},
			"daily_earnings": bson.M{"$sum": "$amount"},
			"daily_rides":    bson.M{"$sum": 1},
		}}},
		{{"$sort", bson.M{"_id.date": 1}}},
	}

	cursor, err = r.collection.Aggregate(ctx, dailyPipeline)
	if err != nil {
		return nil, fmt.Errorf("failed to get daily earnings: %w", err)
	}
	defer cursor.Close(ctx)

	var dailyBreakdown []map[string]interface{}
	for cursor.Next(ctx) {
		var dailyResult struct {
			ID struct {
				Date string `bson:"date"`
			} `bson:"_id"`
			DailyEarnings float64 `bson:"daily_earnings"`
			DailyRides    int64   `bson:"daily_rides"`
		}

		if err := cursor.Decode(&dailyResult); err != nil {
			return nil, fmt.Errorf("failed to decode daily earnings: %w", err)
		}

		dailyBreakdown = append(dailyBreakdown, map[string]interface{}{
			"date":           dailyResult.ID.Date,
			"daily_earnings": dailyResult.DailyEarnings,
			"daily_rides":    dailyResult.DailyRides,
		})
	}

	return map[string]interface{}{
		"driver_id":            driverID,
		"total_earnings":       result.TotalEarnings,
		"total_rides":          result.TotalRides,
		"avg_earning_per_ride": result.AvgEarningPerRide,
		"daily_breakdown":      dailyBreakdown,
		"start_date":           startDate,
		"end_date":             endDate,
	}, nil
}

// Analytics
func (r *paymentRepository) GetTotalRevenue(ctx context.Context, startDate, endDate time.Time) (float64, error) {
	pipeline := mongo.Pipeline{
		{{"$match", bson.M{
			"created_at": bson.M{
				"$gte": startDate,
				"$lte": endDate,
			},
			"status": models.PaymentStatusCompleted,
			"type":   bson.M{"$ne": models.PaymentTypeRefund},
		}}},
		{{"$group", bson.M{
			"_id":   nil,
			"total": bson.M{"$sum": "$amount"},
		}}},
	}

	cursor, err := r.collection.Aggregate(ctx, pipeline)
	if err != nil {
		return 0, fmt.Errorf("failed to get total revenue: %w", err)
	}
	defer cursor.Close(ctx)

	var result struct {
		Total float64 `bson:"total"`
	}

	if cursor.Next(ctx) {
		if err := cursor.Decode(&result); err != nil {
			return 0, fmt.Errorf("failed to decode total revenue: %w", err)
		}
	}

	return result.Total, nil
}

func (r *paymentRepository) GetAverageRideValue(ctx context.Context, days int) (float64, error) {
	startDate := time.Now().AddDate(0, 0, -days)

	pipeline := mongo.Pipeline{
		{{"$match", bson.M{
			"created_at": bson.M{"$gte": startDate},
			"status":     models.PaymentStatusCompleted,
			"type":       models.PaymentTypeRide,
		}}},
		{{"$group", bson.M{
			"_id": nil,
			"avg": bson.M{"$avg": "$amount"},
		}}},
	}

	cursor, err := r.collection.Aggregate(ctx, pipeline)
	if err != nil {
		return 0, fmt.Errorf("failed to get average ride value: %w", err)
	}
	defer cursor.Close(ctx)

	var result struct {
		Avg float64 `bson:"avg"`
	}

	if cursor.Next(ctx) {
		if err := cursor.Decode(&result); err != nil {
			return 0, fmt.Errorf("failed to decode average ride value: %w", err)
		}
	}

	return result.Avg, nil
}

func (r *paymentRepository) GetPaymentMethodStats(ctx context.Context, days int) (map[string]int64, error) {
	startDate := time.Now().AddDate(0, 0, -days)

	pipeline := mongo.Pipeline{
		{{"$match", bson.M{
			"created_at": bson.M{"$gte": startDate},
			"status":     models.PaymentStatusCompleted,
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
			Method models.PaymentMethod `bson:"_id"`
			Count  int64                `bson:"count"`
		}

		if err := cursor.Decode(&result); err != nil {
			return nil, fmt.Errorf("failed to decode payment method stats: %w", err)
		}

		stats[string(result.Method)] = result.Count
	}

	return stats, nil
}

// Helper methods
func (r *paymentRepository) findPaymentsWithFilter(ctx context.Context, filter bson.M, params *utils.PaginationParams) ([]*models.Payment, int64, error) {
	// Add search filter if provided
	if params.Search != "" {
		searchFields := []string{"transaction_id", "external_id", "description"}
		filter = bson.M{
			"$and": []bson.M{
				filter,
				params.GetSearchFilter(searchFields),
			},
		}
	}

	// Get total count
	total, err := r.collection.CountDocuments(ctx, filter)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to count payments: %w", err)
	}

	// Get paginated results
	opts := params.GetSortOptions()
	// Default sort by created_at descending for payments
	if params.Sort == "created_at" || params.Sort == "" {
		opts.SetSort(bson.D{{Key: "created_at", Value: -1}})
	}

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
		r.cache.Set(ctx, cacheKey, payment, 1*time.Hour)
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
	}
}
