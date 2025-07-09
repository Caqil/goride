package models

import (
	"time"

	"go.mongodb.org/mongo-driver/bson/primitive"
)
type TransactionType string
type TransactionStatus string

const (
	TransactionTypeCredit TransactionType = "credit"
	TransactionTypeDebit  TransactionType = "debit"

	TransactionStatusPending   TransactionStatus = "pending"
	TransactionStatusCompleted TransactionStatus = "completed"
	TransactionStatusFailed    TransactionStatus = "failed"
	TransactionStatusCancelled TransactionStatus = "cancelled"
)

type Transaction struct {
	ID            primitive.ObjectID `json:"id" bson:"_id,omitempty"`
	UserID        primitive.ObjectID `json:"user_id" bson:"user_id" validate:"required"`
	PaymentID     *primitive.ObjectID `json:"payment_id" bson:"payment_id"`
	RideID        *primitive.ObjectID `json:"ride_id" bson:"ride_id"`
	Type          TransactionType    `json:"type" bson:"type" validate:"required"`
	Status        TransactionStatus  `json:"status" bson:"status" default:"pending"`
	Amount        float64            `json:"amount" bson:"amount" validate:"required"`
	Currency      string             `json:"currency" bson:"currency" default:"USD"`
	Description   string             `json:"description" bson:"description"`
	Reference     string             `json:"reference" bson:"reference"`
	BalanceBefore float64            `json:"balance_before" bson:"balance_before"`
	BalanceAfter  float64            `json:"balance_after" bson:"balance_after"`
	ProcessedAt   *time.Time         `json:"processed_at" bson:"processed_at"`
	CreatedAt     time.Time          `json:"created_at" bson:"created_at"`
	UpdatedAt     time.Time          `json:"updated_at" bson:"updated_at"`
}
