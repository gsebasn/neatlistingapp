package validation

import (
	"fmt"
	"reflect"
	"strings"

	val "github.com/go-playground/validator/v10"
)

// Validator is a wrapper around validator.Validate with custom rules
type Validator struct {
	validate *val.Validate
}

// New creates a new Validator instance with custom rules
func New() *Validator {
	v := val.New()

	// Register custom validators
	v.RegisterValidation("notempty", notEmpty)
	v.RegisterValidation("price", validatePrice)
	v.RegisterValidation("coordinates", validateCoordinates)

	// Register custom type functions
	v.RegisterTagNameFunc(func(fld reflect.StructField) string {
		name := strings.SplitN(fld.Tag.Get("json"), ",", 2)[0]
		if name == "-" {
			return fld.Name
		}
		return name
	})

	return &Validator{validate: v}
}

// Validate validates a struct using the validator
func (v *Validator) Validate(i interface{}) error {
	return v.validate.Struct(i)
}

// notEmpty validates that a string is not empty after trimming
func notEmpty(fl val.FieldLevel) bool {
	value := fl.Field().String()
	return strings.TrimSpace(value) != ""
}

// validatePrice validates that a price is positive and has at most 2 decimal places
func validatePrice(fl val.FieldLevel) bool {
	value := fl.Field().Float()
	if value <= 0 {
		return false
	}
	// Check if there are more than 2 decimal places
	str := fmt.Sprintf("%.2f", value)
	return str == fmt.Sprintf("%.2f", value)
}

// validateCoordinates validates that coordinates are within valid ranges
func validateCoordinates(fl val.FieldLevel) bool {
	value := fl.Field().Float()
	return value >= -180 && value <= 180
}

// ValidationError represents a validation error
type ValidationError struct {
	Field   string `json:"field"`
	Message string `json:"message"`
	Value   string `json:"value,omitempty"`
	Tag     string `json:"tag,omitempty"`
}

// Error returns the error message
func (e ValidationError) Error() string {
	return fmt.Sprintf("%s: %s", e.Field, e.Message)
}

// ValidationErrors represents multiple validation errors
type ValidationErrors []ValidationError

// Error returns the error message
func (e ValidationErrors) Error() string {
	var messages []string
	for _, err := range e {
		messages = append(messages, err.Error())
	}
	return strings.Join(messages, "; ")
}

// ValidationResponse represents the complete validation response
type ValidationResponse struct {
	Errors     ValidationErrors `json:"errors"`
	ErrorCount int              `json:"error_count"`
	HasErrors  bool             `json:"has_errors"`
}

// TranslateValidationErrors translates validator errors to our custom error type
func TranslateValidationErrors(err error) ValidationResponse {
	if err == nil {
		return ValidationResponse{
			Errors:     nil,
			ErrorCount: 0,
			HasErrors:  false,
		}
	}

	validationErrors, ok := err.(val.ValidationErrors)
	if !ok {
		return ValidationResponse{
			Errors: ValidationErrors{{
				Field:   "error",
				Message: err.Error(),
			}},
			ErrorCount: 1,
			HasErrors:  true,
		}
	}

	var errors ValidationErrors
	errorMap := make(map[string]ValidationError) // For aggregating errors by field

	for _, e := range validationErrors {
		field := e.Field()
		tag := e.Tag()
		param := e.Param()
		value := fmt.Sprintf("%v", e.Value())

		// Create error message
		message := translateValidationError(field, tag, param)

		// If we already have an error for this field, append the new error message
		if existing, exists := errorMap[field]; exists {
			existing.Message = fmt.Sprintf("%s; %s", existing.Message, message)
			errorMap[field] = existing
		} else {
			errorMap[field] = ValidationError{
				Field:   field,
				Message: message,
				Value:   value,
				Tag:     tag,
			}
		}
	}

	// Convert map to slice
	for _, err := range errorMap {
		errors = append(errors, err)
	}

	return ValidationResponse{
		Errors:     errors,
		ErrorCount: len(errors),
		HasErrors:  true,
	}
}

// translateValidationError translates a validation error to a human-readable message
func translateValidationError(field, tag, param string) string {
	switch tag {
	case "required":
		return "is required"
	case "email":
		return "must be a valid email address"
	case "min":
		return fmt.Sprintf("must be at least %s", param)
	case "max":
		return fmt.Sprintf("must be at most %s", param)
	case "notempty":
		return "cannot be empty"
	case "price":
		return "must be a valid price (positive number with max 2 decimal places)"
	case "coordinates":
		return "must be valid coordinates (-180 to 180)"
	case "uuid":
		return "must be a valid UUID"
	case "oneof":
		return fmt.Sprintf("must be one of: %s", param)
	default:
		return fmt.Sprintf("failed validation: %s", tag)
	}
}
