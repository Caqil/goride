package models

import (
	"time"

	"go.mongodb.org/mongo-driver/bson/primitive"
)
type PaymentStatus string
type PaymentMethod string
type PaymentType string

const (
	PaymentStatusPending   PaymentStatus = "pending"
	PaymentStatusCompleted PaymentStatus = "completed"
	PaymentStatusFailed    PaymentStatus = "failed"
	PaymentStatusRefunded  PaymentStatus = "refunded"
	PaymentStatusCancelled PaymentStatus = "cancelled"

	PaymentMethodCreditCard PaymentMethod = "credit_card"
	PaymentMethodDebitCard  PaymentMethod = "debit_card"
	PaymentMethodPayPal     PaymentMethod = "paypal"
	PaymentMethodApplePay   PaymentMethod = "apple_pay"
	PaymentMethodGooglePay  PaymentMethod = "google_pay"
	PaymentMethodCash       PaymentMethod = "cash"
	PaymentMethodWallet     PaymentMethod = "wallet"

	PaymentTypeRide     PaymentType = "ride"
	PaymentTypeTip      PaymentType = "tip"
	PaymentTypeRefund   PaymentType = "refund"
	PaymentTypePenalty  PaymentType = "penalty"
	PaymentTypeBonus    PaymentType = "bonus"
)

type Payment struct {
	ID                primitive.ObjectID `json:"id" bson:"_id,omitempty"`
	RideID            primitive.ObjectID `json:"ride_id" bson:"ride_id" validate:"required"`
	PayerID           primitive.ObjectID `json:"payer_id" bson:"payer_id" validate:"required"`
	PayeeID           primitive.ObjectID `json:"payee_id" bson:"payee_id"`
	PaymentMethodID   primitive.ObjectID `json:"payment_method_id" bson:"payment_method_id"`
	TransactionID     string             `json:"transaction_id" bson:"transaction_id"`
	ExternalID        string             `json:"external_id" bson:"external_id"`
	PaymentMethod     PaymentMethod      `json:"payment_method" bson:"payment_method" validate:"required"`
	PaymentType       PaymentType        `json:"payment_type" bson:"payment_type" default:"ride"`
	Status            PaymentStatus      `json:"status" bson:"status" default:"pending"`
	Amount            float64            `json:"amount" bson:"amount" validate:"required"`
	Currency          string             `json:"currency" bson:"currency" default:"USD"`
	BaseFare          float64            `json:"base_fare" bson:"base_fare"`
	DistanceFare      float64            `json:"distance_fare" bson:"distance_fare"`
	TimeFare          float64            `json:"time_fare" bson:"time_fare"`
	SurgeAmount       float64            `json:"surge_amount" bson:"surge_amount" default:"0"`
	TipAmount         float64            `json:"tip_amount" bson:"tip_amount" default:"0"`
	TaxAmount         float64            `json:"tax_amount" bson:"tax_amount" default:"0"`
	DiscountAmount    float64            `json:"discount_amount" bson:"discount_amount" default:"0"`
	PlatformFee       float64            `json:"platform_fee" bson:"platform_fee" default:"0"`
	DriverEarnings    float64            `json:"driver_earnings" bson:"driver_earnings"`
	PromoCode         string             `json:"promo_code" bson:"promo_code"`
	FailureReason     string             `json:"failure_reason" bson:"failure_reason"`
	RefundAmount      float64            `json:"refund_amount" bson:"refund_amount" default:"0"`
	ProcessedAt       *time.Time         `json:"processed_at" bson:"processed_at"`
	FailedAt          *time.Time         `json:"failed_at" bson:"failed_at"`
	RefundedAt        *time.Time         `json:"refunded_at" bson:"refunded_at"`
	CreatedAt         time.Time          `json:"created_at" bson:"created_at"`
	UpdatedAt         time.Time          `json:"updated_at" bson:"updated_at"`
}