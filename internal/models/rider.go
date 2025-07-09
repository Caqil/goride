package models

import (
	"time"

	"go.mongodb.org/mongo-driver/bson/primitive"
)

type Rider struct {
	ID                 primitive.ObjectID   `json:"id" bson:"_id,omitempty"`
	UserID             primitive.ObjectID   `json:"user_id" bson:"user_id" validate:"required"`
	Rating             float64              `json:"rating" bson:"rating" default:"0"`
	TotalRatings       int64                `json:"total_ratings" bson:"total_ratings" default:"0"`
	TotalRides         int64                `json:"total_rides" bson:"total_rides" default:"0"`
	TotalSpent         float64              `json:"total_spent" bson:"total_spent" default:"0"`
	FavoriteLocations  []FavoriteLocation   `json:"favorite_locations" bson:"favorite_locations"`
	PaymentMethods     []primitive.ObjectID `json:"payment_methods" bson:"payment_methods"`
	DefaultPaymentID   *primitive.ObjectID  `json:"default_payment_id" bson:"default_payment_id"`
	EmergencyContacts  []EmergencyContact   `json:"emergency_contacts" bson:"emergency_contacts"`
	AccessibilityNeeds []string             `json:"accessibility_needs" bson:"accessibility_needs"`
	RidePreferences    *RidePreferences     `json:"ride_preferences" bson:"ride_preferences"`
	LoyaltyPoints      int64                `json:"loyalty_points" bson:"loyalty_points" default:"0"`
	ReferralCode       string               `json:"referral_code" bson:"referral_code"`
	ReferredBy         *primitive.ObjectID  `json:"referred_by" bson:"referred_by"`
	CreatedAt          time.Time            `json:"created_at" bson:"created_at"`
	UpdatedAt          time.Time            `json:"updated_at" bson:"updated_at"`
}

type FavoriteLocation struct {
	Name      string    `json:"name" bson:"name"`
	Address   string    `json:"address" bson:"address"`
	Location  Location  `json:"location" bson:"location"`
	Type      string    `json:"type" bson:"type"` // home, work, other
	CreatedAt time.Time `json:"created_at" bson:"created_at"`
}

type RidePreferences struct {
	PreferredRideTypes  []string `json:"preferred_ride_types" bson:"preferred_ride_types"`
	Temperature         int      `json:"temperature" bson:"temperature"`
	MusicPreference     string   `json:"music_preference" bson:"music_preference"`
	ConversationLevel   string   `json:"conversation_level" bson:"conversation_level"`
	AllowPetFriendly    bool     `json:"allow_pet_friendly" bson:"allow_pet_friendly"`
	AllowSharedRides    bool     `json:"allow_shared_rides" bson:"allow_shared_rides"`
	PreferFemaleDrivers bool     `json:"prefer_female_drivers" bson:"prefer_female_drivers"`
}
