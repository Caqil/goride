package ml

import (
	"context"
	"math"
	"time"
)

type DemandForecaster struct {
	model     *DemandModel
	isEnabled bool
	threshold float64
}

type DemandModel struct {
	TimeSeriesWeights map[string]float64 `json:"time_series_weights"`
	SeasonalWeights   map[string]float64 `json:"seasonal_weights"`
	EventWeights      map[string]float64 `json:"event_weights"`
	Intercept         float64            `json:"intercept"`
	Version           string             `json:"version"`
	TrainedAt         time.Time          `json:"trained_at"`
}

type DemandForecastRequest struct {
	Area             string    `json:"area"`
	Latitude         float64   `json:"latitude"`
	Longitude        float64   `json:"longitude"`
	ForecastTime     time.Time `json:"forecast_time"`
	HistoricalDemand []float64 `json:"historical_demand"`
	WeatherCondition string    `json:"weather_condition"`
	EventsNearby     []string  `json:"events_nearby"`
	IsHoliday        bool      `json:"is_holiday"`
	IsWeekend        bool      `json:"is_weekend"`
}

type DemandForecastResponse struct {
	PredictedDemand   float64 `json:"predicted_demand"`
	DemandLevel       string  `json:"demand_level"`
	SuggestedSurge    float64 `json:"suggested_surge"`
	Confidence        float64 `json:"confidence"`
	HistoricalAverage float64 `json:"historical_average"`
	ModelVersion      string  `json:"model_version"`
	UsedMLModel       bool    `json:"used_ml_model"`
}

func NewDemandForecaster(modelPath string, enabled bool, threshold float64) (*DemandForecaster, error) {
	var model *DemandModel

	if enabled && modelPath != "" {
		// In a real implementation, load from file/database
		model = &DemandModel{
			TimeSeriesWeights: map[string]float64{
				"hour_of_day":    0.8,
				"day_of_week":    0.6,
				"month":          0.4,
				"historical_avg": 1.2,
			},
			SeasonalWeights: map[string]float64{
				"is_weekend":     1.5,
				"is_holiday":     2.0,
				"weather_factor": 0.7,
			},
			EventWeights: map[string]float64{
				"concert":    3.0,
				"sports":     2.5,
				"conference": 1.8,
				"festival":   2.2,
			},
			Intercept: 10.0,
			Version:   "1.0.0",
			TrainedAt: time.Now(),
		}
	}

	return &DemandForecaster{
		model:     model,
		isEnabled: enabled,
		threshold: threshold,
	}, nil
}

func (d *DemandForecaster) ForecastDemand(ctx context.Context, request *DemandForecastRequest) (*DemandForecastResponse, error) {
	// Calculate historical average
	historicalAvg := d.calculateHistoricalAverage(request.HistoricalDemand)

	response := &DemandForecastResponse{
		HistoricalAverage: historicalAvg,
		UsedMLModel:       false,
		ModelVersion:      "baseline",
		Confidence:        0.6,
	}

	// Use ML model if enabled
	if d.isEnabled && d.model != nil {
		mlDemand, confidence, err := d.predictWithML(request)
		if err == nil && confidence >= d.threshold {
			response.PredictedDemand = mlDemand
			response.Confidence = confidence
			response.UsedMLModel = true
			response.ModelVersion = d.model.Version
		} else {
			response.PredictedDemand = d.calculateBaselineDemand(request, historicalAvg)
		}
	} else {
		response.PredictedDemand = d.calculateBaselineDemand(request, historicalAvg)
	}

	// Calculate demand level and suggested surge
	response.DemandLevel = d.categorizeDemand(response.PredictedDemand, historicalAvg)
	response.SuggestedSurge = d.calculateSuggestedSurge(response.PredictedDemand, historicalAvg)

	return response, nil
}

// calculateSuggestedSurge computes a surge multiplier based on predicted and historical demand.
func (d *DemandForecaster) calculateSuggestedSurge(predicted, historicalAvg float64) float64 {
	if historicalAvg <= 0 {
		return 1.0
	}
	ratio := predicted / historicalAvg
	switch {
	case ratio >= 2.0:
		return 2.0
	case ratio >= 1.5:
		return 1.5
	case ratio >= 1.2:
		return 1.2
	default:
		return 1.0
	}
}

// categorizeDemand classifies the demand level based on predicted and historical average demand.
func (d *DemandForecaster) categorizeDemand(predicted, historicalAvg float64) string {
	ratio := predicted / (historicalAvg + 1e-6) // avoid division by zero
	switch {
	case ratio >= 1.5:
		return "high"
	case ratio >= 1.1:
		return "medium"
	default:
		return "low"
	}
}

func (d *DemandForecaster) calculateHistoricalAverage(historical []float64) float64 {
	if len(historical) == 0 {
		return 20.0 // Default demand
	}

	sum := 0.0
	for _, value := range historical {
		sum += value
	}

	return sum / float64(len(historical))
}

func (d *DemandForecaster) calculateBaselineDemand(request *DemandForecastRequest, historicalAvg float64) float64 {
	demand := historicalAvg

	// Time-based adjustments
	hour := request.ForecastTime.Hour()
	if (hour >= 7 && hour <= 9) || (hour >= 17 && hour <= 19) {
		demand *= 1.8 // Rush hours
	} else if hour >= 22 || hour <= 5 {
		demand *= 0.4 // Late night
	}

	// Weekend adjustment
	if request.IsWeekend {
		demand *= 1.3
	}

	// Holiday adjustment
	if request.IsHoliday {
		demand *= 1.6
	}

	// Weather adjustment
	weatherFactor := d.getWeatherDemandFactor(request.WeatherCondition)
	demand *= weatherFactor

	// Event adjustment
	eventFactor := d.getEventFactor(request.EventsNearby)
	demand *= eventFactor

	return math.Max(demand, 1.0)
}

// getWeatherDemandFactor returns a multiplier based on the weather condition.
func (d *DemandForecaster) getWeatherDemandFactor(weather string) float64 {
	switch weather {
	case "rain":
		return 1.5
	case "storm":
		return 1.8
	case "snow":
		return 2.0
	case "fog":
		return 1.2
	case "clear":
		return 1.0
	case "hot":
		return 1.1
	case "cold":
		return 1.2
	default:
		return 1.0
	}
}

// getEventFactor returns a multiplier based on the presence of events nearby.
func (d *DemandForecaster) getEventFactor(events []string) float64 {
	if d.model == nil || len(events) == 0 {
		return 1.0
	}
	factor := 1.0
	for _, event := range events {
		if weight, exists := d.model.EventWeights[event]; exists {
			factor += weight
		}
	}
	return math.Max(factor, 1.0)
}

func (d *DemandForecaster) predictWithML(request *DemandForecastRequest) (float64, float64, error) {
	features := d.extractFeatures(request)

	prediction := d.model.Intercept

	// Time series features
	for feature, value := range features {
		if weight, exists := d.model.TimeSeriesWeights[feature]; exists {
			prediction += weight * value
		}
		if weight, exists := d.model.SeasonalWeights[feature]; exists {
			prediction += weight * value
		}
	}

	// Event features
	for _, event := range request.EventsNearby {
		if weight, exists := d.model.EventWeights[event]; exists {
			prediction += weight
		}
	}

	confidence := d.calculateDemandConfidence(features)

	return math.Max(prediction, 1.0), confidence, nil
}

func (d *DemandForecaster) extractFeatures(request *DemandForecastRequest) map[string]float64 {
	features := make(map[string]float64)

	features["hour_of_day"] = float64(request.ForecastTime.Hour())
	features["day_of_week"] = float64(request.ForecastTime.Weekday())
	features["month"] = float64(request.ForecastTime.Month())

	if len(request.HistoricalDemand) > 0 {
		features["historical_avg"] = d.calculateHistoricalAverage(request.HistoricalDemand)
	}

	if request.IsWeekend {
		features["is_weekend"] = 1.0
	}

	if request.IsHoliday {
		features["is_holiday"] = 1.0
	}

	features["weather_factor"] = d.getWeatherDemandFactor(request.WeatherCondition)

	return features
}

// calculateDemandConfidence estimates the confidence of the demand prediction based on feature completeness.
func (d *DemandForecaster) calculateDemandConfidence(features map[string]float64) float64 {
	// Simple heuristic: confidence increases with the number of non-zero features
	total := 0
	nonZero := 0
	for _, v := range features {
		total++
		if v != 0 {
			nonZero++
		}
	}
	if total == 0 {
		return 0.5
	}
	confidence := 0.5 + 0.5*float64(nonZero)/float64(total)
	if confidence > 1.0 {
		return 1.0
	}
	return confidence
}
