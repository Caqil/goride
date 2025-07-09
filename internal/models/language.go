package models

import (
	"time"

	"go.mongodb.org/mongo-driver/bson/primitive"
)
type Language struct {
	ID          primitive.ObjectID `json:"id" bson:"_id,omitempty"`
	Code        string             `json:"code" bson:"code" validate:"required"`
	Name        string             `json:"name" bson:"name" validate:"required"`
	NativeName  string             `json:"native_name" bson:"native_name"`
	IsRTL       bool               `json:"is_rtl" bson:"is_rtl" default:"false"`
	IsActive    bool               `json:"is_active" bson:"is_active" default:"true"`
	IsDefault   bool               `json:"is_default" bson:"is_default" default:"false"`
	SortOrder   int                `json:"sort_order" bson:"sort_order" default:"0"`
	CreatedAt   time.Time          `json:"created_at" bson:"created_at"`
	UpdatedAt   time.Time          `json:"updated_at" bson:"updated_at"`
}
