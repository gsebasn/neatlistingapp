package middleware

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"strings"
	"time"

	"listing/internal/infrastructure/config"
	"listing/internal/infrastructure/logging"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
)

// bodyLogWriter is a custom response writer that captures the response body
type bodyLogWriter struct {
	gin.ResponseWriter
	body *bytes.Buffer
}

// Write captures the response body
func (w bodyLogWriter) Write(b []byte) (int, error) {
	w.body.Write(b)
	return w.ResponseWriter.Write(b)
}

// maskValue applies the specified mask pattern to a value
func maskValue(value string, pattern config.MaskPattern, replacement string) string {
	if len(value) <= 4 {
		return replacement
	}

	switch pattern {
	case config.FullMask:
		return replacement
	case config.FirstLast:
		return fmt.Sprintf("%c%s%c", value[0], replacement, value[len(value)-1])
	case config.LastFour:
		return fmt.Sprintf("%s%s", replacement, value[len(value)-4:])
	case config.FirstFour:
		return fmt.Sprintf("%s%s", value[:4], replacement)
	case config.MiddleMask:
		return fmt.Sprintf("%c%s%c", value[0], replacement, value[len(value)-1])
	default:
		return replacement
	}
}

// maskSensitiveData masks sensitive fields in JSON data
func maskSensitiveData(data []byte, maskedFields []string, pattern config.MaskPattern, replacement string) []byte {
	if len(data) == 0 || len(maskedFields) == 0 {
		return data
	}

	// Try to parse as JSON
	var jsonData interface{}
	if err := json.Unmarshal(data, &jsonData); err != nil {
		return data // Return original if not JSON
	}

	// Create a map of masked fields for quick lookup
	maskedFieldsMap := make(map[string]bool)
	for _, field := range maskedFields {
		maskedFieldsMap[field] = true
	}

	// Recursively mask sensitive fields
	maskedData := maskJSONValue(jsonData, maskedFieldsMap, pattern, replacement)

	// Convert back to JSON
	result, err := json.Marshal(maskedData)
	if err != nil {
		return data // Return original if marshaling fails
	}

	return result
}

// maskJSONValue recursively masks sensitive fields in JSON data
func maskJSONValue(value interface{}, maskedFields map[string]bool, pattern config.MaskPattern, replacement string) interface{} {
	switch v := value.(type) {
	case map[string]interface{}:
		result := make(map[string]interface{})
		for key, val := range v {
			if maskedFields[key] {
				if strVal, ok := val.(string); ok {
					result[key] = maskValue(strVal, pattern, replacement)
				} else {
					result[key] = replacement
				}
			} else {
				result[key] = maskJSONValue(val, maskedFields, pattern, replacement)
			}
		}
		return result
	case []interface{}:
		result := make([]interface{}, len(v))
		for i, val := range v {
			result[i] = maskJSONValue(val, maskedFields, pattern, replacement)
		}
		return result
	default:
		return value
	}
}

// truncateBody truncates the body if it exceeds the size limit
func truncateBody(body []byte, limit int64) []byte {
	if int64(len(body)) <= limit {
		return body
	}
	return append(body[:limit], []byte("... (truncated)")...)
}

// getEndpointConfig returns the configuration for a specific endpoint
func getEndpointConfig(path string, cfg *config.Config) *config.EndpointConfig {
	for _, endpoint := range cfg.Logging.EndpointConfigs {
		if strings.HasPrefix(path, endpoint.Path) {
			return &endpoint
		}
	}
	return nil
}

// LoggingMiddleware creates a middleware that logs HTTP requests
func LoggingMiddleware(logger *logging.Logger, cfg *config.Config) gin.HandlerFunc {
	return func(c *gin.Context) {
		start := time.Now()
		path := c.Request.URL.Path
		query := c.Request.URL.RawQuery
		clientIP := c.ClientIP()
		method := c.Request.Method
		userAgent := c.Request.UserAgent()
		correlationID := GetCorrelationID(c)

		// Get endpoint-specific configuration
		endpointConfig := getEndpointConfig(path, cfg)
		bodySizeLimit := cfg.Logging.BodySizeLimit
		maskedFields := cfg.Logging.MaskedFields
		maskPattern := cfg.Logging.MaskPattern

		if endpointConfig != nil {
			bodySizeLimit = endpointConfig.BodySizeLimit
			maskedFields = endpointConfig.MaskedFields
			maskPattern = endpointConfig.MaskPattern
		}

		// Capture request body for dev/test environments
		var requestBody []byte
		if cfg.Env == config.Development || cfg.Env == config.Test {
			if c.Request.Body != nil {
				requestBody, _ = io.ReadAll(c.Request.Body)
				// Restore the request body for subsequent middleware/handlers
				c.Request.Body = io.NopCloser(bytes.NewBuffer(requestBody))

				// Apply size limit and masking
				requestBody = truncateBody(requestBody, bodySizeLimit)
				requestBody = maskSensitiveData(requestBody, maskedFields, maskPattern, cfg.Logging.MaskReplacement)
			}
		}

		// Capture response body for dev/test environments
		var responseBody *bytes.Buffer
		if cfg.Env == config.Development || cfg.Env == config.Test {
			responseBody = bytes.NewBufferString("")
			writer := &bodyLogWriter{ResponseWriter: c.Writer, body: responseBody}
			c.Writer = writer
		}

		// Process request
		c.Next()

		// Calculate latency
		latency := time.Since(start)
		statusCode := c.Writer.Status()
		errorMessage := c.Errors.ByType(gin.ErrorTypePrivate).String()

		// Prepare log fields
		fields := []zap.Field{
			zap.String("method", method),
			zap.String("path", path),
			zap.String("query", query),
			zap.String("ip", clientIP),
			zap.String("user-agent", userAgent),
			zap.Int("status", statusCode),
			zap.Duration("latency", latency),
			zap.String("error", errorMessage),
			zap.String("correlation_id", correlationID),
		}

		// Add request/response body for dev/test environments
		if cfg.Env == config.Development || cfg.Env == config.Test {
			if len(requestBody) > 0 {
				fields = append(fields, zap.String("request_body", string(requestBody)))
			}
			if responseBody != nil && responseBody.Len() > 0 {
				// Apply size limit and masking to response body
				responseBytes := responseBody.Bytes()
				responseBytes = truncateBody(responseBytes, bodySizeLimit)
				responseBytes = maskSensitiveData(responseBytes, maskedFields, maskPattern, cfg.Logging.MaskReplacement)
				fields = append(fields, zap.String("response_body", string(responseBytes)))
			}
		}

		// Log request details
		logger.Info("HTTP Request", fields...)

		// Log errors with additional context
		if statusCode >= 400 {
			logger.Error("HTTP Request Error", fields...)
		}
	}
}
