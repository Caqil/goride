package validators

import (
	"go.mongodb.org/mongo-driver/bson/primitive"
)

type RatingCreateRequest struct {
	RideID      primitive.ObjectID `json:"ride_id" validate:"required,object_id"`
	RatedID     primitive.ObjectID `json:"rated_id" validate:"required,object_id"`
	RaterType   string             `json:"rater_type" validate:"required,oneof=rider driver"`
	Rating      float64            `json:"rating" validate:"required,rating_value"`
	Comment     string             `json:"comment" validate:"omitempty,max=500"`
	Tags        []string           `json:"tags" validate:"omitempty,max=10"`
	IsAnonymous bool               `json:"is_anonymous"`
}

type RatingUpdateRequest struct {
	Rating      float64  `json:"rating" validate:"omitempty,rating_value"`
	Comment     string   `json:"comment" validate:"omitempty,max=500"`
	Tags        []string `json:"tags" validate:"omitempty,max=10"`
	IsAnonymous bool     `json:"is_anonymous"`
}

type RatingReportRequest struct {
	RatingID primitive.ObjectID `json:"rating_id" validate:"required,object_id"`
	Reason   string             `json:"reason" validate:"required,max=255"`
	Details  string             `json:"details" validate:"omitempty,max=1000"`
}

type RatingResponseRequest struct {
	RatingID primitive.ObjectID `json:"rating_id" validate:"required,object_id"`
	Response string             `json:"response" validate:"required,min=10,max=500"`
}

func ValidateRatingCreate(req *RatingCreateRequest) ValidationErrors {
	errors := ValidateStruct(req)
	
	// Validate rating tags
	validTags := map[string][]string{
		"rider": {
			"polite",
			"punctual",
			"clean",
			"respectful",
			"good_communication",
			"easy_to_find",
			"patient",
			"friendly",
			"quiet",
			"talkative",
		},
		"driver": {
			"safe_driving",
			"clean_vehicle",
			"punctual",
			"polite",
			"good_communication",
			"helpful",
			"professional",
			"smooth_ride",
			"good_music",
			"air_conditioning",
			"route_knowledge",
			"vehicle_comfort",
		},
	}
	
	allowedTags := validTags[req.RaterType]
	for i, tag := range req.Tags {
		if !contains(allowedTags, tag) {
			errors = append(errors, ValidationError{
				Field:   fmt.Sprintf("tags[%d]", i),
				Message: fmt.Sprintf("Invalid tag '%s' for %s rating", tag, req.RaterType),
			})
		}
	}
	
	// Validate comment content (basic profanity filter)
	if req.Comment != "" && containsProfanity(req.Comment) {
		errors = append(errors, ValidationError{
			Field:   "comment",
			Message: "Comment contains inappropriate language",
		})
	}
	
	// Validate rating consistency with comment sentiment
	if req.Rating <= 2.0 && req.Comment == "" {
		errors = append(errors, ValidationError{
			Field:   "comment",
			Message: "Comment is required for ratings 2 stars or below",
		})
	}
	
	return errors
}

func ValidateRatingUpdate(req *RatingUpdateRequest) ValidationErrors {
	errors := ValidateStruct(req)
	
	// Validate comment content
	if req.Comment != "" && containsProfanity(req.Comment) {
		errors = append(errors, ValidationError{
			Field:   "comment",
			Message: "Comment contains inappropriate language",
		})
	}
	
	return errors
}

func ValidateRatingReport(req *RatingReportRequest) ValidationErrors {
	errors := ValidateStruct(req)
	
	// Validate report reasons
	validReasons := []string{
		"inappropriate_language",
		"harassment",
		"false_information",
		"spam",
		"discrimination",
		"threatening_behavior",
		"irrelevant_content",
		"personal_information",
		"copyright_violation",
		"other",
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
			Message: "Please provide a valid report reason",
		})
	}
	
	return errors
}

func ValidateRatingResponse(req *RatingResponseRequest) ValidationErrors {
	errors := ValidateStruct(req)
	
	// Validate response content
	if containsProfanity(req.Response) {
		errors = append(errors, ValidationError{
			Field:   "response",
			Message: "Response contains inappropriate language",
		})
	}
	
	return errors
}

// Helper functions
func containsProfanity(text string) bool {
	// Simplified profanity filter - in production, use a comprehensive filter
	profanityWords := []string{
		"damn", "hell", "stupid", "idiot", "hate", "awful", "terrible",
		// Add more words as needed
	}
	
	lowerText := strings.ToLower(text)
	for _, word := range profanityWords {
		if strings.Contains(lowerText, word) {
			return true
		}
	}
	
	return false
}