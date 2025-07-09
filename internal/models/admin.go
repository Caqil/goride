package models

import (
	"time"

	"go.mongodb.org/mongo-driver/bson/primitive"
)
type AdminRole string
type AdminPermission string

const (
	AdminRoleSuperAdmin AdminRole = "super_admin"
	AdminRoleAdmin      AdminRole = "admin"
	AdminRoleModerator  AdminRole = "moderator"
	AdminRoleSupport    AdminRole = "support"
	AdminRoleAnalyst    AdminRole = "analyst"

	PermissionUserManagement    AdminPermission = "user_management"
	PermissionDriverManagement  AdminPermission = "driver_management"
	PermissionRideManagement    AdminPermission = "ride_management"
	PermissionPaymentManagement AdminPermission = "payment_management"
	PermissionAnalytics         AdminPermission = "analytics"
	PermissionSettings          AdminPermission = "settings"
	PermissionSupport           AdminPermission = "support"
	PermissionDisputes          AdminPermission = "disputes"
	PermissionPromotions        AdminPermission = "promotions"
	PermissionReports           AdminPermission = "reports"
)

type Admin struct {
	ID            primitive.ObjectID `json:"id" bson:"_id,omitempty"`
	UserID        primitive.ObjectID `json:"user_id" bson:"user_id" validate:"required"`
	Role          AdminRole          `json:"role" bson:"role" validate:"required"`
	Permissions   []AdminPermission  `json:"permissions" bson:"permissions"`
	Department    string             `json:"department" bson:"department"`
	IsActive      bool               `json:"is_active" bson:"is_active" default:"true"`
	LastLoginAt   *time.Time         `json:"last_login_at" bson:"last_login_at"`
	CreatedBy     primitive.ObjectID `json:"created_by" bson:"created_by"`
	CreatedAt     time.Time          `json:"created_at" bson:"created_at"`
	UpdatedAt     time.Time          `json:"updated_at" bson:"updated_at"`
	DeactivatedAt *time.Time         `json:"deactivated_at" bson:"deactivated_at"`
}
