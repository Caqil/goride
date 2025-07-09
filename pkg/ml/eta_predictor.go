package ml

import (
	"context"
	"math"
	"time"
)

type ETAPredictor struct {
	model     *ETAModel
	isEnabled bool
	threshold float64
	features  *FeatureExtractor
}

type ETAModel struct {
	Weights   map[string]float64 `json:"weights"`
	Intercept float64            `json:"intercept"`
	Version   string             `json:"version"`
	TrainedAt time.Time          `json:"trained_at"`
}

type ETAPredictionRequest struct {
	PickupLatitude     float64   `json:"pickup_latitude"`
	PickupLongitude    float64   `json:"pickup_longitude"`
	DropoffLatitude    float64   `json:"dropoff_latitude"`
	DropoffLongitude   float64   `json:"dropoff_longitude"`
	RequestTime        time.Time `json:"request_time"`
	DayOfWeek          int       `json:"day_of_week"`
	HourOfDay          int       `json:"hour_of_day"`
	Distance           float64   `json:"distance"`
	TrafficCondition   string    `json:"traffic_condition"`
	WeatherCondition   string    `json:"weather_condition"`
	RideType           string    `json:"ride_type"`
	HistoricalAvgSpeed float64   `json:"historical_avg_speed"`
}

type ETAPredictionResponse struct {
	PredictedETA   int     `json:"predicted_eta_minutes"`
	Confidence     float64 `json:"confidence"`
	BaselineETA    int     `json:"baseline_eta_minutes"`
	TrafficFactor  float64 `json:"traffic_factor"`
	WeatherFactor  float64 `json:"weather_factor"`
	SeasonalFactor float64 `json:"seasonal_factor"`
	ModelVersion   string  `json:"model_version"`
	UsedMLModel    bool    `json:"used_ml_model"`
}

func NewETAPredictor(modelPath string, enabled bool, threshold float64) (*ETAPredictor, error) {
	var model *ETAModel

	if enabled && modelPath != "" {
		// In a real implementation, load the model from file/database
		model = &ETAModel{
			Weights: map[string]float64{
				"distance":             2.1,
				"traffic_factor":       1.8,
				"weather_factor":       1.2,
				"hour_of_day":          0.3,
				"day_of_week":          0.2,
				"historical_avg_speed": -0.8,
				"ride_type_premium":    -0.5,
				"pickup_zone_busy":     1.5,
			},
			Intercept: 5.2,
			Version:   "1.0.0",
			TrainedAt: time.Now(),
		}
	}

	return &ETAPredictor{
		model:     model,
		isEnabled: enabled,
		threshold: threshold,
		features:  NewFeatureExtractor(),
	}, nil
}

func (e *ETAPredictor) PredictETA(ctx context.Context, request *ETAPredictionRequest) (*ETAPredictionResponse, error) {
	// Calculate baseline ETA using simple distance/speed
	baselineETA := e.calculateBaselineETA(request)

	response := &ETAPredictionResponse{
		BaselineETA:  baselineETA,
		UsedMLModel:  false,
		ModelVersion: "baseline",
		Confidence:   0.7,
	}

	// Use ML model if enabled and available
	if e.isEnabled && e.model != nil {
		mlETA, confidence, err := e.predictWithML(request)
		if err == nil && confidence >= e.threshold {
			response.PredictedETA = mlETA
			response.Confidence = confidence
			response.UsedMLModel = true
			response.ModelVersion = e.model.Version
		} else {
			response.PredictedETA = baselineETA
		}
	} else {
		response.PredictedETA = baselineETA
	}

	// Calculate individual factors
	response.TrafficFactor = e.getTrafficFactor(request.TrafficCondition)
	response.WeatherFactor = e.getWeatherFactor(request.WeatherCondition)
	response.SeasonalFactor = e.getSeasonalFactor(request.RequestTime)

	return response, nil
}

func (e *ETAPredictor) calculateBaselineETA(request *ETAPredictionRequest) int {
	// Average city speed in km/h
	avgSpeed := 30.0

	// Adjust for traffic
	trafficFactor := e.getTrafficFactor(request.TrafficCondition)
	avgSpeed = avgSpeed / trafficFactor

	// Adjust for weather
	weatherFactor := e.getWeatherFactor(request.WeatherCondition)
	avgSpeed = avgSpeed / weatherFactor

	// Adjust for time of day
	timeFactor := e.getTimeFactor(request.HourOfDay)
	avgSpeed = avgSpeed / timeFactor

	// Calculate ETA in minutes
	etaHours := request.Distance / avgSpeed
	etaMinutes := etaHours * 60

	// Add buffer time (minimum 2 minutes)
	return int(math.Max(etaMinutes+2, 2))
}

func (e *ETAPredictor) predictWithML(request *ETAPredictionRequest) (int, float64, error) {
	features := e.features.ExtractETAFeatures(request)

	// Linear regression prediction
	prediction := e.model.Intercept
	for feature, value := range features {
		if weight, exists := e.model.Weights[feature]; exists {
			prediction += weight * value
		}
	}

	// Convert to minutes and ensure minimum
	etaMinutes := int(math.Max(prediction, 1))

	// Calculate confidence based on feature completeness
	confidence := e.calculateConfidence(features)

	return etaMinutes, confidence, nil
}

func (e *ETAPredictor) getTrafficFactor(condition string) float64 {
	switch condition {
	case "heavy":
		return 2.0
	case "moderate":
		return 1.5
	case "light":
		return 1.2
	default:
		return 1.0
	}
}

func (e *ETAPredictor) getWeatherFactor(condition string) float64 {
	switch condition {
	case "rain", "snow":
		return 1.3
	case "fog":
		return 1.5
	case "storm":
		return 2.0
	default:
		return 1.0
	}
}

func (e *ETAPredictor) getSeasonalFactor(requestTime time.Time) float64 {
	month := requestTime.Month()

	// Winter months might have slower traffic
	if month == 12 || month == 1 || month == 2 {
		return 1.2
	}

	// Summer months might have more traffic
	if month >= 6 && month <= 8 {
		return 1.1
	}

	return 1.0
}

func (e *ETAPredictor) getTimeFactor(hour int) float64 {
	// Rush hour factors
	if (hour >= 7 && hour <= 9) || (hour >= 17 && hour <= 19) {
		return 1.8
	}

	// Late night
	if hour >= 22 || hour <= 5 {
		return 0.8
	}

	return 1.0
}

func (e *ETAPredictor) calculateConfidence(features map[string]float64) float64 {
	// Simple confidence calculation based on feature availability
	totalFeatures := len(e.model.Weights)
	availableFeatures := len(features)

	baseConfidence := float64(availableFeatures) / float64(totalFeatures)

	// Add some noise to make it more realistic
	return math.Min(baseConfidence*0.9+0.1, 1.0)
}
