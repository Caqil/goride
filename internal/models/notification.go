package models

import (
	"time"

	"go.mongodb.org/mongo-driver/bson/primitive"
)

type NotificationType string
type NotificationStatus string

const (
	NotificationTypeRideRequest    NotificationType = "ride_request"
	NotificationTypeRideAccepted   NotificationType = "ride_accepted"
	NotificationTypeDriverArrived  NotificationType = "driver_arrived"
	NotificationTypeRideStarted    NotificationType = "ride_started"
	NotificationTypeRideCompleted  NotificationType = "ride_completed"
	NotificationTypeRideCancelled  NotificationType = "ride_cancelled"
	NotificationTypePaymentSuccess NotificationType = "payment_success"
	NotificationTypePaymentFailed  NotificationType = "payment_failed"
	NotificationTypePromotion      NotificationType = "promotion"
	NotificationTypeEmergency      NotificationType = "emergency"
	NotificationTypeGeneral        NotificationType = "general"

	NotificationStatusExpired NotificationStatus = "expired"
	NotificationStatusUnread  NotificationStatus = "unread"
	NotificationStatusRead    NotificationStatus = "read"
	NotificationStatusSent    NotificationStatus = "sent"
	NotificationStatusFailed  NotificationStatus = "failed"
)

type Notification struct {
	ID            primitive.ObjectID     `json:"id" bson:"_id,omitempty"`
	UserID        primitive.ObjectID     `json:"user_id" bson:"user_id" validate:"required"`
	Type          NotificationType       `json:"type" bson:"type" validate:"required"`
	Status        NotificationStatus     `json:"status" bson:"status" default:"unread"`
	Title         string                 `json:"title" bson:"title" validate:"required"`
	Message       string                 `json:"message" bson:"message" validate:"required"`
	Data          map[string]interface{} `json:"data" bson:"data"`
	ImageURL      string                 `json:"image_url" bson:"image_url"`
	DeepLink      string                 `json:"deep_link" bson:"deep_link"`
	ActionButtons []ActionButton         `json:"action_buttons" bson:"action_buttons"`
	Priority      int                    `json:"priority" bson:"priority" default:"0"`
	ExpiresAt     *time.Time             `json:"expires_at" bson:"expires_at"`
	ReadAt        *time.Time             `json:"read_at" bson:"read_at"`
	SentAt        *time.Time             `json:"sent_at" bson:"sent_at"`
	CreatedAt     time.Time              `json:"created_at" bson:"created_at"`
	UpdatedAt     time.Time              `json:"updated_at" bson:"updated_at"`
}

type ActionButton struct {
	Text   string `json:"text" bson:"text"`
	Action string `json:"action" bson:"action"`
	URL    string `json:"url" bson:"url"`
}
