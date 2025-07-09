package models

import (
	"time"

	"go.mongodb.org/mongo-driver/bson/primitive"
)
type AuditAction string

const (
	AuditActionCreate AuditAction = "create"
	AuditActionRead   AuditAction = "read"
	AuditActionUpdate AuditAction = "update"
	AuditActionDelete AuditAction = "delete"
	AuditActionLogin  AuditAction = "login"
	AuditActionLogout AuditAction = "logout"
)

type AuditLog struct {
	ID           primitive.ObjectID `json:"id" bson:"_id,omitempty"`
	UserID       *primitive.ObjectID `json:"user_id" bson:"user_id"`
	Action       AuditAction        `json:"action" bson:"action" validate:"required"`
	Resource     string             `json:"resource" bson:"resource" validate:"required"`
	ResourceID   string             `json:"resource_id" bson:"resource_id"`
	OldValues    map[string]interface{} `json:"old_values" bson:"old_values"`
	NewValues    map[string]interface{} `json:"new_values" bson:"new_values"`
	IPAddress    string             `json:"ip_address" bson:"ip_address"`
	UserAgent    string             `json:"user_agent" bson:"user_agent"`
	Location     *Location          `json:"location" bson:"location"`
	Metadata     map[string]interface{} `json:"metadata" bson:"metadata"`
	CreatedAt    time.Time          `json:"created_at" bson:"created_at"`
}