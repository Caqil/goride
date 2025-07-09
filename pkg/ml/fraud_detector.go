package ml

import (
	"context"
	"math"
	"time"
)

type FraudDetector struct {
	model     *FraudModel
	isEnabled bool
	threshold float64
}

type FraudModel struct {
	Weights       map[string]float64 `json:"weights"`
	Intercept     float64            `json:"intercept"`
	Version       string             `json:"version"`
	TrainedAt     time.Time          `json:"trained_at"`
	FraudPatterns []FraudPattern     `json:"fraud_patterns"`
}

type FraudPattern struct {
	Name     string      `json:"name"`
	Rules    []FraudRule `json:"rules"`
	Severity string      `json:"severity"`
	Action   string      `json:"action"`
}

type FraudRule struct {
	Field    string  `json:"field"`
	Operator string  `json:"operator"`
	Value    float64 `json:"value"`
	Weight   float64 `json:"weight"`
}

type FraudDetectionRequest struct {
	UserID               string    `json:"user_id"`
	PaymentMethodID      string    `json:"payment_method_id"`
	Amount               float64   `json:"amount"`
	Currency             string    `json:"currency"`
	PickupLatitude       float64   `json:"pickup_latitude"`
	PickupLongitude      float64   `json:"pickup_longitude"`
	DropoffLatitude      float64   `json:"dropoff_latitude"`
	DropoffLongitude     float64   `json:"dropoff_longitude"`
	RequestTime          time.Time `json:"request_time"`
	UserRegistrationAge  int       `json:"user_registration_age_days"`
	UserTotalRides       int       `json:"user_total_rides"`
	UserAverageRating    float64   `json:"user_average_rating"`
	DeviceID             string    `json:"device_id"`
	IPAddress            string    `json:"ip_address"`
	UserAgent            string    `json:"user_agent"`
	IsNewDevice          bool      `json:"is_new_device"`
	IsVPN                bool      `json:"is_vpn"`
	TimeFromLastRide     int       `json:"time_from_last_ride_minutes"`
	DistanceFromLastRide float64   `json:"distance_from_last_ride_km"`
}

type FraudDetectionResponse struct {
	RiskScore         float64                `json:"risk_score"`
	RiskLevel         string                 `json:"risk_level"`
	IsHighRisk        bool                   `json:"is_high_risk"`
	RecommendedAction string                 `json:"recommended_action"`
	TriggeredRules    []string               `json:"triggered_rules"`
	Confidence        float64                `json:"confidence"`
	ModelVersion      string                 `json:"model_version"`
	UsedMLModel       bool                   `json:"used_ml_model"`
	Details           map[string]interface{} `json:"details"`
}

func NewFraudDetector(modelPath string, enabled bool, threshold float64) (*FraudDetector, error) {
	var model *FraudModel

	if enabled && modelPath != "" {
		// In a real implementation, load from file/database
		model = &FraudModel{
			Weights: map[string]float64{
				"amount_zscore":     2.1,
				"new_user_factor":   1.8,
				"device_risk":       1.5,
				"location_velocity": 3.0,
				"time_anomaly":      1.2,
				"rating_factor":     -0.8,
				"payment_risk":      2.5,
				"ip_risk":           1.7,
			},
			Intercept: -2.5,
			Version:   "1.0.0",
			TrainedAt: time.Now(),
			FraudPatterns: []FraudPattern{
				{
					Name:     "velocity_fraud",
					Severity: "high",
					Action:   "block",
					Rules: []FraudRule{
						{Field: "location_velocity", Operator: ">", Value: 1000, Weight: 3.0},
					},
				},
				{
					Name:     "new_user_high_amount",
					Severity: "medium",
					Action:   "review",
					Rules: []FraudRule{
						{Field: "user_registration_age", Operator: "<", Value: 1, Weight: 2.0},
						{Field: "amount", Operator: ">", Value: 100, Weight: 1.5},
					},
				},
			},
		}
	}

	return &FraudDetector{
		model:     model,
		isEnabled: enabled,
		threshold: threshold,
	}, nil
}

func (f *FraudDetector) DetectFraud(ctx context.Context, request *FraudDetectionRequest) (*FraudDetectionResponse, error) {
	response := &FraudDetectionResponse{
		UsedMLModel:  false,
		ModelVersion: "baseline",
		Confidence:   0.7,
		Details:      make(map[string]interface{}),
	}

	// Rule-based detection
	ruleScore, triggeredRules := f.checkFraudRules(request)
	response.TriggeredRules = triggeredRules

	// Use ML model if enabled
	if f.isEnabled && f.model != nil {
		mlScore, confidence, err := f.predictWithML(request)
		if err == nil && confidence >= f.threshold {
			response.RiskScore = mlScore
			response.Confidence = confidence
			response.UsedMLModel = true
			response.ModelVersion = f.model.Version
		} else {
			response.RiskScore = ruleScore
		}
	} else {
		response.RiskScore = ruleScore
	}

	// Determine risk level and action
	response.RiskLevel = f.categorizeRisk(response.RiskScore)
	response.IsHighRisk = response.RiskScore >= 0.7
	response.RecommendedAction = f.getRecommendedAction(response.RiskScore, triggeredRules)

	// Add details
	response.Details["amount_factor"] = f.calculateAmountFactor(request)
	response.Details["user_factor"] = f.calculateUserFactor(request)
	response.Details["location_factor"] = f.calculateLocationFactor(request)
	response.Details["device_factor"] = f.calculateDeviceFactor(request)

	return response, nil
}

func (f *FraudDetector) checkFraudRules(request *FraudDetectionRequest) (float64, []string) {
	var triggeredRules []string
	totalScore := 0.0

	if f.model == nil {
		return f.calculateBaselineRisk(request), triggeredRules
	}

	// Check each fraud pattern
	for _, pattern := range f.model.FraudPatterns {
		patternScore := 0.0
		allRulesMet := true

		for _, rule := range pattern.Rules {
			fieldValue := f.getFieldValue(request, rule.Field)

			ruleTriggered := false
			switch rule.Operator {
			case ">":
				ruleTriggered = fieldValue > rule.Value
			case "<":
				ruleTriggered = fieldValue < rule.Value
			case "=":
				ruleTriggered = fieldValue == rule.Value
			case ">=":
				ruleTriggered = fieldValue >= rule.Value
			case "<=":
				ruleTriggered = fieldValue <= rule.Value
			}

			if ruleTriggered {
				patternScore += rule.Weight
			} else {
				allRulesMet = false
			}
		}

		if allRulesMet && patternScore > 0 {
			triggeredRules = append(triggeredRules, pattern.Name)
			totalScore += patternScore
		}
	}

	// Normalize score to 0-1 range
	normalizedScore := math.Min(totalScore/10.0, 1.0)

	return normalizedScore, triggeredRules
}

func (f *FraudDetector) predictWithML(request *FraudDetectionRequest) (float64, float64, error) {
	features := f.extractFraudFeatures(request)

	// Logistic regression prediction
	logit := f.model.Intercept
	for feature, value := range features {
		if weight, exists := f.model.Weights[feature]; exists {
			logit += weight * value
		}
	}

	// Convert to probability using sigmoid function
	probability := 1.0 / (1.0 + math.Exp(-logit))

	confidence := f.calculateFraudConfidence(features)

	return probability, confidence, nil
}

func (f *FraudDetector) extractFraudFeatures(request *FraudDetectionRequest) map[string]float64 {
	features := make(map[string]float64)

	// Amount features
	features["amount_zscore"] = f.calculateAmountZScore(request.Amount)

	// User features
	features["new_user_factor"] = f.calculateNewUserFactor(request)
	features["rating_factor"] = f.calculateRatingFactor(request.UserAverageRating)

	// Device and IP features
	features["device_risk"] = f.calculateDeviceRisk(request)
	features["ip_risk"] = f.calculateIPRisk(request)

	// Location features
	features["location_velocity"] = f.calculateLocationVelocity(request)

	// Time features
	features["time_anomaly"] = f.calculateTimeAnomaly(request)

	// Payment features
	features["payment_risk"] = f.calculatePaymentRisk(request)

	return features
}

func (f *FraudDetector) calculateBaselineRisk(request *FraudDetectionRequest) float64 {
	risk := 0.0

	// High amount risk
	if request.Amount > 200 {
		risk += 0.3
	}

	// New user risk
	if request.UserRegistrationAge < 7 {
		risk += 0.2
	}

	// Low rating risk
	if request.UserAverageRating < 3.0 && request.UserTotalRides > 5 {
		risk += 0.25
	}

	// New device risk
	if request.IsNewDevice {
		risk += 0.15
	}

	// VPN risk
	if request.IsVPN {
		risk += 0.1
	}

	// Velocity risk
	if request.TimeFromLastRide < 30 && request.DistanceFromLastRide > 50 {
		risk += 0.4
	}

	return math.Min(risk, 1.0)
}

func (f *FraudDetector) getFieldValue(request *FraudDetectionRequest, field string) float64 {
	switch field {
	case "amount":
		return request.Amount
	case "user_registration_age":
		return float64(request.UserRegistrationAge)
	case "user_total_rides":
		return float64(request.UserTotalRides)
	case "user_average_rating":
		return request.UserAverageRating
	case "time_from_last_ride":
		return float64(request.TimeFromLastRide)
	case "distance_from_last_ride":
		return request.DistanceFromLastRide
	case "location_velocity":
		return f.calculateLocationVelocity(request)
	default:
		return 0.0
	}
}

func (f *FraudDetector) calculateAmountZScore(amount float64) float64 {
	// Assume mean=50, std=30 for ride amounts
	mean := 50.0
	std := 30.0
	return (amount - mean) / std
}

func (f *FraudDetector) calculateNewUserFactor(request *FraudDetectionRequest) float64 {
	if request.UserRegistrationAge < 1 {
		return 1.0
	} else if request.UserRegistrationAge < 7 {
		return 0.7
	} else if request.UserRegistrationAge < 30 {
		return 0.3
	}
	return 0.0
}

func (f *FraudDetector) calculateRatingFactor(rating float64) float64 {
	if rating < 3.0 {
		return 1.0
	} else if rating < 4.0 {
		return 0.5
	}
	return 0.0
}

func (f *FraudDetector) calculateDeviceRisk(request *FraudDetectionRequest) float64 {
	risk := 0.0
	if request.IsNewDevice {
		risk += 0.5
	}
	// Add more device-based risk factors here
	return risk
}

func (f *FraudDetector) calculateIPRisk(request *FraudDetectionRequest) float64 {
	risk := 0.0
	if request.IsVPN {
		risk += 0.7
	}
	// Add more IP-based risk factors here
	return risk
}

func (f *FraudDetector) calculateLocationVelocity(request *FraudDetectionRequest) float64 {
	if request.TimeFromLastRide == 0 {
		return 0.0
	}

	// Calculate velocity in km/h
	timeHours := float64(request.TimeFromLastRide) / 60.0
	velocity := request.DistanceFromLastRide / timeHours

	return velocity
}

func (f *FraudDetector) calculateTimeAnomaly(request *FraudDetectionRequest) float64 {
	hour := request.RequestTime.Hour()

	// Late night rides might be riskier
	if hour >= 2 && hour <= 5 {
		return 0.3
	}

	return 0.0
}

func (f *FraudDetector) calculatePaymentRisk(request *FraudDetectionRequest) float64 {
	// This would be based on payment method history, etc.
	return 0.0
}

func (f *FraudDetector) calculateAmountFactor(request *FraudDetectionRequest) float64 {
	// Normalize amount risk
	if request.Amount > 500 {
		return 1.0
	} else if request.Amount > 200 {
		return 0.7
	} else if request.Amount > 100 {
		return 0.4
	}
	return 0.1
}

func (f *FraudDetector) calculateUserFactor(request *FraudDetectionRequest) float64 {
	factor := 0.0

	// New user factor
	if request.UserRegistrationAge < 7 {
		factor += 0.5
	}

	// Low rating factor
	if request.UserAverageRating < 3.0 && request.UserTotalRides > 5 {
		factor += 0.3
	}

	// Few rides factor
	if request.UserTotalRides < 5 {
		factor += 0.2
	}

	return math.Min(factor, 1.0)
}

func (f *FraudDetector) calculateLocationFactor(request *FraudDetectionRequest) float64 {
	velocity := f.calculateLocationVelocity(request)

	if velocity > 1000 { // Impossible velocity
		return 1.0
	} else if velocity > 500 {
		return 0.8
	} else if velocity > 200 {
		return 0.5
	}

	return 0.0
}

func (f *FraudDetector) calculateDeviceFactor(request *FraudDetectionRequest) float64 {
	factor := 0.0

	if request.IsNewDevice {
		factor += 0.4
	}

	if request.IsVPN {
		factor += 0.6
	}

	return math.Min(factor, 1.0)
}

func (f *FraudDetector) categorizeRisk(score float64) string {
	if score >= 0.8 {
		return "very_high"
	} else if score >= 0.6 {
		return "high"
	} else if score >= 0.4 {
		return "medium"
	} else if score >= 0.2 {
		return "low"
	}
	return "very_low"
}

func (f *FraudDetector) getRecommendedAction(score float64, triggeredRules []string) string {
	// Check for high-severity rules
	for _, rule := range triggeredRules {
		if rule == "velocity_fraud" {
			return "block"
		}
	}

	if score >= 0.8 {
		return "block"
	} else if score >= 0.6 {
		return "manual_review"
	} else if score >= 0.4 {
		return "additional_verification"
	} else if score >= 0.2 {
		return "monitor"
	}

	return "allow"
}

func (f *FraudDetector) calculateFraudConfidence(features map[string]float64) float64 {
	// Simple confidence calculation
	requiredFeatures := []string{"amount_zscore", "new_user_factor", "device_risk", "location_velocity"}
	availableCount := 0

	for _, feature := range requiredFeatures {
		if _, exists := features[feature]; exists {
			availableCount++
		}
	}

	return float64(availableCount) / float64(len(requiredFeatures))
}

// Feature extractor helper
type FeatureExtractor struct{}

func NewFeatureExtractor() *FeatureExtractor {
	return &FeatureExtractor{}
}

func (fe *FeatureExtractor) ExtractETAFeatures(request *ETAPredictionRequest) map[string]float64 {
	features := make(map[string]float64)

	features["distance"] = request.Distance
	features["hour_of_day"] = float64(request.HourOfDay)
	features["day_of_week"] = float64(request.DayOfWeek)
	features["historical_avg_speed"] = request.HistoricalAvgSpeed

	// Traffic condition encoding
	switch request.TrafficCondition {
	case "heavy":
		features["traffic_factor"] = 2.0
	case "moderate":
		features["traffic_factor"] = 1.5
	case "light":
		features["traffic_factor"] = 1.2
	default:
		features["traffic_factor"] = 1.0
	}

	// Weather condition encoding
	switch request.WeatherCondition {
	case "rain", "snow":
		features["weather_factor"] = 1.3
	case "storm":
		features["weather_factor"] = 2.0
	default:
		features["weather_factor"] = 1.0
	}

	// Ride type encoding
	if request.RideType == "premium" {
		features["ride_type_premium"] = 1.0
	}

	// Rush hour indicator
	if (request.HourOfDay >= 7 && request.HourOfDay <= 9) ||
		(request.HourOfDay >= 17 && request.HourOfDay <= 19) {
		features["rush_hour"] = 1.0
	}

	return features
}
