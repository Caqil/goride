package validators

import (
	"math"
	"time"

	"go.mongodb.org/mongo-driver/bson/primitive"
)

type RideRequestRequest struct {
	RiderID           primitive.ObjectID    `json:"rider_id" validate:"required,object_id"`
	RideType          string                `json:"ride_type" validate:"required,oneof=standard premium luxury shared xl accessible"`
	PickupLocation    LocationRequest       `json:"pickup_location" validate:"required"`
	DropoffLocation   LocationRequest       `json:"dropoff_location" validate:"required"`
	Waypoints         []LocationRequest     `json:"waypoints" validate:"omitempty,max=5,dive"`
	ScheduledTime     *time.Time            `json:"scheduled_time" validate:"omitempty"`
	SpecialRequests   []string              `json:"special_requests" validate:"omitempty,max=5"`
	PaymentMethodID   primitive.ObjectID    `json:"payment_method_id" validate:"required,object_id"`
	PromoCode         string                `json:"promo_code" validate:"omitempty,max=20"`
	IsShared          bool                  `json:"is_shared"`
	MaxWaitTime       int                   `json:"max_wait_time" validate:"omitempty,min=1,max=15"`
}

type LocationRequest struct {
	Latitude    float64 `json:"latitude" validate:"required,min=-90,max=90"`
	Longitude   float64 `json:"longitude" validate:"required,min=-180,max=180"`
	Address     string  `json:"address" validate:"required,min=5,max=255"`
	City        string  `json:"city" validate:"omitempty,max=100"`
	State       string  `json:"state" validate:"omitempty,max=100"`
	Country     string  `json:"country" validate:"omitempty,max=100"`
	PostalCode  string  `json:"postal_code" validate:"omitempty,max=20"`
	PlaceID     string  `json:"place_id" validate:"omitempty,max=255"`
}

type RideUpdateRequest struct {
	Status             string             `json:"status" validate:"omitempty,oneof=accepted driver_arrived in_progress completed cancelled"`
	DriverID           *primitive.ObjectID `json:"driver_id" validate:"omitempty,object_id"`
	VehicleID          *primitive.ObjectID `json:"vehicle_id" validate:"omitempty,object_id"`
	CancellationReason string             `json:"cancellation_reason" validate:"omitempty,max=255"`
	ActualDistance     *float64           `json:"actual_distance" validate:"omitempty,distance"`
	ActualDuration     *int               `json:"actual_duration" validate:"omitempty,duration"`
}

type RideAcceptRequest struct {
	DriverID  primitive.ObjectID `json:"driver_id" validate:"required,object_id"`
	VehicleID primitive.ObjectID `json:"vehicle_id" validate:"required,object_id"`
	ETA       int                `json:"eta" validate:"required,min=1,max=120"` // minutes
}

type RideCancelRequest struct {
	Reason    string `json:"reason" validate:"required,max=255"`
	CancelledBy string `json:"cancelled_by" validate:"required,oneof=rider driver admin"`
}

type WaypointRequest struct {
	RideID    primitive.ObjectID `json:"ride_id" validate:"required,object_id"`
	Location  LocationRequest    `json:"location" validate:"required"`
	Order     int                `json:"order" validate:"required,min=1"`
}

type RideShareRequest struct {
	RideID  primitive.ObjectID   `json:"ride_id" validate:"required,object_id"`
	UserIDs []primitive.ObjectID `json:"user_ids" validate:"required,min=1,max=4,dive,object_id"`
}

func ValidateRideRequest(req *RideRequestRequest) ValidationErrors {
	errors := ValidateStruct(req)
	
	// Validate pickup and dropoff are different
	if isSameLocation(req.PickupLocation, req.DropoffLocation) {
		errors = append(errors, ValidationError{
			Field:   "dropoff_location",
			Message: "Pickup and dropoff locations must be different",
		})
	}
	
	// Validate distance
	distance := calculateDistance(req.PickupLocation, req.DropoffLocation)
	if distance < 0.1 { // Minimum 100 meters
		errors = append(errors, ValidationError{
			Field:   "distance",
			Message: "Ride distance too short (minimum 100 meters)",
		})
	}
	if distance > 500 { // Maximum 500 km
		errors = append(errors, ValidationError{
			Field:   "distance",
			Message: "Ride distance too long (maximum 500 km)",
		})
	}
	
	// Validate scheduled time
	if req.ScheduledTime != nil {
		if req.ScheduledTime.Before(time.Now()) {
			errors = append(errors, ValidationError{
				Field:   "scheduled_time",
				Message: "Scheduled time must be in the future",
			})
		}
		if req.ScheduledTime.After(time.Now().AddDate(0, 0, 7)) {
			errors = append(errors, ValidationError{
				Field:   "scheduled_time",
				Message: "Cannot schedule rides more than 7 days in advance",
			})
		}
	}
	
	// Validate special requests
	validRequests := []string{
		"child_seat",
		"pet_friendly",
		"wheelchair_accessible",
		"quiet_ride",
		"help_with_luggage",
		"air_conditioning",
		"no_music",
	}
	
	for i, request := range req.SpecialRequests {
		if !contains(validRequests, request) {
			errors = append(errors, ValidationError{
				Field:   fmt.Sprintf("special_requests[%d]", i),
				Message: "Invalid special request",
			})
		}
	}
	
	// Validate waypoints
	if len(req.Waypoints) > 0 {
		totalDistance := distance
		for i, waypoint := range req.Waypoints {
			// Validate waypoint is not too close to pickup/dropoff
			if isSameLocation(LocationRequest{
				Latitude:  waypoint.Latitude,
				Longitude: waypoint.Longitude,
			}, req.PickupLocation) {
				errors = append(errors, ValidationError{
					Field:   fmt.Sprintf("waypoints[%d]", i),
					Message: "Waypoint too close to pickup location",
				})
			}
			
			if isSameLocation(LocationRequest{
				Latitude:  waypoint.Latitude,
				Longitude: waypoint.Longitude,
			}, req.DropoffLocation) {
				errors = append(errors, ValidationError{
					Field:   fmt.Sprintf("waypoints[%d]", i),
					Message: "Waypoint too close to dropoff location",
				})
			}
		}
		
		// Calculate total distance with waypoints
		if totalDistance > 600 { // Maximum with waypoints
			errors = append(errors, ValidationError{
				Field:   "waypoints",
				Message: "Total ride distance with waypoints exceeds limit",
			})
		}
	}
	
	return errors
}

func ValidateRideUpdate(req *RideUpdateRequest) ValidationErrors {
	errors := ValidateStruct(req)
	
	// Validate cancellation reason is provided when status is cancelled
	if req.Status == "cancelled" && req.CancellationReason == "" {
		errors = append(errors, ValidationError{
			Field:   "cancellation_reason",
			Message: "Cancellation reason is required when cancelling ride",
		})
	}
	
	// Validate actual distance and duration
	if req.ActualDistance != nil && *req.ActualDistance > 1000 {
		errors = append(errors, ValidationError{
			Field:   "actual_distance",
			Message: "Actual distance seems too high",
		})
	}
	
	if req.ActualDuration != nil && *req.ActualDuration > 43200 { // 12 hours
		errors = append(errors, ValidationError{
			Field:   "actual_duration",
			Message: "Actual duration seems too high",
		})
	}
	
	return errors
}

func ValidateRideAccept(req *RideAcceptRequest) ValidationErrors {
	return ValidateStruct(req)
}

func ValidateRideCancel(req *RideCancelRequest) ValidationErrors {
	errors := ValidateStruct(req)
	
	// Validate cancellation reasons
	validReasons := []string{
		"driver_no_show",
		"rider_no_show",
		"vehicle_issue",
		"emergency",
		"traffic",
		"wrong_location",
		"rider_request",
		"driver_request",
		"payment_issue",
		"safety_concern",
		"other",
	}
	
	// Check if reason contains valid keywords
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
			Message: "Please provide a valid cancellation reason",
		})
	}
	
	return errors
}

func ValidateWaypoint(req *WaypointRequest) ValidationErrors {
	return ValidateStruct(req)
}

func ValidateRideShare(req *RideShareRequest) ValidationErrors {
	errors := ValidateStruct(req)
	
	// Validate unique user IDs
	userMap := make(map[string]bool)
	for i, userID := range req.UserIDs {
		userIDStr := userID.Hex()
		if userMap[userIDStr] {
			errors = append(errors, ValidationError{
				Field:   fmt.Sprintf("user_ids[%d]", i),
				Message: "Duplicate user ID found",
			})
		}
		userMap[userIDStr] = true
	}
	
	return errors
}

// Helper functions
func isSameLocation(loc1, loc2 LocationRequest) bool {
	// Consider locations the same if within 50 meters
	distance := calculateDistance(loc1, loc2)
	return distance < 0.05 // 50 meters
}

func calculateDistance(loc1, loc2 LocationRequest) float64 {
	// Haversine formula
	const earthRadiusKM = 6371
	
	lat1Rad := loc1.Latitude * math.Pi / 180
	lat2Rad := loc2.Latitude * math.Pi / 180
	deltaLatRad := (loc2.Latitude - loc1.Latitude) * math.Pi / 180
	deltaLngRad := (loc2.Longitude - loc1.Longitude) * math.Pi / 180
	
	a := math.Sin(deltaLatRad/2)*math.Sin(deltaLatRad/2) +
		math.Cos(lat1Rad)*math.Cos(lat2Rad)*
		math.Sin(deltaLngRad/2)*math.Sin(deltaLngRad/2)
	c := 2 * math.Atan2(math.Sqrt(a), math.Sqrt(1-a))
	
	return earthRadiusKM * c
}