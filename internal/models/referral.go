package models

import (
	"time"

	"go.mongodb.org/mongo-driver/bson/primitive"
)
type ReferralStatus string

const (
	ReferralStatusPending   ReferralStatus = "pending"
	ReferralStatusCompleted ReferralStatus = "completed"
	ReferralStatusExpired   ReferralStatus = "expired"
	ReferralStatusCancelled ReferralStatus = "cancelled"
)

type Referral struct {
	ID             primitive.ObjectID `json:"id" bson:"_id,omitempty"`
	ReferrerID     primitive.ObjectID `json:"referrer_id" bson:"referrer_id" validate:"required"`
	RefereeID      *primitive.ObjectID `json:"referee_id" bson:"referee_id"`
	ReferralCode   string             `json:"referral_code" bson:"referral_code" validate:"required"`
	Status         ReferralStatus     `json:"status" bson:"status" default:"pending"`
	ReferrerReward float64            `json:"referrer_reward" bson:"referrer_reward"`
	RefereeReward  float64            `json:"referee_reward" bson:"referee_reward"`
	RequiredRides  int                `json:"required_rides" bson:"required_rides" default:"1"`
	CompletedRides int                `json:"completed_rides" bson:"completed_rides" default:"0"`
	ReferredAt     time.Time          `json:"referred_at" bson:"referred_at"`
	CompletedAt    *time.Time         `json:"completed_at" bson:"completed_at"`
	ExpiresAt      time.Time          `json:"expires_at" bson:"expires_at"`
	CreatedAt      time.Time          `json:"created_at" bson:"created_at"`
	UpdatedAt      time.Time          `json:"updated_at" bson:"updated_at"`
}
