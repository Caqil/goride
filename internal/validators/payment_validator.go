package validators

import (
	"go.mongodb.org/mongo-driver/bson/primitive"
)

type PaymentProcessRequest struct {
	RideID          primitive.ObjectID `json:"ride_id" validate:"required,object_id"`
	PaymentMethodID primitive.ObjectID `json:"payment_method_id" validate:"required,object_id"`
	Amount          float64            `json:"amount" validate:"required,fare_amount"`
	Currency        string             `json:"currency" validate:"required,currency_code"`
	TipAmount       float64            `json:"tip_amount" validate:"omitempty,min=0,max=100"`
	PromoCode       string             `json:"promo_code" validate:"omitempty,max=20"`
}

type RefundRequest struct {
	PaymentID    primitive.ObjectID `json:"payment_id" validate:"required,object_id"`
	Amount       float64            `json:"amount" validate:"required,min=0.01"`
	Reason       string             `json:"reason" validate:"required,max=255"`
	RefundType   string             `json:"refund_type" validate:"required,oneof=full partial"`
}

type PaymentMethodCreateRequest struct {
	UserID         primitive.ObjectID     `json:"user_id" validate:"required,object_id"`
	Type           string                 `json:"type" validate:"required,oneof=credit_card debit_card paypal apple_pay google_pay wallet"`
	Token          string                 `json:"token" validate:"required"`
	LastFourDigits string                 `json:"last_four_digits" validate:"omitempty,len=4,numeric"`
	ExpiryMonth    int                    `json:"expiry_month" validate:"omitempty,min=1,max=12"`
	ExpiryYear     int                    `json:"expiry_year" validate:"omitempty,min=2024"`
	BillingAddress *BillingAddressRequest `json:"billing_address" validate:"omitempty"`
	IsDefault      bool                   `json:"is_default"`
}

type SplitPaymentRequest struct {
	RideID      primitive.ObjectID          `json:"ride_id" validate:"required,object_id"`
	TotalAmount float64                     `json:"total_amount" validate:"required,fare_amount"`
	Currency    string                      `json:"currency" validate:"required,currency_code"`
	Splits      []PaymentSplitRequest       `json:"splits" validate:"required,min=2,max=4,dive"`
}

type PaymentSplitRequest struct {
	UserID          primitive.ObjectID `json:"user_id" validate:"required,object_id"`
	PaymentMethodID primitive.ObjectID `json:"payment_method_id" validate:"required,object_id"`
	Amount          float64            `json:"amount" validate:"required,min=0.01"`
	Percentage      float64            `json:"percentage" validate:"omitempty,min=0,max=100"`
}

type WalletTopUpRequest struct {
	UserID          primitive.ObjectID `json:"user_id" validate:"required,object_id"`
	Amount          float64            `json:"amount" validate:"required,min=5,max=1000"`
	Currency        string             `json:"currency" validate:"required,currency_code"`
	PaymentMethodID primitive.ObjectID `json:"payment_method_id" validate:"required,object_id"`
}

type WalletWithdrawRequest struct {
	UserID      primitive.ObjectID `json:"user_id" validate:"required,object_id"`
	Amount      float64            `json:"amount" validate:"required,min=10"`
	Currency    string             `json:"currency" validate:"required,currency_code"`
	BankAccount *BankAccountRequest `json:"bank_account" validate:"required"`
}

type FareCalculationRequest struct {
	RideType        string            `json:"ride_type" validate:"required,oneof=standard premium luxury shared xl accessible"`
	Distance        float64           `json:"distance" validate:"required,distance"`
	Duration        int               `json:"duration" validate:"required,duration"`
	City            string            `json:"city" validate:"required,min=2,max=100"`
	SurgeMultiplier float64           `json:"surge_multiplier" validate:"omitempty,min=1,max=5"`
	PromoCode       string            `json:"promo_code" validate:"omitempty,max=20"`
	Waypoints       int               `json:"waypoints" validate:"omitempty,min=0,max=5"`
}

func ValidatePaymentProcess(req *PaymentProcessRequest) ValidationErrors {
	errors := ValidateStruct(req)
	
	// Validate tip amount is reasonable (max 50% of ride amount)
	if req.TipAmount > req.Amount*0.5 {
		errors = append(errors, ValidationError{
			Field:   "tip_amount",
			Message: "Tip amount cannot exceed 50% of ride amount",
		})
	}
	
	// Validate minimum payment amount
	if req.Amount < 1.0 {
		errors = append(errors, ValidationError{
			Field:   "amount",
			Message: "Payment amount must be at least $1.00",
		})
	}
	
	return errors
}

func ValidateRefund(req *RefundRequest) ValidationErrors {
	errors := ValidateStruct(req)
	
	// Validate refund reasons
	validReasons := []string{
		"ride_cancelled",
		"driver_no_show",
		"vehicle_issue",
		"poor_service",
		"billing_error",
		"duplicate_charge",
		"customer_request",
		"technical_issue",
		"safety_issue",
	}
	
	reasonFound := false
	for _, validReason := range validReasons {
		if strings.Contains(strings.ToLower(req.Reason), validReason) {
			reasonFound = true
			break
		}
	}
	
	if !reasonFound {
		errors = append(errors, ValidationError{
			Field:   "reason",
			Message: "Please provide a valid refund reason",
		})
	}
	
	return errors
}

func ValidatePaymentMethodCreate(req *PaymentMethodCreateRequest) ValidationErrors {
	errors := ValidateStruct(req)
	
	// Validate card-specific fields
	if req.Type == "credit_card" || req.Type == "debit_card" {
		if req.ExpiryMonth == 0 || req.ExpiryYear == 0 {
			errors = append(errors, ValidationError{
				Field:   "expiry",
				Message: "Expiry month and year are required for cards",
			})
		}
		
		if req.LastFourDigits == "" {
			errors = append(errors, ValidationError{
				Field:   "last_four_digits",
				Message: "Last four digits are required for cards",
			})
		}
		
		// Validate card not expired
		currentYear := time.Now().Year()
		currentMonth := int(time.Now().Month())
		
		if req.ExpiryYear < currentYear || 
		   (req.ExpiryYear == currentYear && req.ExpiryMonth < currentMonth) {
			errors = append(errors, ValidationError{
				Field:   "expiry",
				Message: "Card is expired",
			})
		}
	}
	
	return errors
}

func ValidateSplitPayment(req *SplitPaymentRequest) ValidationErrors {
	errors := ValidateStruct(req)
	
	// Validate split amounts add up to total
	var totalSplitAmount float64
	var totalPercentage float64
	userMap := make(map[string]bool)
	
	for i, split := range req.Splits {
		totalSplitAmount += split.Amount
		totalPercentage += split.Percentage
		
		// Check for duplicate users
		userIDStr := split.UserID.Hex()
		if userMap[userIDStr] {
			errors = append(errors, ValidationError{
				Field:   fmt.Sprintf("splits[%d].user_id", i),
				Message: "Duplicate user in split payment",
			})
		}
		userMap[userIDStr] = true
		
		// Validate either amount or percentage is provided
		if split.Amount == 0 && split.Percentage == 0 {
			errors = append(errors, ValidationError{
				Field:   fmt.Sprintf("splits[%d]", i),
				Message: "Either amount or percentage must be specified",
			})
		}
		
		if split.Amount > 0 && split.Percentage > 0 {
			errors = append(errors, ValidationError{
				Field:   fmt.Sprintf("splits[%d]", i),
				Message: "Specify either amount or percentage, not both",
			})
		}
	}
	
	// Check if amounts add up (allow small rounding differences)
	if math.Abs(totalSplitAmount-req.TotalAmount) > 0.01 && totalSplitAmount > 0 {
		errors = append(errors, ValidationError{
			Field:   "splits",
			Message: "Split amounts do not add up to total amount",
		})
	}
	
	// Check if percentages add up to 100%
	if math.Abs(totalPercentage-100.0) > 0.01 && totalPercentage > 0 {
		errors = append(errors, ValidationError{
			Field:   "splits",
			Message: "Split percentages do not add up to 100%",
		})
	}
	
	return errors
}

func ValidateWalletTopUp(req *WalletTopUpRequest) ValidationErrors {
	errors := ValidateStruct(req)
	
	// Additional business logic for wallet top-up limits
	if req.Amount > 500 {
		errors = append(errors, ValidationError{
			Field:   "amount",
			Message: "Single top-up amount cannot exceed $500",
		})
	}
	
	return errors
}

func ValidateWalletWithdraw(req *WalletWithdrawRequest) ValidationErrors {
	errors := ValidateStruct(req)
	
	// Additional business logic for withdrawal limits
	if req.Amount > 1000 {
		errors = append(errors, ValidationError{
			Field:   "amount",
			Message: "Single withdrawal amount cannot exceed $1000",
		})
	}
	
	return errors
}

func ValidateFareCalculation(req *FareCalculationRequest) ValidationErrors {
	errors := ValidateStruct(req)
	
	// Validate surge multiplier
	if req.SurgeMultiplier != 0 && (req.SurgeMultiplier < 1.0 || req.SurgeMultiplier > 5.0) {
		errors = append(errors, ValidationError{
			Field:   "surge_multiplier",
			Message: "Surge multiplier must be between 1.0 and 5.0",
		})
	}
	
	// Validate reasonable duration for distance
	avgSpeed := req.Distance / (float64(req.Duration) / 3600) // km/h
	if avgSpeed > 200 { // Unrealistic speed
		errors = append(errors, ValidationError{
			Field:   "duration",
			Message: "Duration seems too short for the distance",
		})
	}
	
	if avgSpeed < 1 && req.Duration > 0 { // Too slow
		errors = append(errors, ValidationError{
			Field:   "duration",
			Message: "Duration seems too long for the distance",
		})
	}
	
	return errors
}