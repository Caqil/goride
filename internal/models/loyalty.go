package models

import (
	"time"

	"go.mongodb.org/mongo-driver/bson/primitive"
)
type LoyaltyProgram struct {
	ID                    primitive.ObjectID `json:"id" bson:"_id,omitempty"`
	Name                  string             `json:"name" bson:"name" validate:"required"`
	Description           string             `json:"description" bson:"description"`
	PointsPerRide         int                `json:"points_per_ride" bson:"points_per_ride" default:"10"`
	PointsPerDollar       float64            `json:"points_per_dollar" bson:"points_per_dollar" default:"1"`
	MinimumRedemption     int                `json:"minimum_redemption" bson:"minimum_redemption" default:"100"`
	RedemptionValue       float64            `json:"redemption_value" bson:"redemption_value" default:"0.01"` // 100 points = $1
	TierBenefits          []TierBenefit      `json:"tier_benefits" bson:"tier_benefits"`
	IsActive              bool               `json:"is_active" bson:"is_active" default:"true"`
	CreatedAt             time.Time          `json:"created_at" bson:"created_at"`
	UpdatedAt             time.Time          `json:"updated_at" bson:"updated_at"`
}

type TierBenefit struct {
	TierName           string  `json:"tier_name" bson:"tier_name"`
	MinimumPoints      int     `json:"minimum_points" bson:"minimum_points"`
	BonusMultiplier    float64 `json:"bonus_multiplier" bson:"bonus_multiplier"`
	PrioritySupport    bool    `json:"priority_support" bson:"priority_support"`
	FreeUpgrades       bool    `json:"free_upgrades" bson:"free_upgrades"`
	CancellationFlex   bool    `json:"cancellation_flex" bson:"cancellation_flex"`
}
