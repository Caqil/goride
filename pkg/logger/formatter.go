package logger

import (
	"bytes"
	"encoding/json"
	"fmt"
	"math"
	"sort"
	"strings"
	"time"

	"github.com/sirupsen/logrus"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

type CustomJSONFormatter struct {
	TimestampFormat string
	PrettyPrint     bool
	AppName         string
	Version         string
}

type CustomTextFormatter struct {
	TimestampFormat string
	ForceColors     bool
	DisableColors   bool
	AppName         string
	Version         string
}

func (f *CustomJSONFormatter) Format(entry *logrus.Entry) ([]byte, error) {
	data := make(map[string]interface{})

	// Add timestamp
	timestampFormat := f.TimestampFormat
	if timestampFormat == "" {
		timestampFormat = time.RFC3339
	}
	data["timestamp"] = entry.Time.Format(timestampFormat)

	// Add level
	data["level"] = entry.Level.String()

	// Add message
	data["message"] = entry.Message

	// Add app info
	if f.AppName != "" {
		data["app"] = f.AppName
	}
	if f.Version != "" {
		data["version"] = f.Version
	}

	// Add caller info
	if entry.HasCaller() {
		data["caller"] = fmt.Sprintf("%s:%d", entry.Caller.File, entry.Caller.Line)
		data["function"] = entry.Caller.Function
	}

	// Add fields
	for k, v := range entry.Data {
		data[k] = v
	}

	var b *bytes.Buffer
	if entry.Buffer != nil {
		b = entry.Buffer
	} else {
		b = &bytes.Buffer{}
	}

	encoder := json.NewEncoder(b)
	if f.PrettyPrint {
		encoder.SetIndent("", "  ")
	}

	if err := encoder.Encode(data); err != nil {
		return nil, fmt.Errorf("failed to marshal fields to JSON: %w", err)
	}

	return b.Bytes(), nil
}

func (f *CustomTextFormatter) Format(entry *logrus.Entry) ([]byte, error) {
	var b *bytes.Buffer
	if entry.Buffer != nil {
		b = entry.Buffer
	} else {
		b = &bytes.Buffer{}
	}

	// Timestamp
	timestampFormat := f.TimestampFormat
	if timestampFormat == "" {
		timestampFormat = "2006-01-02 15:04:05"
	}

	// Color codes
	var levelColor string
	if !f.DisableColors && (f.ForceColors || isTerminal()) {
		switch entry.Level {
		case logrus.ErrorLevel, logrus.FatalLevel, logrus.PanicLevel:
			levelColor = "\033[31m" // Red
		case logrus.WarnLevel:
			levelColor = "\033[33m" // Yellow
		case logrus.InfoLevel:
			levelColor = "\033[36m" // Cyan
		case logrus.DebugLevel:
			levelColor = "\033[37m" // White
		default:
			levelColor = "\033[0m" // Reset
		}
	}

	// Format log entry
	fmt.Fprintf(b, "%s[%s%s%s] ",
		entry.Time.Format(timestampFormat),
		levelColor,
		strings.ToUpper(entry.Level.String()),
		"\033[0m", // Reset color
	)

	// Add app info
	if f.AppName != "" {
		fmt.Fprintf(b, "[%s] ", f.AppName)
	}

	// Add caller info
	if entry.HasCaller() {
		fmt.Fprintf(b, "[%s:%d] ", entry.Caller.File, entry.Caller.Line)
	}

	// Add message
	fmt.Fprintf(b, "%s", entry.Message)

	// Add fields
	if len(entry.Data) > 0 {
		fields := make([]string, 0, len(entry.Data))
		for k, v := range entry.Data {
			fields = append(fields, fmt.Sprintf("%s=%v", k, v))
		}
		sort.Strings(fields)
		fmt.Fprintf(b, " %s", strings.Join(fields, " "))
	}

	b.WriteByte('\n')

	return b.Bytes(), nil
}

// Check if output is a terminal
func isTerminal() bool {
	// Simplified check - in production, use a proper terminal detection library
	return false
}

// Audit logger for compliance and security
type AuditLogger struct {
	logger *Logger
}

func NewAuditLogger(config *Config) (*AuditLogger, error) {
	// Force JSON format for audit logs
	config.Format = "json"

	logger, err := NewLogger(config)
	if err != nil {
		return nil, err
	}

	return &AuditLogger{
		logger: logger,
	}, nil
}

func (a *AuditLogger) LogAction(action, resource string, userID *primitive.ObjectID, details map[string]interface{}) {
	fields := map[string]interface{}{
		"action":    action,
		"resource":  resource,
		"timestamp": time.Now().UTC(),
		"type":      "audit",
	}

	if userID != nil {
		fields["user_id"] = userID.Hex()
	}

	for k, v := range details {
		fields[k] = v
	}

	a.logger.WithFields(fields).Info("Audit log entry")
}

func (a *AuditLogger) LogDataAccess(table, operation string, userID *primitive.ObjectID, recordID string, sensitive bool) {
	fields := map[string]interface{}{
		"table":     table,
		"operation": operation,
		"record_id": recordID,
		"sensitive": sensitive,
		"type":      "data_access",
	}

	if userID != nil {
		fields["user_id"] = userID.Hex()
	}

	a.logger.WithFields(fields).Info("Data access logged")
}

func (a *AuditLogger) LogAuthEvent(eventType string, userID *primitive.ObjectID, ipAddress, userAgent string, success bool) {
	fields := map[string]interface{}{
		"event_type": eventType,
		"ip_address": ipAddress,
		"user_agent": userAgent,
		"success":    success,
		"type":       "auth_event",
	}

	if userID != nil {
		fields["user_id"] = userID.Hex()
	}

	a.logger.WithFields(fields).Info("Authentication event logged")
}

func (a *AuditLogger) LogPaymentAudit(paymentID primitive.ObjectID, amount float64, currency string, method string, status string) {
	a.logger.WithFields(map[string]interface{}{
		"payment_id": paymentID.Hex(),
		"amount":     amount,
		"currency":   currency,
		"method":     method,
		"status":     status,
		"type":       "payment_audit",
	}).Info("Payment audit logged")
}

func (a *AuditLogger) LogConfigChange(setting, oldValue, newValue string, changedBy *primitive.ObjectID) {
	fields := map[string]interface{}{
		"setting":   setting,
		"old_value": oldValue,
		"new_value": newValue,
		"type":      "config_change",
	}

	if changedBy != nil {
		fields["changed_by"] = changedBy.Hex()
	}

	a.logger.WithFields(fields).Info("Configuration change logged")
}

func (d *DemandForecaster) getWeatherDemandFactor(condition string) float64 {
	switch condition {
	case "rain", "snow":
		return 1.4 // More people request rides
	case "storm":
		return 1.8
	case "clear", "sunny":
		return 0.9 // People might walk more
	default:
		return 1.0
	}
}

func (d *DemandForecaster) getEventFactor(events []string) float64 {
	if len(events) == 0 {
		return 1.0
	}

	maxFactor := 1.0
	for _, event := range events {
		switch event {
		case "concert":
			maxFactor = math.Max(maxFactor, 3.0)
		case "sports":
			maxFactor = math.Max(maxFactor, 2.5)
		case "conference":
			maxFactor = math.Max(maxFactor, 1.8)
		case "festival":
			maxFactor = math.Max(maxFactor, 2.2)
		}
	}

	return maxFactor
}

func (d *DemandForecaster) categorizeDemand(predicted, historical float64) string {
	ratio := predicted / historical

	if ratio >= 2.0 {
		return "very_high"
	} else if ratio >= 1.5 {
		return "high"
	} else if ratio >= 1.2 {
		return "elevated"
	} else if ratio >= 0.8 {
		return "normal"
	} else {
		return "low"
	}
}

func (d *DemandForecaster) calculateSuggestedSurge(predicted, historical float64) float64 {
	ratio := predicted / historical

	if ratio >= 3.0 {
		return 3.0
	} else if ratio >= 2.5 {
		return 2.5
	} else if ratio >= 2.0 {
		return 2.0
	} else if ratio >= 1.5 {
		return 1.5
	} else if ratio >= 1.2 {
		return 1.2
	}

	return 1.0 // No surge
}

func (d *DemandForecaster) calculateDemandConfidence(features map[string]float64) float64 {
	// Simple confidence based on feature availability
	requiredFeatures := []string{"hour_of_day", "day_of_week", "historical_avg"}
	availableCount := 0

	for _, feature := range requiredFeatures {
		if _, exists := features[feature]; exists {
			availableCount++
		}
	}

	return float64(availableCount) / float64(len(requiredFeatures))
}
