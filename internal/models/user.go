package models

import (
	"time"

	"go.mongodb.org/mongo-driver/bson/primitive"
)

type UserStatus string
type UserType string
type AuthProvider string

const (
	UserStatusActive    UserStatus = "active"
	UserStatusInactive  UserStatus = "inactive"
	UserStatusSuspended UserStatus = "suspended"
	UserStatusBanned    UserStatus = "banned"

	UserTypeRider  UserType = "rider"
	UserTypeDriver UserType = "driver"
	UserTypeAdmin  UserType = "admin"

	AuthProviderEmail    AuthProvider = "email"
	AuthProviderPhone    AuthProvider = "phone"
	AuthProviderGoogle   AuthProvider = "google"
	AuthProviderFacebook AuthProvider = "facebook"
	AuthProviderApple    AuthProvider = "apple"
)

type User struct {
	ID               primitive.ObjectID `json:"id" bson:"_id,omitempty"`
	FirstName        string             `json:"first_name" bson:"first_name" validate:"required,min=2,max=50"`
	LastName         string             `json:"last_name" bson:"last_name" validate:"required,min=2,max=50"`
	Email            string             `json:"email" bson:"email" validate:"required,email"`
	Phone            string             `json:"phone" bson:"phone" validate:"required"`
	CountryCode      string             `json:"country_code" bson:"country_code" validate:"required"`
	Password         string             `json:"-" bson:"password"`
	ProfilePicture   string             `json:"profile_picture" bson:"profile_picture"`
	DateOfBirth      time.Time          `json:"date_of_birth" bson:"date_of_birth"`
	Gender           string             `json:"gender" bson:"gender"`
	Language         string             `json:"language" bson:"language" default:"en"`
	UserType         UserType           `json:"user_type" bson:"user_type" validate:"required"`
	Status           UserStatus         `json:"status" bson:"status" default:"active"`
	AuthProvider     AuthProvider       `json:"auth_provider" bson:"auth_provider" default:"email"`
	SocialID         string             `json:"social_id" bson:"social_id"`
	IsEmailVerified  bool               `json:"is_email_verified" bson:"is_email_verified" default:"false"`
	IsPhoneVerified  bool               `json:"is_phone_verified" bson:"is_phone_verified" default:"false"`
	TwoFactorEnabled bool               `json:"two_factor_enabled" bson:"two_factor_enabled" default:"false"`
	LastLoginAt      *time.Time         `json:"last_login_at" bson:"last_login_at"`
	LastActiveAt     *time.Time         `json:"last_active_at" bson:"last_active_at"`
	CreatedAt        time.Time          `json:"created_at" bson:"created_at"`
	UpdatedAt        time.Time          `json:"updated_at" bson:"updated_at"`
	DeletedAt        *time.Time         `json:"deleted_at" bson:"deleted_at"`
}
