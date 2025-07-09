package logger

import (
	"context"
	"io"
	"os"
	"time"

	"github.com/sirupsen/logrus"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

type Logger struct {
	logger *logrus.Logger
	fields logrus.Fields
}

type LogLevel string

const (
	DebugLevel LogLevel = "debug"
	InfoLevel  LogLevel = "info"
	WarnLevel  LogLevel = "warn"
	ErrorLevel LogLevel = "error"
	FatalLevel LogLevel = "fatal"
	PanicLevel LogLevel = "panic"
)

type Config struct {
	Level      LogLevel `json:"level"`
	Format     string   `json:"format"` // json, text
	Output     string   `json:"output"` // stdout, stderr, file path
	TimeFormat string   `json:"time_format"`
	Caller     bool     `json:"caller"`
	Colors     bool     `json:"colors"`
}

func NewLogger(config *Config) (*Logger, error) {
	logger := logrus.New()

	// Set level
	level, err := logrus.ParseLevel(string(config.Level))
	if err != nil {
		level = logrus.InfoLevel
	}
	logger.SetLevel(level)

	// Set formatter
	if config.Format == "json" {
		logger.SetFormatter(&logrus.JSONFormatter{
			TimestampFormat: config.TimeFormat,
		})
	} else {
		logger.SetFormatter(&logrus.TextFormatter{
			TimestampFormat: config.TimeFormat,
			ForceColors:     config.Colors,
			DisableColors:   !config.Colors,
		})
	}

	// Set output
	if config.Output == "stderr" {
		logger.SetOutput(os.Stderr)
	} else if config.Output == "stdout" || config.Output == "" {
		logger.SetOutput(os.Stdout)
	} else {
		file, err := os.OpenFile(config.Output, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0666)
		if err != nil {
			return nil, err
		}
		logger.SetOutput(file)
	}

	// Set caller reporting
	logger.SetReportCaller(config.Caller)

	return &Logger{
		logger: logger,
		fields: make(logrus.Fields),
	}, nil
}

func (l *Logger) WithField(key string, value interface{}) *Logger {
	newFields := make(logrus.Fields)
	for k, v := range l.fields {
		newFields[k] = v
	}
	newFields[key] = value

	return &Logger{
		logger: l.logger,
		fields: newFields,
	}
}

func (l *Logger) WithFields(fields map[string]interface{}) *Logger {
	newFields := make(logrus.Fields)
	for k, v := range l.fields {
		newFields[k] = v
	}
	for k, v := range fields {
		newFields[k] = v
	}

	return &Logger{
		logger: l.logger,
		fields: newFields,
	}
}

func (l *Logger) WithContext(ctx context.Context) *Logger {
	fields := extractContextFields(ctx)
	return l.WithFields(fields)
}

func (l *Logger) WithError(err error) *Logger {
	return l.WithField("error", err.Error())
}

func (l *Logger) WithUserID(userID primitive.ObjectID) *Logger {
	return l.WithField("user_id", userID.Hex())
}

func (l *Logger) WithRideID(rideID primitive.ObjectID) *Logger {
	return l.WithField("ride_id", rideID.Hex())
}

func (l *Logger) WithRequestID(requestID string) *Logger {
	return l.WithField("request_id", requestID)
}

func (l *Logger) Debug(msg string) {
	l.logger.WithFields(l.fields).Debug(msg)
}

func (l *Logger) Debugf(format string, args ...interface{}) {
	l.logger.WithFields(l.fields).Debugf(format, args...)
}

func (l *Logger) Info(msg string) {
	l.logger.WithFields(l.fields).Info(msg)
}

func (l *Logger) Infof(format string, args ...interface{}) {
	l.logger.WithFields(l.fields).Infof(format, args...)
}

func (l *Logger) Warn(msg string) {
	l.logger.WithFields(l.fields).Warn(msg)
}

func (l *Logger) Warnf(format string, args ...interface{}) {
	l.logger.WithFields(l.fields).Warnf(format, args...)
}

func (l *Logger) Error(msg string) {
	l.logger.WithFields(l.fields).Error(msg)
}

func (l *Logger) Errorf(format string, args ...interface{}) {
	l.logger.WithFields(l.fields).Errorf(format, args...)
}

func (l *Logger) Fatal(msg string) {
	l.logger.WithFields(l.fields).Fatal(msg)
}

func (l *Logger) Fatalf(format string, args ...interface{}) {
	l.logger.WithFields(l.fields).Fatalf(format, args...)
}

func (l *Logger) Panic(msg string) {
	l.logger.WithFields(l.fields).Panic(msg)
}

func (l *Logger) Panicf(format string, args ...interface{}) {
	l.logger.WithFields(l.fields).Panicf(format, args...)
}

// Structured logging methods
func (l *Logger) LogUserAction(userID primitive.ObjectID, action string, details map[string]interface{}) {
	fields := map[string]interface{}{
		"user_id": userID.Hex(),
		"action":  action,
		"type":    "user_action",
	}

	for k, v := range details {
		fields[k] = v
	}

	l.WithFields(fields).Info("User action performed")
}

func (l *Logger) LogRideEvent(rideID primitive.ObjectID, event string, details map[string]interface{}) {
	fields := map[string]interface{}{
		"ride_id": rideID.Hex(),
		"event":   event,
		"type":    "ride_event",
	}

	for k, v := range details {
		fields[k] = v
	}

	l.WithFields(fields).Info("Ride event occurred")
}

func (l *Logger) LogPaymentEvent(paymentID primitive.ObjectID, event string, amount float64, currency string) {
	l.WithFields(map[string]interface{}{
		"payment_id": paymentID.Hex(),
		"event":      event,
		"amount":     amount,
		"currency":   currency,
		"type":       "payment_event",
	}).Info("Payment event occurred")
}

func (l *Logger) LogAPIRequest(method, endpoint string, statusCode int, duration time.Duration, userID *primitive.ObjectID) {
	fields := map[string]interface{}{
		"method":      method,
		"endpoint":    endpoint,
		"status_code": statusCode,
		"duration_ms": duration.Milliseconds(),
		"type":        "api_request",
	}

	if userID != nil {
		fields["user_id"] = userID.Hex()
	}

	l.WithFields(fields).Info("API request processed")
}

func (l *Logger) LogSecurityEvent(eventType string, severity string, details map[string]interface{}) {
	fields := map[string]interface{}{
		"event_type": eventType,
		"severity":   severity,
		"type":       "security_event",
	}

	for k, v := range details {
		fields[k] = v
	}

	if severity == "high" || severity == "critical" {
		l.WithFields(fields).Error("Security event detected")
	} else {
		l.WithFields(fields).Warn("Security event detected")
	}
}

func (l *Logger) LogPerformanceMetric(metric string, value float64, unit string, tags map[string]string) {
	fields := map[string]interface{}{
		"metric": metric,
		"value":  value,
		"unit":   unit,
		"type":   "performance_metric",
	}

	for k, v := range tags {
		fields[k] = v
	}

	l.WithFields(fields).Info("Performance metric recorded")
}

func (l *Logger) SetOutput(output io.Writer) {
	l.logger.SetOutput(output)
}

func (l *Logger) SetLevel(level LogLevel) {
	logrusLevel, err := logrus.ParseLevel(string(level))
	if err != nil {
		logrusLevel = logrus.InfoLevel
	}
	l.logger.SetLevel(logrusLevel)
}

// Helper function to extract fields from context
func extractContextFields(ctx context.Context) map[string]interface{} {
	fields := make(map[string]interface{})

	// Extract common context values
	if userID := ctx.Value("user_id"); userID != nil {
		if oid, ok := userID.(primitive.ObjectID); ok {
			fields["user_id"] = oid.Hex()
		} else if str, ok := userID.(string); ok {
			fields["user_id"] = str
		}
	}

	if requestID := ctx.Value("request_id"); requestID != nil {
		if str, ok := requestID.(string); ok {
			fields["request_id"] = str
		}
	}

	if rideID := ctx.Value("ride_id"); rideID != nil {
		if oid, ok := rideID.(primitive.ObjectID); ok {
			fields["ride_id"] = oid.Hex()
		} else if str, ok := rideID.(string); ok {
			fields["ride_id"] = str
		}
	}

	return fields
}
