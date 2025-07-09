package models

import (
	"time"

	"go.mongodb.org/mongo-driver/bson/primitive"
)
type PromotionType string
type PromotionStatus string

const (
	PromotionTypePercentage PromotionType = "percentage"
	PromotionTypeFixed      PromotionType = "fixed"
	PromotionTypeFreeRide   PromotionType = "free_ride"
	PromotionTypeBOGO       PromotionType = "bogo"

	PromotionStatusActive   PromotionStatus = "active"
	PromotionStatusInactive PromotionStatus = "inactive"
	PromotionStatusExpired  PromotionStatus = "expired"
	PromotionStatusUsed     PromotionStatus = "used"
)

type Promotion struct {
	ID             primitive.ObjectID `json:"id" bson:"_id,omitempty"`
	Code           string             `json:"code" bson:"code" validate:"required"`
	Title          string             `json:"title" bson:"title" validate:"required"`
	Description    string             `json:"description" bson:"description"`
	Type           PromotionType      `json:"type" bson:"type" validate:"required"`
	Status         PromotionStatus    `json:"status" bson:"status" default:"active"`
	DiscountValue  float64            `json:"discount_value" bson:"discount_value" validate:"required"`
	MaxDiscount    float64            `json:"max_discount" bson:"max_discount"`
	MinRideAmount  float64            `json:"min_ride_amount" bson:"min_ride_amount"`
	UsageLimit     int                `json:"usage_limit" bson:"usage_limit"`
	UserLimit      int                `json:"user_limit" bson:"user_limit" default:"1"`
	UsedCount      int                `json:"used_count" bson:"used_count" default:"0"`
	ApplicableRideTypes []RideType    `json:"applicable_ride_types" bson:"applicable_ride_types"`
	ApplicableUserTypes []UserType    `json:"applicable_user_types" bson:"applicable_user_types"`
	ValidFrom      time.Time          `json:"valid_from" bson:"valid_from"`
	ValidUntil     time.Time          `json:"valid_until" bson:"valid_until"`
	IsFirstRideOnly bool              `json:"is_first_ride_only" bson:"is_first_ride_only" default:"false"`
	IsReferralOnly bool               `json:"is_referral_only" bson:"is_referral_only" default:"false"`
	TargetCities   []string           `json:"target_cities" bson:"target_cities"`
	CreatedAt      time.Time          `json:"created_at" bson:"created_at"`
	UpdatedAt      time.Time          `json:"updated_at" bson:"updated_at"`
}