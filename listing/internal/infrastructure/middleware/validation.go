package middleware

import (
	"net/http"
	"reflect"

	"listing/internal/infrastructure/validation"

	"github.com/gin-gonic/gin"
)

// Validator is a middleware that validates request bodies
type Validator struct {
	validator *validation.Validator
}

// NewValidator creates a new Validator middleware
func NewValidator() *Validator {
	return &Validator{
		validator: validation.New(),
	}
}

// ValidateRequest validates the request body against the provided type
func (v *Validator) ValidateRequest(modelType reflect.Type) gin.HandlerFunc {
	return func(c *gin.Context) {
		// Create a new instance of the model type
		model := reflect.New(modelType).Interface()

		// Bind the request body to the model
		if err := c.ShouldBindJSON(model); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{
				"error": "Invalid request format",
			})
			c.Abort()
			return
		}

		// Validate the model
		if err := v.validator.Validate(model); err != nil {
			validationResponse := validation.TranslateValidationErrors(err)
			c.JSON(http.StatusBadRequest, validationResponse)
			c.Abort()
			return
		}

		// Store the validated model in the context for the handler to use
		c.Set("validated_model", model)
		c.Next()
	}
}

// ValidateQuery validates query parameters against the provided type
func (v *Validator) ValidateQuery(modelType reflect.Type) gin.HandlerFunc {
	return func(c *gin.Context) {
		// Create a new instance of the model type
		model := reflect.New(modelType).Interface()

		// Bind query parameters to the model
		if err := c.ShouldBindQuery(model); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{
				"error": "Invalid query parameters",
			})
			c.Abort()
			return
		}

		// Validate the model
		if err := v.validator.Validate(model); err != nil {
			validationResponse := validation.TranslateValidationErrors(err)
			c.JSON(http.StatusBadRequest, validationResponse)
			c.Abort()
			return
		}

		// Store the validated model in the context
		c.Set("validated_query", model)
		c.Next()
	}
}

// ValidatePath validates path parameters against the provided type
func (v *Validator) ValidatePath(modelType reflect.Type) gin.HandlerFunc {
	return func(c *gin.Context) {
		// Create a new instance of the model type
		model := reflect.New(modelType).Interface()

		// Bind path parameters to the model
		if err := c.ShouldBindUri(model); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{
				"error": "Invalid path parameters",
			})
			c.Abort()
			return
		}

		// Validate the model
		if err := v.validator.Validate(model); err != nil {
			validationResponse := validation.TranslateValidationErrors(err)
			c.JSON(http.StatusBadRequest, validationResponse)
			c.Abort()
			return
		}

		// Store the validated model in the context
		c.Set("validated_path", model)
		c.Next()
	}
}

// GetValidatedModel retrieves the validated request body from the context
func GetValidatedModel(c *gin.Context) interface{} {
	return c.MustGet("validated_model")
}

// GetValidatedQuery retrieves the validated query parameters from the context
func GetValidatedQuery(c *gin.Context) interface{} {
	return c.MustGet("validated_query")
}

// GetValidatedPath retrieves the validated path parameters from the context
func GetValidatedPath(c *gin.Context) interface{} {
	return c.MustGet("validated_path")
}
