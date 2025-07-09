package utils

import (
	"encoding/json"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
)

type APIResponse struct {
	Status    string      `json:"status"`
	Message   string      `json:"message,omitempty"`
	Data      interface{} `json:"data,omitempty"`
	Error     *APIError   `json:"error,omitempty"`
	Meta      *Meta       `json:"meta,omitempty"`
	Timestamp time.Time   `json:"timestamp"`
}

type APIError struct {
	Code    string            `json:"code"`
	Message string            `json:"message"`
	Details map[string]string `json:"details,omitempty"`
}

type Meta struct {
	Pagination *PaginationMeta `json:"pagination,omitempty"`
	Total      int64           `json:"total,omitempty"`
	Count      int             `json:"count,omitempty"`
}

func SuccessResponse(c *gin.Context, message string, data interface{}) {
	c.JSON(http.StatusOK, APIResponse{
		Status:    StatusSuccess,
		Message:   message,
		Data:      data,
		Timestamp: time.Now(),
	})
}

func SuccessResponseWithMeta(c *gin.Context, message string, data interface{}, meta *Meta) {
	c.JSON(http.StatusOK, APIResponse{
		Status:    StatusSuccess,
		Message:   message,
		Data:      data,
		Meta:      meta,
		Timestamp: time.Now(),
	})
}

func ErrorResponse(c *gin.Context, statusCode int, code, message string) {
	c.JSON(statusCode, APIResponse{
		Status: StatusError,
		Error: &APIError{
			Code:    code,
			Message: message,
		},
		Timestamp: time.Now(),
	})
}

func ErrorResponseWithDetails(c *gin.Context, statusCode int, code, message string, details map[string]string) {
	c.JSON(statusCode, APIResponse{
		Status: StatusError,
		Error: &APIError{
			Code:    code,
			Message: message,
			Details: details,
		},
		Timestamp: time.Now(),
	})
}

func ValidationErrorResponse(c *gin.Context, errors map[string]string) {
	ErrorResponseWithDetails(c, http.StatusBadRequest, "VALIDATION_ERROR", ErrValidationFailed, errors)
}

func InternalServerErrorResponse(c *gin.Context) {
	ErrorResponse(c, http.StatusInternalServerError, "INTERNAL_ERROR", ErrInternalServer)
}

func UnauthorizedResponse(c *gin.Context) {
	ErrorResponse(c, http.StatusUnauthorized, "UNAUTHORIZED", ErrUnauthorized)
}

func ForbiddenResponse(c *gin.Context) {
	ErrorResponse(c, http.StatusForbidden, "FORBIDDEN", ErrForbidden)
}

func NotFoundResponse(c *gin.Context, resource string) {
	ErrorResponse(c, http.StatusNotFound, "NOT_FOUND", resource+" not found")
}

func ConflictResponse(c *gin.Context, message string) {
	ErrorResponse(c, http.StatusConflict, "CONFLICT", message)
}

func BadRequestResponse(c *gin.Context, message string) {
	ErrorResponse(c, http.StatusBadRequest, "BAD_REQUEST", message)
}

func CreatedResponse(c *gin.Context, message string, data interface{}) {
	c.JSON(http.StatusCreated, APIResponse{
		Status:    StatusSuccess,
		Message:   message,
		Data:      data,
		Timestamp: time.Now(),
	})
}

func NoContentResponse(c *gin.Context) {
	c.Status(http.StatusNoContent)
}

func PrettyJSON(data interface{}) string {
	b, _ := json.MarshalIndent(data, "", "  ")
	return string(b)
}
