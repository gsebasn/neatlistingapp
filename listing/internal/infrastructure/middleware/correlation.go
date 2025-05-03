package middleware

import (
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

const (
	// CorrelationIDHeader is the header name for correlation ID
	CorrelationIDHeader = "X-Correlation-ID"
	// CorrelationIDKey is the context key for correlation ID
	CorrelationIDKey = "correlation_id"
)

// CorrelationMiddleware adds a correlation ID to each request
func CorrelationMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		// Get correlation ID from header or generate new one
		correlationID := c.GetHeader(CorrelationIDHeader)
		if correlationID == "" {
			correlationID = uuid.New().String()
		}

		// Set correlation ID in context
		c.Set(CorrelationIDKey, correlationID)

		// Add correlation ID to response headers
		c.Header(CorrelationIDHeader, correlationID)

		c.Next()
	}
}

// GetCorrelationID returns the correlation ID from the context
func GetCorrelationID(c *gin.Context) string {
	if id, exists := c.Get(CorrelationIDKey); exists {
		return id.(string)
	}
	return ""
}
