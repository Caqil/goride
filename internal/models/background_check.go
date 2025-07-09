package models

import (
	"time"

	"go.mongodb.org/mongo-driver/bson/primitive"
)

type BackgroundCheckStatus string
type BackgroundCheckType string

const (
	BackgroundCheckStatusPending          BackgroundCheckStatus = "pending"
	BackgroundCheckStatusPendingDocuments BackgroundCheckStatus = "pending_documents"
	BackgroundCheckStatusSubmitted        BackgroundCheckStatus = "submitted"
	BackgroundCheckStatusInProgress       BackgroundCheckStatus = "in_progress"
	BackgroundCheckStatusRequiresReview   BackgroundCheckStatus = "requires_review"
	BackgroundCheckStatusPassed           BackgroundCheckStatus = "passed"
	BackgroundCheckStatusFailed           BackgroundCheckStatus = "failed"
	BackgroundCheckStatusCompleted        BackgroundCheckStatus = "completed"
	BackgroundCheckStatusExpired          BackgroundCheckStatus = "expired"
	BackgroundCheckStatusCancelled        BackgroundCheckStatus = "cancelled"

	BackgroundCheckTypeBasic         BackgroundCheckType = "basic"
	BackgroundCheckTypeStandard      BackgroundCheckType = "standard"
	BackgroundCheckTypeComprehensive BackgroundCheckType = "comprehensive"
	BackgroundCheckTypeRenewal       BackgroundCheckType = "renewal"
)

type BackgroundCheck struct {
	ID                primitive.ObjectID     `json:"id" bson:"_id,omitempty"`
	DriverID          primitive.ObjectID     `json:"driver_id" bson:"driver_id" validate:"required"`
	CheckType         BackgroundCheckType    `json:"check_type" bson:"check_type" validate:"required"`
	Status            BackgroundCheckStatus  `json:"status" bson:"status" default:"pending"`
	RequesterID       primitive.ObjectID     `json:"requester_id" bson:"requester_id"`
	RequiredChecks    []string               `json:"required_checks" bson:"required_checks"`
	CompletedChecks   []string               `json:"completed_checks" bson:"completed_checks"`
	FailedChecks      []string               `json:"failed_checks" bson:"failed_checks"`
	Metadata          map[string]interface{} `json:"metadata" bson:"metadata"`
	PersonalInfo      *PersonalInfo          `json:"personal_info" bson:"personal_info"`
	EmploymentHistory []*EmploymentRecord    `json:"employment_history" bson:"employment_history"`
	CriminalHistory   *CriminalHistoryData   `json:"criminal_history" bson:"criminal_history"`
	MVRecord          *MotorVehicleRecord    `json:"motor_vehicle_record" bson:"motor_vehicle_record"`
	References        []*Reference           `json:"references" bson:"references"`
	Consent           *ConsentData           `json:"consent" bson:"consent"`
	Documents         map[string]string      `json:"documents" bson:"documents"`
	ProviderData      map[string]interface{} `json:"provider_data" bson:"provider_data"`
	ReviewNotes       []string               `json:"review_notes" bson:"review_notes"`
	Score             float64                `json:"score" bson:"score"`
	ExpiresAt         *time.Time             `json:"expires_at" bson:"expires_at"`
	SubmittedAt       *time.Time             `json:"submitted_at" bson:"submitted_at"`
	CompletedAt       *time.Time             `json:"completed_at" bson:"completed_at"`
	CreatedAt         time.Time              `json:"created_at" bson:"created_at"`
	UpdatedAt         time.Time              `json:"updated_at" bson:"updated_at"`
}

type Appeal struct {
	ID             primitive.ObjectID  `json:"id" bson:"_id,omitempty"`
	DriverID       primitive.ObjectID  `json:"driver_id" bson:"driver_id" validate:"required"`
	CheckID        primitive.ObjectID  `json:"check_id" bson:"check_id" validate:"required"`
	AppealType     string              `json:"appeal_type" bson:"appeal_type" validate:"required"`
	Reason         string              `json:"reason" bson:"reason" validate:"required"`
	Evidence       []string            `json:"evidence" bson:"evidence"`
	Status         string              `json:"status" bson:"status" default:"pending"`
	CaseNumber     string              `json:"case_number" bson:"case_number"`
	ContactInfo    string              `json:"contact_info" bson:"contact_info"`
	Priority       string              `json:"priority" bson:"priority" default:"normal"`
	ReviewerID     *primitive.ObjectID `json:"reviewer_id" bson:"reviewer_id"`
	Decision       string              `json:"decision" bson:"decision"`
	Explanation    string              `json:"explanation" bson:"explanation"`
	ActionRequired []string            `json:"action_required" bson:"action_required"`
	NewStatus      *string             `json:"new_status" bson:"new_status"`
	CreatedAt      time.Time           `json:"created_at" bson:"created_at"`
	UpdatedAt      time.Time           `json:"updated_at" bson:"updated_at"`
	ResolvedAt     *time.Time          `json:"resolved_at" bson:"resolved_at"`
}

// Supporting types for background checks
type PersonalInfo struct {
	FirstName   string    `json:"first_name" bson:"first_name"`
	LastName    string    `json:"last_name" bson:"last_name"`
	MiddleName  string    `json:"middle_name" bson:"middle_name"`
	DateOfBirth time.Time `json:"date_of_birth" bson:"date_of_birth"`
	SSN         string    `json:"ssn" bson:"ssn"`
	Address     *Address  `json:"address" bson:"address"`
	Phone       string    `json:"phone" bson:"phone"`
	Email       string    `json:"email" bson:"email"`
}

type Address struct {
	Street  string `json:"street" bson:"street"`
	City    string `json:"city" bson:"city"`
	State   string `json:"state" bson:"state"`
	ZipCode string `json:"zip_code" bson:"zip_code"`
	Country string `json:"country" bson:"country"`
}

type EmploymentRecord struct {
	Company    string     `json:"company" bson:"company"`
	Position   string     `json:"position" bson:"position"`
	StartDate  time.Time  `json:"start_date" bson:"start_date"`
	EndDate    *time.Time `json:"end_date" bson:"end_date"`
	Supervisor string     `json:"supervisor" bson:"supervisor"`
	Phone      string     `json:"phone" bson:"phone"`
	Verified   bool       `json:"verified" bson:"verified"`
}

type CriminalHistoryData struct {
	HasCriminalRecord bool              `json:"has_criminal_record" bson:"has_criminal_record"`
	Records           []*CriminalRecord `json:"records" bson:"records"`
	ConsentGiven      bool              `json:"consent_given" bson:"consent_given"`
}

type CriminalRecord struct {
	Charge       string    `json:"charge" bson:"charge"`
	Date         time.Time `json:"date" bson:"date"`
	Disposition  string    `json:"disposition" bson:"disposition"`
	Jurisdiction string    `json:"jurisdiction" bson:"jurisdiction"`
	CaseNumber   string    `json:"case_number" bson:"case_number"`
}

type MotorVehicleRecord struct {
	LicenseNumber string              `json:"license_number" bson:"license_number"`
	State         string              `json:"state" bson:"state"`
	ExpiryDate    time.Time           `json:"expiry_date" bson:"expiry_date"`
	Violations    []*TrafficViolation `json:"violations" bson:"violations"`
	Accidents     []*Accident         `json:"accidents" bson:"accidents"`
	LicenseStatus string              `json:"license_status" bson:"license_status"`
}

type TrafficViolation struct {
	Date      time.Time  `json:"date" bson:"date"`
	Violation string     `json:"violation" bson:"violation"`
	Fine      float64    `json:"fine" bson:"fine"`
	Points    int        `json:"points" bson:"points"`
	CourtDate *time.Time `json:"court_date" bson:"court_date"`
}

type Accident struct {
	Date        time.Time `json:"date" bson:"date"`
	Description string    `json:"description" bson:"description"`
	AtFault     bool      `json:"at_fault" bson:"at_fault"`
	Injuries    bool      `json:"injuries" bson:"injuries"`
	Damages     float64   `json:"damages" bson:"damages"`
}

type Reference struct {
	Name         string `json:"name" bson:"name"`
	Relationship string `json:"relationship" bson:"relationship"`
	Phone        string `json:"phone" bson:"phone"`
	Email        string `json:"email" bson:"email"`
	Verified     bool   `json:"verified" bson:"verified"`
}

type ConsentData struct {
	ConsentGiven bool      `json:"consent_given" bson:"consent_given"`
	ConsentDate  time.Time `json:"consent_date" bson:"consent_date"`
	IPAddress    string    `json:"ip_address" bson:"ip_address"`
	UserAgent    string    `json:"user_agent" bson:"user_agent"`
}

// Statistics and reporting types
type BackgroundCheckStats struct {
	TotalChecks           int64                     `json:"total_checks"`
	ChecksByStatus        map[string]int64          `json:"checks_by_status"`
	ChecksByType          map[string]int64          `json:"checks_by_type"`
	AverageProcessingTime time.Duration             `json:"average_processing_time"`
	ComplianceRate        float64                   `json:"compliance_rate"`
	ManualReviewRate      float64                   `json:"manual_review_rate"`
	AppealRate            float64                   `json:"appeal_rate"`
	ProviderPerformance   map[string]*ProviderStats `json:"provider_performance"`
}

type ProviderStats struct {
	Provider    string        `json:"provider"`
	TotalChecks int64         `json:"total_checks"`
	SuccessRate float64       `json:"success_rate"`
	AvgTime     time.Duration `json:"avg_time"`
	Reliability float64       `json:"reliability"`
}

// Additional supporting types for reporting
type CheckTypeStats struct {
	Total   int64         `json:"total"`
	Passed  int64         `json:"passed"`
	Failed  int64         `json:"failed"`
	Pending int64         `json:"pending"`
	AvgTime time.Duration `json:"avg_time"`
}

type RegionStats struct {
	Region         string        `json:"region"`
	Total          int64         `json:"total"`
	ComplianceRate float64       `json:"compliance_rate"`
	AvgTime        time.Duration `json:"avg_time"`
}

type ComplianceTrends struct {
	Monthly      map[string]float64 `json:"monthly"`
	Quarterly    map[string]float64 `json:"quarterly"`
	YearOverYear float64            `json:"year_over_year"`
}
