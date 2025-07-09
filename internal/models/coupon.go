package models

import (
	"time"

	"go.mongodb.org/mongo-driver/bson/primitive"
)
type CouponStatus string

const (
	CouponStatusAvailable CouponStatus = "available"
	CouponStatusUsed      CouponStatus = "used"
	CouponStatusExpired   CouponStatus = "expired"
	CouponStatusRevoked   CouponStatus = "revoked"
)

type Coupon struct {
	ID           primitive.ObjectID `json:"id" bson:"_id,omitempty"`
	UserID       primitive.ObjectID `json:"user_id" bson:"user_id" validate:"required"`
	PromotionID  primitive.ObjectID `json:"promotion_id" bson:"promotion_id" validate:"required"`
	Code         string             `json:"code" bson:"code" validate:"required"`
	Status       CouponStatus       `json:"status" bson:"status" default:"available"`
	UsedRideID   *primitive.ObjectID `json:"used_ride_id" bson:"used_ride_id"`
	AssignedAt   time.Time          `json:"assigned_at" bson:"assigned_at"`
	UsedAt       *time.Time         `json:"used_at" bson:"used_at"`
	ExpiresAt    time.Time          `json:"expires_at" bson:"expires_at"`
	CreatedAt    time.Time          `json:"created_at" bson:"created_at"`
	UpdatedAt    time.Time          `json:"updated_at" bson:"updated_at"`
}