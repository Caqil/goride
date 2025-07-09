package services

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"goride/internal/models"
	"goride/internal/repositories/interfaces"
	"goride/internal/utils"
	"goride/pkg/logger"

	"go.mongodb.org/mongo-driver/bson/primitive"
)

type BackgroundCheckService interface {
	// Background check initiation
	InitiateBackgroundCheck(ctx context.Context, request *BackgroundCheckRequest) (*BackgroundCheckResponse, error)
	SubmitBackgroundCheckData(ctx context.Context, checkID primitive.ObjectID, data *BackgroundCheckData) error
	UpdateBackgroundCheckStatus(ctx context.Context, checkID primitive.ObjectID, status models.BackgroundCheckStatus, notes string) error

	// Driver onboarding integration
	ProcessDriverBackgroundCheck(ctx context.Context, driverID primitive.ObjectID) (*BackgroundCheckResponse, error)
	GetDriverBackgroundCheckStatus(ctx context.Context, driverID primitive.ObjectID) (*DriverBackgroundCheckStatus, error)
	RequireAdditionalDocuments(ctx context.Context, checkID primitive.ObjectID, documents []string) error

	// Background check retrieval
	GetBackgroundCheckByID(ctx context.Context, id primitive.ObjectID) (*models.BackgroundCheck, error)
	GetBackgroundChecksByDriver(ctx context.Context, driverID primitive.ObjectID, params *utils.PaginationParams) ([]*models.BackgroundCheck, int64, error)
	GetBackgroundChecksByStatus(ctx context.Context, status models.BackgroundCheckStatus, params *utils.PaginationParams) ([]*models.BackgroundCheck, int64, error)

	// Document verification
	VerifyDocument(ctx context.Context, request *DocumentVerificationRequest) (*DocumentVerificationResponse, error)
	UploadVerificationDocument(ctx context.Context, checkID primitive.ObjectID, documentType string, fileURL string) error
	GetRequiredDocuments(ctx context.Context, userType string, region string) ([]RequiredDocument, error)

	// Third-party provider integration
	SubmitToProvider(ctx context.Context, checkID primitive.ObjectID, providerName string) (*ProviderSubmissionResponse, error)
	ProcessProviderWebhook(ctx context.Context, providerName string, payload []byte) error
	GetProviderStatus(ctx context.Context, checkID primitive.ObjectID, providerName string) (*ProviderStatus, error)

	// Quality assurance and review
	AssignForManualReview(ctx context.Context, checkID primitive.ObjectID, reviewerID primitive.ObjectID) error
	CompleteManualReview(ctx context.Context, checkID primitive.ObjectID, decision *ReviewDecision) error
	GetPendingReviews(ctx context.Context, reviewerID primitive.ObjectID, params *utils.PaginationParams) ([]*models.BackgroundCheck, int64, error)

	// Compliance and reporting
	GenerateComplianceReport(ctx context.Context, params *ComplianceReportParams) (*ComplianceReport, error)
	GetBackgroundCheckStats(ctx context.Context, startDate, endDate time.Time) (*BackgroundCheckStats, error)
	SchedulePeriodicRechecks(ctx context.Context) error

	// Appeals and disputes
	SubmitAppeal(ctx context.Context, request *AppealRequest) (*AppealResponse, error)
	ProcessAppeal(ctx context.Context, appealID primitive.ObjectID, decision *AppealDecision) error
	GetAppealsByDriver(ctx context.Context, driverID primitive.ObjectID) ([]*models.Appeal, error)
}

type backgroundCheckService struct {
	backgroundCheckRepo interfaces.BackgroundCheckRepository
	userRepo            interfaces.UserRepository
	driverRepo          interfaces.DriverRepository
	auditLogRepo        interfaces.AuditLogRepository
	notificationRepo    interfaces.NotificationRepository
	cache               CacheService
	storageService      StorageService
	logger              *logger.Logger
	providers           map[string]BackgroundCheckProvider
}

type StorageService interface {
	UploadFile(ctx context.Context, key string, data []byte, contentType string) (string, error)
	GetFileURL(ctx context.Context, key string) (string, error)
	DeleteFile(ctx context.Context, key string) error
}

type BackgroundCheckProvider interface {
	SubmitCheck(ctx context.Context, request *ProviderCheckRequest) (*ProviderCheckResponse, error)
	GetStatus(ctx context.Context, checkID string) (*ProviderStatus, error)
	CancelCheck(ctx context.Context, checkID string) error
}

type BackgroundCheckRequest struct {
	DriverID       primitive.ObjectID         `json:"driver_id" validate:"required"`
	CheckType      models.BackgroundCheckType `json:"check_type" validate:"required"`
	RequesterID    primitive.ObjectID         `json:"requester_id" validate:"required"`
	Priority       string                     `json:"priority"`
	RequiredChecks []string                   `json:"required_checks"`
	Metadata       map[string]interface{}     `json:"metadata"`
	ExpiryDays     int                        `json:"expiry_days"`
}

type BackgroundCheckResponse struct {
	CheckID       primitive.ObjectID `json:"check_id"`
	Status        string             `json:"status"`
	EstimatedTime time.Duration      `json:"estimated_time"`
	RequiredDocs  []RequiredDocument `json:"required_documents"`
	NextSteps     []string           `json:"next_steps"`
	TrackingCode  string             `json:"tracking_code"`
	ExpiresAt     time.Time          `json:"expires_at"`
}

type BackgroundCheckData struct {
	PersonalInfo       *models.PersonalInfo        `json:"personal_info"`
	EmploymentHistory  []*models.EmploymentRecord  `json:"employment_history"`
	CriminalHistory    *models.CriminalHistoryData `json:"criminal_history"`
	MotorVehicleRecord *models.MotorVehicleRecord  `json:"motor_vehicle_record"`
	References         []*models.Reference         `json:"references"`
	Consent            *models.ConsentData         `json:"consent"`
	Documents          map[string]string           `json:"documents"`
}

type DriverBackgroundCheckStatus struct {
	DriverID        primitive.ObjectID           `json:"driver_id"`
	CurrentCheck    *models.BackgroundCheck      `json:"current_check"`
	CheckHistory    []*models.BackgroundCheck    `json:"check_history"`
	OverallStatus   models.BackgroundCheckStatus `json:"overall_status"`
	LastCheckDate   time.Time                    `json:"last_check_date"`
	NextCheckDue    *time.Time                   `json:"next_check_due"`
	ComplianceScore float64                      `json:"compliance_score"`
	RedFlags        []string                     `json:"red_flags"`
	RequiredActions []string                     `json:"required_actions"`
}

type DocumentVerificationRequest struct {
	CheckID          primitive.ObjectID `json:"check_id" validate:"required"`
	DocumentType     string             `json:"document_type" validate:"required"`
	DocumentNumber   string             `json:"document_number"`
	FileURL          string             `json:"file_url" validate:"required"`
	ExpiryDate       *time.Time         `json:"expiry_date"`
	IssueDate        *time.Time         `json:"issue_date"`
	IssuingAuthority string             `json:"issuing_authority"`
}

type DocumentVerificationResponse struct {
	VerificationID       primitive.ObjectID     `json:"verification_id"`
	Status               string                 `json:"status"`
	VerifiedData         map[string]interface{} `json:"verified_data"`
	DiscrepanciesFound   []string               `json:"discrepancies_found"`
	ConfidenceScore      float64                `json:"confidence_score"`
	ManualReviewRequired bool                   `json:"manual_review_required"`
	ProcessingTime       time.Duration          `json:"processing_time"`
}

type RequiredDocument struct {
	Type        string   `json:"type"`
	Name        string   `json:"name"`
	Description string   `json:"description"`
	Required    bool     `json:"required"`
	Formats     []string `json:"formats"`
	MaxSizeMB   int      `json:"max_size_mb"`
	ExampleURL  string   `json:"example_url"`
}

type ProviderSubmissionResponse struct {
	ProviderCheckID     string    `json:"provider_check_id"`
	Status              string    `json:"status"`
	EstimatedCompletion time.Time `json:"estimated_completion"`
	WebhookURL          string    `json:"webhook_url"`
	TrackingURL         string    `json:"tracking_url"`
}

type ProviderStatus struct {
	CheckID         string                 `json:"check_id"`
	Status          string                 `json:"status"`
	Progress        float64                `json:"progress"`
	CompletedChecks []string               `json:"completed_checks"`
	PendingChecks   []string               `json:"pending_checks"`
	Results         map[string]interface{} `json:"results"`
	LastUpdated     time.Time              `json:"last_updated"`
}

type ReviewDecision struct {
	ReviewerID      primitive.ObjectID `json:"reviewer_id"`
	Decision        string             `json:"decision"`
	Notes           string             `json:"notes"`
	RequiredActions []string           `json:"required_actions"`
	NextReviewDate  *time.Time         `json:"next_review_date"`
	Confidence      float64            `json:"confidence"`
}

type ComplianceReportParams struct {
	StartDate      time.Time `json:"start_date"`
	EndDate        time.Time `json:"end_date"`
	IncludeDrivers bool      `json:"include_drivers"`
	Region         string    `json:"region"`
	CheckTypes     []string  `json:"check_types"`
	StatusFilter   []string  `json:"status_filter"`
}

type ComplianceReport struct {
	GeneratedAt           time.Time                         `json:"generated_at"`
	Period                string                            `json:"period"`
	TotalChecks           int64                             `json:"total_checks"`
	PassedChecks          int64                             `json:"passed_checks"`
	FailedChecks          int64                             `json:"failed_checks"`
	PendingChecks         int64                             `json:"pending_checks"`
	ComplianceRate        float64                           `json:"compliance_rate"`
	AverageProcessingTime time.Duration                     `json:"average_processing_time"`
	ByCheckType           map[string]*models.CheckTypeStats `json:"by_check_type"`
	ByRegion              map[string]*models.RegionStats    `json:"by_region"`
	TrendAnalysis         *models.ComplianceTrends          `json:"trend_analysis"`
	Recommendations       []string                          `json:"recommendations"`
}

type BackgroundCheckStats struct {
	TotalChecks           int64                            `json:"total_checks"`
	ChecksByStatus        map[string]int64                 `json:"checks_by_status"`
	ChecksByType          map[string]int64                 `json:"checks_by_type"`
	AverageProcessingTime time.Duration                    `json:"average_processing_time"`
	ComplianceRate        float64                          `json:"compliance_rate"`
	ManualReviewRate      float64                          `json:"manual_review_rate"`
	AppealRate            float64                          `json:"appeal_rate"`
	ProviderPerformance   map[string]*models.ProviderStats `json:"provider_performance"`
}

type AppealRequest struct {
	DriverID    primitive.ObjectID `json:"driver_id" validate:"required"`
	CheckID     primitive.ObjectID `json:"check_id" validate:"required"`
	AppealType  string             `json:"appeal_type" validate:"required"`
	Reason      string             `json:"reason" validate:"required"`
	Evidence    []string           `json:"evidence"`
	ContactInfo string             `json:"contact_info"`
	Priority    string             `json:"priority"`
}

type AppealResponse struct {
	AppealID      primitive.ObjectID `json:"appeal_id"`
	Status        string             `json:"status"`
	CaseNumber    string             `json:"case_number"`
	EstimatedTime time.Duration      `json:"estimated_time"`
	ContactInfo   string             `json:"contact_info"`
	NextSteps     []string           `json:"next_steps"`
}

type AppealDecision struct {
	ReviewerID     primitive.ObjectID `json:"reviewer_id"`
	Decision       string             `json:"decision"`
	Explanation    string             `json:"explanation"`
	ActionRequired []string           `json:"action_required"`
	NewStatus      *string            `json:"new_status"`
}

type ProviderCheckRequest struct {
	CheckID      string                 `json:"check_id"`
	PersonalInfo *models.PersonalInfo   `json:"personal_info"`
	CheckTypes   []string               `json:"check_types"`
	CallbackURL  string                 `json:"callback_url"`
	Metadata     map[string]interface{} `json:"metadata"`
}

type ProviderCheckResponse struct {
	ProviderCheckID string `json:"provider_check_id"`
	Status          string `json:"status"`
	EstimatedTime   string `json:"estimated_time"`
	TrackingURL     string `json:"tracking_url"`
}

func NewBackgroundCheckService(
	backgroundCheckRepo interfaces.BackgroundCheckRepository,
	userRepo interfaces.UserRepository,
	driverRepo interfaces.DriverRepository,
	auditLogRepo interfaces.AuditLogRepository,
	notificationRepo interfaces.NotificationRepository,
	cache CacheService,
	storageService StorageService,
	logger *logger.Logger,
) BackgroundCheckService {
	return &backgroundCheckService{
		backgroundCheckRepo: backgroundCheckRepo,
		userRepo:            userRepo,
		driverRepo:          driverRepo,
		auditLogRepo:        auditLogRepo,
		notificationRepo:    notificationRepo,
		cache:               cache,
		storageService:      storageService,
		logger:              logger,
		providers:           make(map[string]BackgroundCheckProvider),
	}
}

func (s *backgroundCheckService) InitiateBackgroundCheck(ctx context.Context, request *BackgroundCheckRequest) (*BackgroundCheckResponse, error) {
	// Validate request
	if err := utils.ValidateStruct(request); err != nil {
		return nil, fmt.Errorf("validation failed: %w", err)
	}

	// Get driver information
	driver, err := s.driverRepo.GetByID(ctx, request.DriverID)
	if err != nil {
		return nil, fmt.Errorf("failed to get driver: %w", err)
	}

	// Get user information
	user, err := s.userRepo.GetByID(ctx, driver.UserID)
	if err != nil {
		return nil, fmt.Errorf("failed to get user: %w", err)
	}

	// Create background check record
	backgroundCheck := &models.BackgroundCheck{
		DriverID:       request.DriverID,
		CheckType:      request.CheckType,
		Status:         models.BackgroundCheckStatusPending,
		RequesterID:    request.RequesterID,
		RequiredChecks: request.RequiredChecks,
		Metadata:       request.Metadata,
		CreatedAt:      time.Now(),
		UpdatedAt:      time.Now(),
	}

	if request.ExpiryDays > 0 {
		expiryDate := time.Now().AddDate(0, 0, request.ExpiryDays)
		backgroundCheck.ExpiresAt = &expiryDate
	}

	// Save to database
	if err := s.backgroundCheckRepo.Create(ctx, backgroundCheck); err != nil {
		s.logger.WithError(err).Error("Failed to create background check")
		return nil, fmt.Errorf("failed to create background check: %w", err)
	}

	// Generate tracking code
	trackingCode := fmt.Sprintf("BC-%s", backgroundCheck.ID.Hex()[:8])

	// Get required documents based on check type and user location
	requiredDocs, err := s.GetRequiredDocuments(ctx, string(user.UserType), "US") // Default to US, could be enhanced with location service
	if err != nil {
		s.logger.WithError(err).Warn("Failed to get required documents")
		requiredDocs = []RequiredDocument{}
	}

	// Create audit log
	s.auditLogRepo.Create(ctx, &models.AuditLog{
		UserID:     &request.RequesterID,
		Action:     models.AuditActionCreate,
		Resource:   "background_check",
		ResourceID: backgroundCheck.ID.Hex(),
		NewValues: map[string]interface{}{
			"driver_id":  request.DriverID,
			"check_type": request.CheckType,
		},
		CreatedAt: time.Now(),
	})

	// Send notification to driver
	s.notificationRepo.Create(ctx, &models.Notification{
		UserID:  request.DriverID,
		Type:    models.NotificationTypeBackgroundCheck,
		Title:   "Background Check Initiated",
		Message: "A background check has been initiated for your account. Please provide the required documents.",
		Data: map[string]interface{}{
			"check_id":      backgroundCheck.ID,
			"tracking_code": trackingCode,
		},
		CreatedAt: time.Now(),
	})

	s.logger.WithField("check_id", backgroundCheck.ID.Hex()).
		WithField("driver_id", request.DriverID.Hex()).
		Info("Background check initiated")

	return &BackgroundCheckResponse{
		CheckID:       backgroundCheck.ID,
		Status:        string(backgroundCheck.Status),
		EstimatedTime: s.calculateEstimatedTime(request.CheckType, request.RequiredChecks),
		RequiredDocs:  requiredDocs,
		NextSteps:     s.generateNextSteps(backgroundCheck),
		TrackingCode:  trackingCode,
		ExpiresAt:     *backgroundCheck.ExpiresAt,
	}, nil
}

func (s *backgroundCheckService) SubmitBackgroundCheckData(ctx context.Context, checkID primitive.ObjectID, data *BackgroundCheckData) error {
	// Get background check
	check, err := s.backgroundCheckRepo.GetByID(ctx, checkID)
	if err != nil {
		return fmt.Errorf("failed to get background check: %w", err)
	}

	// Validate data completeness
	if err := s.validateBackgroundCheckData(data, check.RequiredChecks); err != nil {
		return fmt.Errorf("data validation failed: %w", err)
	}

	// Update background check with submitted data
	updates := map[string]interface{}{
		"personal_info":        data.PersonalInfo,
		"employment_history":   data.EmploymentHistory,
		"criminal_history":     data.CriminalHistory,
		"motor_vehicle_record": data.MotorVehicleRecord,
		"references":           data.References,
		"consent":              data.Consent,
		"documents":            data.Documents,
		"status":               models.BackgroundCheckStatusSubmitted,
		"submitted_at":         time.Now(),
		"updated_at":           time.Now(),
	}

	if err := s.backgroundCheckRepo.Update(ctx, checkID, updates); err != nil {
		return fmt.Errorf("failed to update background check: %w", err)
	}

	// Process the submitted data
	go s.processSubmittedData(ctx, checkID, data)

	s.logger.WithField("check_id", checkID.Hex()).Info("Background check data submitted")
	return nil
}

func (s *backgroundCheckService) ProcessDriverBackgroundCheck(ctx context.Context, driverID primitive.ObjectID) (*BackgroundCheckResponse, error) {
	// Create a comprehensive background check for driver onboarding
	request := &BackgroundCheckRequest{
		DriverID:    driverID,
		CheckType:   models.BackgroundCheckTypeComprehensive,
		RequesterID: primitive.NewObjectID(), // System initiated
		Priority:    "high",
		RequiredChecks: []string{
			"identity_verification",
			"criminal_history",
			"motor_vehicle_record",
			"employment_verification",
			"reference_check",
		},
		ExpiryDays: 365, // 1 year
	}

	return s.InitiateBackgroundCheck(ctx, request)
}

func (s *backgroundCheckService) GetDriverBackgroundCheckStatus(ctx context.Context, driverID primitive.ObjectID) (*DriverBackgroundCheckStatus, error) {
	// Get current and historical background checks for driver
	params := &utils.PaginationParams{
		Page:     1,
		PageSize: 10,
		Sort:     "created_at",
		Order:    "desc",
	}

	checks, _, err := s.backgroundCheckRepo.GetByDriverID(ctx, driverID, params)
	if err != nil {
		return nil, fmt.Errorf("failed to get driver background checks: %w", err)
	}

	var currentCheck *models.BackgroundCheck
	if len(checks) > 0 {
		currentCheck = checks[0]
	}

	// Calculate overall status and compliance score
	overallStatus := s.calculateOverallStatus(checks)
	complianceScore := s.calculateComplianceScore(checks)

	status := &DriverBackgroundCheckStatus{
		DriverID:        driverID,
		CurrentCheck:    currentCheck,
		CheckHistory:    checks,
		OverallStatus:   overallStatus,
		ComplianceScore: complianceScore,
		RedFlags:        s.identifyRedFlags(checks),
		RequiredActions: s.getRequiredActions(checks),
	}

	if currentCheck != nil {
		status.LastCheckDate = currentCheck.CreatedAt
		if currentCheck.ExpiresAt != nil {
			status.NextCheckDue = currentCheck.ExpiresAt
		}
	}

	return status, nil
}

func (s *backgroundCheckService) GetBackgroundCheckByID(ctx context.Context, id primitive.ObjectID) (*models.BackgroundCheck, error) {
	return s.backgroundCheckRepo.GetByID(ctx, id)
}

func (s *backgroundCheckService) GetBackgroundChecksByDriver(ctx context.Context, driverID primitive.ObjectID, params *utils.PaginationParams) ([]*models.BackgroundCheck, int64, error) {
	return s.backgroundCheckRepo.GetByDriverID(ctx, driverID, params)
}

func (s *backgroundCheckService) GetBackgroundChecksByStatus(ctx context.Context, status models.BackgroundCheckStatus, params *utils.PaginationParams) ([]*models.BackgroundCheck, int64, error) {
	return s.backgroundCheckRepo.GetByStatus(ctx, status, params)
}

func (s *backgroundCheckService) UpdateBackgroundCheckStatus(ctx context.Context, checkID primitive.ObjectID, status models.BackgroundCheckStatus, notes string) error {
	updates := map[string]interface{}{
		"status":     status,
		"notes":      notes,
		"updated_at": time.Now(),
	}

	if status == models.BackgroundCheckStatusCompleted {
		updates["completed_at"] = time.Now()
	}

	return s.backgroundCheckRepo.Update(ctx, checkID, updates)
}

// Helper methods
func (s *backgroundCheckService) calculateEstimatedTime(checkType models.BackgroundCheckType, requiredChecks []string) time.Duration {
	baseTime := 24 * time.Hour // 1 day base

	switch checkType {
	case models.BackgroundCheckTypeBasic:
		return baseTime
	case models.BackgroundCheckTypeStandard:
		return 3 * baseTime
	case models.BackgroundCheckTypeComprehensive:
		return 7 * baseTime
	default:
		return baseTime
	}
}

func (s *backgroundCheckService) generateNextSteps(check *models.BackgroundCheck) []string {
	switch check.Status {
	case models.BackgroundCheckStatusPending:
		return []string{
			"Upload required documents",
			"Complete personal information form",
			"Provide consent for background check",
		}
	case models.BackgroundCheckStatusSubmitted:
		return []string{
			"Wait for processing",
			"Check status periodically",
		}
	case models.BackgroundCheckStatusInProgress:
		return []string{
			"Background check is being processed",
			"You will be notified when complete",
		}
	default:
		return []string{}
	}
}

func (s *backgroundCheckService) validateBackgroundCheckData(data *BackgroundCheckData, requiredChecks []string) error {
	// Basic validation
	if data.PersonalInfo == nil {
		return fmt.Errorf("personal information is required")
	}

	if data.Consent == nil || !data.Consent.ConsentGiven {
		return fmt.Errorf("consent is required for background check")
	}

	// Check specific requirements based on required checks
	for _, check := range requiredChecks {
		switch check {
		case "criminal_history":
			if data.CriminalHistory == nil {
				return fmt.Errorf("criminal history information is required")
			}
		case "motor_vehicle_record":
			if data.MotorVehicleRecord == nil {
				return fmt.Errorf("motor vehicle record is required")
			}
		case "employment_verification":
			if len(data.EmploymentHistory) == 0 {
				return fmt.Errorf("employment history is required")
			}
		}
	}

	return nil
}

func (s *backgroundCheckService) processSubmittedData(ctx context.Context, checkID primitive.ObjectID, data *BackgroundCheckData) {
	// This would typically involve:
	// 1. Submitting to third-party providers
	// 2. Running automated validation
	// 3. Flagging for manual review if needed

	s.logger.WithField("check_id", checkID.Hex()).Info("Processing submitted background check data")
}

func (s *backgroundCheckService) calculateOverallStatus(checks []*models.BackgroundCheck) models.BackgroundCheckStatus {
	if len(checks) == 0 {
		return models.BackgroundCheckStatusPending
	}

	// Return status of most recent check
	return checks[0].Status
}

func (s *backgroundCheckService) calculateComplianceScore(checks []*models.BackgroundCheck) float64 {
	if len(checks) == 0 {
		return 0.0
	}

	// Simple calculation - in reality this would be more complex
	passedChecks := 0
	for _, check := range checks {
		if check.Status == models.BackgroundCheckStatusPassed {
			passedChecks++
		}
	}

	return float64(passedChecks) / float64(len(checks)) * 100
}

func (s *backgroundCheckService) identifyRedFlags(checks []*models.BackgroundCheck) []string {
	redFlags := []string{}

	for _, check := range checks {
		if check.Status == models.BackgroundCheckStatusFailed {
			redFlags = append(redFlags, "Failed background check")
		}
		if check.Status == models.BackgroundCheckStatusExpired {
			redFlags = append(redFlags, "Expired background check")
		}
	}

	return redFlags
}

func (s *backgroundCheckService) getRequiredActions(checks []*models.BackgroundCheck) []string {
	actions := []string{}

	for _, check := range checks {
		switch check.Status {
		case models.BackgroundCheckStatusPending:
			actions = append(actions, "Complete pending background check")
		case models.BackgroundCheckStatusExpired:
			actions = append(actions, "Renew expired background check")
		case models.BackgroundCheckStatusRequiresReview:
			actions = append(actions, "Address issues from manual review")
		}
	}

	return actions
}

func (s *backgroundCheckService) allProvidersCompleted(providerData map[string]interface{}) bool {
	if len(providerData) == 0 {
		return false
	}

	for _, data := range providerData {
		if providerInfo, ok := data.(map[string]interface{}); ok {
			if status, ok := providerInfo["status"].(string); ok {
				if status != "completed" && status != "passed" {
					return false
				}
			}
		}
	}

	return true
}

// Implement remaining interface methods
func (s *backgroundCheckService) RequireAdditionalDocuments(ctx context.Context, checkID primitive.ObjectID, documents []string) error {
	// Get background check
	check, err := s.backgroundCheckRepo.GetByID(ctx, checkID)
	if err != nil {
		return fmt.Errorf("failed to get background check: %w", err)
	}

	// Update with additional required documents
	updates := map[string]interface{}{
		"additional_documents_required": documents,
		"status":                        models.BackgroundCheckStatusPendingDocuments,
		"updated_at":                    time.Now(),
	}

	if err := s.backgroundCheckRepo.Update(ctx, checkID, updates); err != nil {
		return fmt.Errorf("failed to update background check: %w", err)
	}

	// Send notification to driver
	s.notificationRepo.Create(ctx, &models.Notification{
		UserID:  check.DriverID,
		Type:    models.NotificationTypeBackgroundCheck,
		Title:   "Additional Documents Required",
		Message: "Additional documents are required for your background check.",
		Data: map[string]interface{}{
			"check_id":  checkID,
			"documents": documents,
		},
		CreatedAt: time.Now(),
	})

	s.logger.WithField("check_id", checkID.Hex()).Info("Additional documents required")
	return nil
}

func (s *backgroundCheckService) VerifyDocument(ctx context.Context, request *DocumentVerificationRequest) (*DocumentVerificationResponse, error) {
	// Validate request
	if err := utils.ValidateStruct(request); err != nil {
		return nil, fmt.Errorf("validation failed: %w", err)
	}

	// Get background check
	check, err := s.backgroundCheckRepo.GetByID(ctx, request.CheckID)
	if err != nil {
		return nil, fmt.Errorf("failed to get background check: %w", err)
	}

	startTime := time.Now()

	// Mock document verification - in reality this would use AI/ML services
	verificationID := primitive.NewObjectID()

	// Simulate verification process
	verifiedData := map[string]interface{}{
		"document_type":   request.DocumentType,
		"document_number": request.DocumentNumber,
		"verified_at":     time.Now(),
	}

	discrepancies := []string{}
	confidenceScore := 0.95
	manualReviewRequired := false

	// Simple validation rules
	if request.DocumentNumber == "" {
		discrepancies = append(discrepancies, "Missing document number")
		confidenceScore = 0.5
		manualReviewRequired = true
	}

	if request.ExpiryDate != nil && request.ExpiryDate.Before(time.Now()) {
		discrepancies = append(discrepancies, "Document has expired")
		confidenceScore = 0.3
		manualReviewRequired = true
	}

	// Update background check with verification result
	updates := map[string]interface{}{
		"updated_at": time.Now(),
	}

	if len(discrepancies) == 0 {
		// Add to completed checks
		completedChecks := append(check.CompletedChecks, request.DocumentType+"_verification")
		updates["completed_checks"] = completedChecks
	} else {
		// Add to failed checks
		failedChecks := append(check.FailedChecks, request.DocumentType+"_verification")
		updates["failed_checks"] = failedChecks
	}

	s.backgroundCheckRepo.Update(ctx, request.CheckID, updates)

	processingTime := time.Since(startTime)

	s.logger.WithField("verification_id", verificationID.Hex()).
		WithField("check_id", request.CheckID.Hex()).
		WithField("confidence_score", confidenceScore).
		Info("Document verification completed")

	return &DocumentVerificationResponse{
		VerificationID:       verificationID,
		Status:               "completed",
		VerifiedData:         verifiedData,
		DiscrepanciesFound:   discrepancies,
		ConfidenceScore:      confidenceScore,
		ManualReviewRequired: manualReviewRequired,
		ProcessingTime:       processingTime,
	}, nil
}

func (s *backgroundCheckService) UploadVerificationDocument(ctx context.Context, checkID primitive.ObjectID, documentType string, fileURL string) error {
	// Get background check
	check, err := s.backgroundCheckRepo.GetByID(ctx, checkID)
	if err != nil {
		return fmt.Errorf("failed to get background check: %w", err)
	}

	// Update documents map
	documents := check.Documents
	if documents == nil {
		documents = make(map[string]string)
	}
	documents[documentType] = fileURL

	// Update background check
	updates := map[string]interface{}{
		"documents":  documents,
		"updated_at": time.Now(),
	}

	if err := s.backgroundCheckRepo.Update(ctx, checkID, updates); err != nil {
		return fmt.Errorf("failed to update background check: %w", err)
	}

	// Create audit log
	s.auditLogRepo.Create(ctx, &models.AuditLog{
		UserID:     &check.DriverID,
		Action:     models.AuditActionUpdate,
		Resource:   "background_check",
		ResourceID: checkID.Hex(),
		NewValues: map[string]interface{}{
			"document_type": documentType,
			"file_url":      fileURL,
		},
		CreatedAt: time.Now(),
	})

	s.logger.WithField("check_id", checkID.Hex()).
		WithField("document_type", documentType).
		Info("Document uploaded for verification")

	return nil
}

func (s *backgroundCheckService) GetRequiredDocuments(ctx context.Context, userType string, region string) ([]RequiredDocument, error) {
	// Mock required documents
	return []RequiredDocument{
		{
			Type:        "drivers_license",
			Name:        "Driver's License",
			Description: "Valid driver's license",
			Required:    true,
			Formats:     []string{"jpg", "png", "pdf"},
			MaxSizeMB:   5,
		},
		{
			Type:        "social_security",
			Name:        "Social Security Card",
			Description: "Social security card or equivalent",
			Required:    true,
			Formats:     []string{"jpg", "png", "pdf"},
			MaxSizeMB:   5,
		},
	}, nil
}

func (s *backgroundCheckService) SubmitToProvider(ctx context.Context, checkID primitive.ObjectID, providerName string) (*ProviderSubmissionResponse, error) {
	// Get background check
	check, err := s.backgroundCheckRepo.GetByID(ctx, checkID)
	if err != nil {
		return nil, fmt.Errorf("failed to get background check: %w", err)
	}

	// Get provider
	provider, exists := s.providers[providerName]
	if !exists {
		return nil, fmt.Errorf("provider %s not found", providerName)
	}

	// Prepare provider request
	providerRequest := &ProviderCheckRequest{
		CheckID:      checkID.Hex(),
		PersonalInfo: check.PersonalInfo,
		CheckTypes:   check.RequiredChecks,
		CallbackURL:  fmt.Sprintf("/api/background-checks/webhook/%s", providerName),
		Metadata: map[string]interface{}{
			"driver_id":  check.DriverID.Hex(),
			"check_type": check.CheckType,
			"created_at": check.CreatedAt,
		},
	}

	// Submit to provider
	providerResponse, err := provider.SubmitCheck(ctx, providerRequest)
	if err != nil {
		return nil, fmt.Errorf("failed to submit to provider: %w", err)
	}

	// Update background check with provider information
	providerData := check.ProviderData
	if providerData == nil {
		providerData = make(map[string]interface{})
	}

	providerData[providerName] = map[string]interface{}{
		"provider_check_id": providerResponse.ProviderCheckID,
		"status":            providerResponse.Status,
		"submitted_at":      time.Now(),
		"tracking_url":      providerResponse.TrackingURL,
	}

	updates := map[string]interface{}{
		"provider_data": providerData,
		"status":        models.BackgroundCheckStatusInProgress,
		"updated_at":    time.Now(),
	}

	s.backgroundCheckRepo.Update(ctx, checkID, updates)

	s.logger.WithField("check_id", checkID.Hex()).
		WithField("provider", providerName).
		WithField("provider_check_id", providerResponse.ProviderCheckID).
		Info("Background check submitted to provider")

	return &ProviderSubmissionResponse{
		ProviderCheckID:     providerResponse.ProviderCheckID,
		Status:              providerResponse.Status,
		EstimatedCompletion: time.Now().Add(72 * time.Hour), // Default 3 days
		WebhookURL:          fmt.Sprintf("/api/background-checks/webhook/%s", providerName),
		TrackingURL:         providerResponse.TrackingURL,
	}, nil
}

func (s *backgroundCheckService) ProcessProviderWebhook(ctx context.Context, providerName string, payload []byte) error {
	// Parse webhook payload based on provider
	var webhookData map[string]interface{}
	if err := json.Unmarshal(payload, &webhookData); err != nil {
		return fmt.Errorf("failed to parse webhook payload: %w", err)
	}

	// Extract check ID from payload
	checkIDStr, ok := webhookData["check_id"].(string)
	if !ok {
		return fmt.Errorf("missing check_id in webhook payload")
	}

	checkID, err := primitive.ObjectIDFromHex(checkIDStr)
	if err != nil {
		return fmt.Errorf("invalid check_id format: %w", err)
	}

	// Get background check
	check, err := s.backgroundCheckRepo.GetByID(ctx, checkID)
	if err != nil {
		return fmt.Errorf("failed to get background check: %w", err)
	}

	// Update provider data
	providerData := check.ProviderData
	if providerData == nil {
		providerData = make(map[string]interface{})
	}

	// Extract status from webhook
	status, _ := webhookData["status"].(string)
	results, _ := webhookData["results"].(map[string]interface{})

	providerData[providerName] = map[string]interface{}{
		"status":       status,
		"results":      results,
		"last_updated": time.Now(),
		"webhook_data": webhookData,
	}

	// Determine overall status based on provider status
	var newStatus models.BackgroundCheckStatus
	switch status {
	case "completed":
		if s.allProvidersCompleted(providerData) {
			newStatus = models.BackgroundCheckStatusCompleted
		} else {
			newStatus = models.BackgroundCheckStatusInProgress
		}
	case "failed":
		newStatus = models.BackgroundCheckStatusFailed
	case "requires_review":
		newStatus = models.BackgroundCheckStatusRequiresReview
	default:
		newStatus = models.BackgroundCheckStatusInProgress
	}

	// Update background check
	updates := map[string]interface{}{
		"provider_data": providerData,
		"status":        newStatus,
		"updated_at":    time.Now(),
	}

	if newStatus == models.BackgroundCheckStatusCompleted {
		updates["completed_at"] = time.Now()
	}

	if err := s.backgroundCheckRepo.Update(ctx, checkID, updates); err != nil {
		return fmt.Errorf("failed to update background check: %w", err)
	}

	// Send notification to driver
	s.notificationRepo.Create(ctx, &models.Notification{
		UserID:  check.DriverID,
		Type:    models.NotificationTypeBackgroundCheck,
		Title:   "Background Check Update",
		Message: fmt.Sprintf("Your background check status has been updated to: %s", newStatus),
		Data: map[string]interface{}{
			"check_id": checkID,
			"status":   newStatus,
		},
		CreatedAt: time.Now(),
	})

	s.logger.WithField("check_id", checkID.Hex()).
		WithField("provider", providerName).
		WithField("status", status).
		Info("Processed provider webhook")

	return nil
}

func (s *backgroundCheckService) GetProviderStatus(ctx context.Context, checkID primitive.ObjectID, providerName string) (*ProviderStatus, error) {
	// Get provider
	provider, exists := s.providers[providerName]
	if !exists {
		return nil, fmt.Errorf("provider %s not found", providerName)
	}

	// Get background check to find provider check ID
	check, err := s.backgroundCheckRepo.GetByID(ctx, checkID)
	if err != nil {
		return nil, fmt.Errorf("failed to get background check: %w", err)
	}

	// Extract provider check ID from provider data
	providerData, exists := check.ProviderData[providerName]
	if !exists {
		return nil, fmt.Errorf("no provider data found for %s", providerName)
	}

	providerInfo, ok := providerData.(map[string]interface{})
	if !ok {
		return nil, fmt.Errorf("invalid provider data format")
	}

	providerCheckID, ok := providerInfo["provider_check_id"].(string)
	if !ok {
		return nil, fmt.Errorf("provider check ID not found")
	}

	// Get status from provider
	status, err := provider.GetStatus(ctx, providerCheckID)
	if err != nil {
		return nil, fmt.Errorf("failed to get provider status: %w", err)
	}

	s.logger.WithField("check_id", checkID.Hex()).
		WithField("provider", providerName).
		WithField("status", status.Status).
		Info("Retrieved provider status")

	return status, nil
}

func (s *backgroundCheckService) AssignForManualReview(ctx context.Context, checkID primitive.ObjectID, reviewerID primitive.ObjectID) error {
	// Get background check
	check, err := s.backgroundCheckRepo.GetByID(ctx, checkID)
	if err != nil {
		return fmt.Errorf("failed to get background check: %w", err)
	}

	// Update background check status and assign reviewer
	updates := map[string]interface{}{
		"status":                    models.BackgroundCheckStatusRequiresReview,
		"reviewer_id":               reviewerID,
		"manual_review_assigned_at": time.Now(),
		"updated_at":                time.Now(),
	}

	if err := s.backgroundCheckRepo.Update(ctx, checkID, updates); err != nil {
		return fmt.Errorf("failed to update background check: %w", err)
	}

	// Send notification to reviewer
	s.notificationRepo.Create(ctx, &models.Notification{
		UserID:  reviewerID,
		Type:    models.NotificationTypeBackgroundCheck,
		Title:   "Manual Review Assigned",
		Message: "A background check has been assigned to you for manual review.",
		Data: map[string]interface{}{
			"check_id":  checkID,
			"driver_id": check.DriverID,
		},
		CreatedAt: time.Now(),
	})

	// Send notification to driver
	s.notificationRepo.Create(ctx, &models.Notification{
		UserID:  check.DriverID,
		Type:    models.NotificationTypeBackgroundCheck,
		Title:   "Manual Review Required",
		Message: "Your background check requires manual review. This may take additional time.",
		Data: map[string]interface{}{
			"check_id": checkID,
		},
		CreatedAt: time.Now(),
	})

	s.logger.WithField("check_id", checkID.Hex()).
		WithField("reviewer_id", reviewerID.Hex()).
		Info("Background check assigned for manual review")

	return nil
}

func (s *backgroundCheckService) CompleteManualReview(ctx context.Context, checkID primitive.ObjectID, decision *ReviewDecision) error {
	// Get background check
	check, err := s.backgroundCheckRepo.GetByID(ctx, checkID)
	if err != nil {
		return fmt.Errorf("failed to get background check: %w", err)
	}

	// Determine new status based on decision
	var newStatus models.BackgroundCheckStatus
	switch decision.Decision {
	case "approved", "passed":
		newStatus = models.BackgroundCheckStatusPassed
	case "rejected", "failed":
		newStatus = models.BackgroundCheckStatusFailed
	case "requires_additional_review":
		newStatus = models.BackgroundCheckStatusRequiresReview
	default:
		return fmt.Errorf("invalid decision: %s", decision.Decision)
	}

	// Update background check
	reviewNotes := append(check.ReviewNotes, decision.Notes)
	updates := map[string]interface{}{
		"status":              newStatus,
		"reviewer_id":         decision.ReviewerID,
		"review_decision":     decision.Decision,
		"review_notes":        reviewNotes,
		"review_completed_at": time.Now(),
		"confidence_score":    decision.Confidence,
		"updated_at":          time.Now(),
	}

	if newStatus == models.BackgroundCheckStatusPassed || newStatus == models.BackgroundCheckStatusFailed {
		updates["completed_at"] = time.Now()
	}

	if decision.NextReviewDate != nil {
		updates["next_review_date"] = *decision.NextReviewDate
	}

	if err := s.backgroundCheckRepo.Update(ctx, checkID, updates); err != nil {
		return fmt.Errorf("failed to update background check: %w", err)
	}

	// Create audit log
	s.auditLogRepo.Create(ctx, &models.AuditLog{
		UserID:     &decision.ReviewerID,
		Action:     models.AuditActionUpdate,
		Resource:   "background_check",
		ResourceID: checkID.Hex(),
		NewValues: map[string]interface{}{
			"decision":    decision.Decision,
			"new_status":  newStatus,
			"reviewer_id": decision.ReviewerID,
			"confidence":  decision.Confidence,
		},
		CreatedAt: time.Now(),
	})

	// Send notification to driver
	var title, message string
	switch newStatus {
	case models.BackgroundCheckStatusPassed:
		title = "Background Check Approved"
		message = "Your background check has been approved. You can now proceed with onboarding."
	case models.BackgroundCheckStatusFailed:
		title = "Background Check Failed"
		message = "Your background check was not approved. Please contact support for more information."
	case models.BackgroundCheckStatusRequiresReview:
		title = "Additional Review Required"
		message = "Your background check requires additional review. We will update you soon."
	}

	s.notificationRepo.Create(ctx, &models.Notification{
		UserID:  check.DriverID,
		Type:    models.NotificationTypeBackgroundCheck,
		Title:   title,
		Message: message,
		Data: map[string]interface{}{
			"check_id": checkID,
			"status":   newStatus,
			"decision": decision.Decision,
		},
		CreatedAt: time.Now(),
	})

	s.logger.WithField("check_id", checkID.Hex()).
		WithField("reviewer_id", decision.ReviewerID.Hex()).
		WithField("decision", decision.Decision).
		WithField("new_status", newStatus).
		Info("Manual review completed")

	return nil
}

func (s *backgroundCheckService) GetPendingReviews(ctx context.Context, reviewerID primitive.ObjectID, params *utils.PaginationParams) ([]*models.BackgroundCheck, int64, error) {
	return s.backgroundCheckRepo.GetPendingReviews(ctx, &reviewerID, params)
}

func (s *backgroundCheckService) GenerateComplianceReport(ctx context.Context, params *ComplianceReportParams) (*ComplianceReport, error) {
	// This is a simplified implementation - in reality this would involve complex queries
	// across multiple collections and time periods

	// Get checks in the specified date range
	checks, _, err := s.backgroundCheckRepo.GetByStatus(ctx, "", &utils.PaginationParams{
		Page:     1,
		PageSize: 10000, // Large number to get all checks
	})
	if err != nil {
		return nil, fmt.Errorf("failed to get background checks: %w", err)
	}

	// Filter checks by date range
	var filteredChecks []*models.BackgroundCheck
	for _, check := range checks {
		if check.CreatedAt.After(params.StartDate) && check.CreatedAt.Before(params.EndDate) {
			filteredChecks = append(filteredChecks, check)
		}
	}

	// Calculate statistics
	totalChecks := int64(len(filteredChecks))
	var passedChecks, failedChecks, pendingChecks int64
	var totalProcessingTime time.Duration
	processedCount := 0

	byCheckType := make(map[string]*models.CheckTypeStats)
	byRegion := make(map[string]*models.RegionStats)

	for _, check := range filteredChecks {
		switch check.Status {
		case models.BackgroundCheckStatusPassed:
			passedChecks++
		case models.BackgroundCheckStatusFailed:
			failedChecks++
		case models.BackgroundCheckStatusPending, models.BackgroundCheckStatusInProgress:
			pendingChecks++
		}

		// Calculate processing time for completed checks
		if check.CompletedAt != nil {
			processingTime := check.CompletedAt.Sub(check.CreatedAt)
			totalProcessingTime += processingTime
			processedCount++
		}

		// Update check type stats
		checkTypeStr := string(check.CheckType)
		if _, exists := byCheckType[checkTypeStr]; !exists {
			byCheckType[checkTypeStr] = &models.CheckTypeStats{}
		}
		byCheckType[checkTypeStr].Total++
		if check.Status == models.BackgroundCheckStatusPassed {
			byCheckType[checkTypeStr].Passed++
		} else if check.Status == models.BackgroundCheckStatusFailed {
			byCheckType[checkTypeStr].Failed++
		} else {
			byCheckType[checkTypeStr].Pending++
		}
	}

	// Calculate rates
	var complianceRate float64
	if totalChecks > 0 {
		complianceRate = float64(passedChecks) / float64(totalChecks) * 100
	}

	var avgProcessingTime time.Duration
	if processedCount > 0 {
		avgProcessingTime = totalProcessingTime / time.Duration(processedCount)
	}

	// Generate recommendations
	recommendations := []string{}
	if complianceRate < 80 {
		recommendations = append(recommendations, "Consider reviewing background check criteria")
	}
	if avgProcessingTime > 72*time.Hour {
		recommendations = append(recommendations, "Processing time exceeds target - consider workflow optimization")
	}
	if pendingChecks > totalChecks/2 {
		recommendations = append(recommendations, "High number of pending checks - increase review capacity")
	}

	report := &ComplianceReport{
		GeneratedAt:           time.Now(),
		Period:                fmt.Sprintf("%s to %s", params.StartDate.Format("2006-01-02"), params.EndDate.Format("2006-01-02")),
		TotalChecks:           totalChecks,
		PassedChecks:          passedChecks,
		FailedChecks:          failedChecks,
		PendingChecks:         pendingChecks,
		ComplianceRate:        complianceRate,
		AverageProcessingTime: avgProcessingTime,
		ByCheckType:           byCheckType,
		ByRegion:              byRegion,
		TrendAnalysis:         &models.ComplianceTrends{}, // Simplified
		Recommendations:       recommendations,
	}

	s.logger.WithField("total_checks", totalChecks).
		WithField("compliance_rate", complianceRate).
		Info("Generated compliance report")

	return report, nil
}

func (s *backgroundCheckService) GetBackgroundCheckStats(ctx context.Context, startDate, endDate time.Time) (*BackgroundCheckStats, error) {
	// Get all checks in the date range
	checks, _, err := s.backgroundCheckRepo.GetByStatus(ctx, "", &utils.PaginationParams{
		Page:     1,
		PageSize: 10000,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to get background checks: %w", err)
	}

	// Filter by date range
	var filteredChecks []*models.BackgroundCheck
	for _, check := range checks {
		if check.CreatedAt.After(startDate) && check.CreatedAt.Before(endDate) {
			filteredChecks = append(filteredChecks, check)
		}
	}

	// Calculate statistics
	totalChecks := int64(len(filteredChecks))
	checksByStatus := make(map[string]int64)
	checksByType := make(map[string]int64)

	var totalProcessingTime time.Duration
	var processedCount int64
	var manualReviewCount int64
	var appealCount int64

	for _, check := range filteredChecks {
		// Count by status
		statusStr := string(check.Status)
		checksByStatus[statusStr]++

		// Count by type
		typeStr := string(check.CheckType)
		checksByType[typeStr]++

		// Calculate processing time for completed checks
		if check.CompletedAt != nil {
			processingTime := check.CompletedAt.Sub(check.CreatedAt)
			totalProcessingTime += processingTime
			processedCount++
		}

		// Count manual reviews
		if check.Status == models.BackgroundCheckStatusRequiresReview {
			manualReviewCount++
		}
	}

	// Calculate rates
	var complianceRate, manualReviewRate, appealRate float64
	var avgProcessingTime time.Duration

	if totalChecks > 0 {
		passedChecks := checksByStatus[string(models.BackgroundCheckStatusPassed)]
		complianceRate = float64(passedChecks) / float64(totalChecks) * 100
		manualReviewRate = float64(manualReviewCount) / float64(totalChecks) * 100
		appealRate = float64(appealCount) / float64(totalChecks) * 100
	}

	if processedCount > 0 {
		avgProcessingTime = totalProcessingTime / time.Duration(processedCount)
	}

	// Mock provider performance data
	providerPerformance := map[string]*models.ProviderStats{
		"acme_checks": {
			Provider:    "ACME Background Checks",
			TotalChecks: totalChecks / 2,
			SuccessRate: 95.5,
			AvgTime:     48 * time.Hour,
			Reliability: 98.2,
		},
	}

	stats := &BackgroundCheckStats{
		TotalChecks:           totalChecks,
		ChecksByStatus:        checksByStatus,
		ChecksByType:          checksByType,
		AverageProcessingTime: avgProcessingTime,
		ComplianceRate:        complianceRate,
		ManualReviewRate:      manualReviewRate,
		AppealRate:            appealRate,
		ProviderPerformance:   providerPerformance,
	}

	s.logger.WithField("total_checks", totalChecks).
		WithField("compliance_rate", complianceRate).
		Info("Generated background check statistics")

	return stats, nil
}

func (s *backgroundCheckService) SchedulePeriodicRechecks(ctx context.Context) error {
	// Get checks that are due for periodic rechecking
	checks, err := s.backgroundCheckRepo.GetChecksForPeriodicRecheck(ctx)
	if err != nil {
		return fmt.Errorf("failed to get checks for recheck: %w", err)
	}

	for _, check := range checks {
		// Create a new background check for renewal
		renewalRequest := &BackgroundCheckRequest{
			DriverID:       check.DriverID,
			CheckType:      models.BackgroundCheckTypeRenewal,
			RequesterID:    primitive.NewObjectID(), // System initiated
			Priority:       "normal",
			RequiredChecks: check.RequiredChecks,
			ExpiryDays:     365, // 1 year renewal
		}

		// Initiate renewal check
		_, err := s.InitiateBackgroundCheck(ctx, renewalRequest)
		if err != nil {
			s.logger.WithError(err).
				WithField("driver_id", check.DriverID.Hex()).
				Error("Failed to schedule periodic recheck")
			continue
		}

		s.logger.WithField("driver_id", check.DriverID.Hex()).
			WithField("original_check_id", check.ID.Hex()).
			Info("Scheduled periodic background check renewal")
	}

	s.logger.WithField("scheduled_count", len(checks)).
		Info("Completed scheduling periodic background check renewals")

	return nil
}

func (s *backgroundCheckService) SubmitAppeal(ctx context.Context, request *AppealRequest) (*AppealResponse, error) {
	// Validate request
	if err := utils.ValidateStruct(request); err != nil {
		return nil, fmt.Errorf("validation failed: %w", err)
	}

	// Get background check to verify it exists and can be appealed
	check, err := s.backgroundCheckRepo.GetByID(ctx, request.CheckID)
	if err != nil {
		return nil, fmt.Errorf("failed to get background check: %w", err)
	}

	// Check if appeal is allowed for this status
	if check.Status != models.BackgroundCheckStatusFailed && check.Status != models.BackgroundCheckStatusRequiresReview {
		return nil, fmt.Errorf("appeals not allowed for background checks with status: %s", check.Status)
	}

	// Create appeal record
	appeal := &models.Appeal{
		DriverID:    request.DriverID,
		CheckID:     request.CheckID,
		AppealType:  request.AppealType,
		Reason:      request.Reason,
		Evidence:    request.Evidence,
		Status:      "pending",
		CaseNumber:  fmt.Sprintf("APP-%d", time.Now().Unix()),
		ContactInfo: request.ContactInfo,
		Priority:    request.Priority,
		CreatedAt:   time.Now(),
		UpdatedAt:   time.Now(),
	}

	// Save appeal (assuming we have an appeal repository)
	// For now, we'll create an audit log entry
	s.auditLogRepo.Create(ctx, &models.AuditLog{
		UserID:     &request.DriverID,
		Action:     models.AuditActionCreate,
		Resource:   "appeal",
		ResourceID: appeal.CaseNumber,
		NewValues: map[string]interface{}{
			"check_id":    request.CheckID,
			"appeal_type": request.AppealType,
			"reason":      request.Reason,
			"case_number": appeal.CaseNumber,
		},
		CreatedAt: time.Now(),
	})

	// Send notification to driver
	s.notificationRepo.Create(ctx, &models.Notification{
		UserID:  request.DriverID,
		Type:    models.NotificationTypeBackgroundCheck,
		Title:   "Appeal Submitted",
		Message: fmt.Sprintf("Your appeal has been submitted with case number: %s", appeal.CaseNumber),
		Data: map[string]interface{}{
			"appeal_id":   appeal.ID,
			"case_number": appeal.CaseNumber,
			"check_id":    request.CheckID,
		},
		CreatedAt: time.Now(),
	})

	s.logger.WithField("case_number", appeal.CaseNumber).
		WithField("driver_id", request.DriverID.Hex()).
		WithField("check_id", request.CheckID.Hex()).
		Info("Background check appeal submitted")

	response := &AppealResponse{
		AppealID:      appeal.ID,
		Status:        appeal.Status,
		CaseNumber:    appeal.CaseNumber,
		EstimatedTime: 5 * 24 * time.Hour, // 5 business days
		ContactInfo:   "appeals@goride.com",
		NextSteps: []string{
			"Your appeal is being reviewed",
			"You will be contacted within 5 business days",
			"Please keep your case number for reference",
		},
	}

	return response, nil
}

func (s *backgroundCheckService) ProcessAppeal(ctx context.Context, appealID primitive.ObjectID, decision *AppealDecision) error {
	// Create audit log for appeal decision
	s.auditLogRepo.Create(ctx, &models.AuditLog{
		UserID:     &decision.ReviewerID,
		Action:     models.AuditActionUpdate,
		Resource:   "appeal",
		ResourceID: appealID.Hex(),
		NewValues: map[string]interface{}{
			"decision":        decision.Decision,
			"explanation":     decision.Explanation,
			"action_required": decision.ActionRequired,
			"new_status":      decision.NewStatus,
			"reviewer_id":     decision.ReviewerID,
		},
		CreatedAt: time.Now(),
	})

	// If the appeal results in a status change, update the background check
	if decision.NewStatus != nil {
		newStatus := models.BackgroundCheckStatus(*decision.NewStatus)

		// Find the background check ID from audit logs or maintain appeal-check mapping
		// For now, we'll log the decision
		s.logger.WithField("appeal_id", appealID.Hex()).
			WithField("decision", decision.Decision).
			WithField("new_status", *decision.NewStatus).
			Info("Appeal decision processed")

		// Update background check if status change is specified
		if newStatus == models.BackgroundCheckStatusPassed || newStatus == models.BackgroundCheckStatusFailed {
			// This would require getting the check ID from the appeal record
			// For now, we'll just log the action
			s.logger.WithField("appeal_id", appealID.Hex()).
				WithField("new_status", newStatus).
				Info("Background check status would be updated based on appeal decision")
		}
	}

	return nil
}

func (s *backgroundCheckService) GetAppealsByDriver(ctx context.Context, driverID primitive.ObjectID) ([]*models.Appeal, error) {
	// This would typically query an appeals repository
	// For now, we'll return an empty slice since we don't have the appeals repository implemented
	appeals := []*models.Appeal{}

	s.logger.WithField("driver_id", driverID.Hex()).
		Info("Retrieved appeals for driver")

	return appeals, nil
}
