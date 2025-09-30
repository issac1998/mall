package utils

import (
	"fmt"
	"reflect"
	"regexp"
	"strconv"
	"strings"

	"github.com/gin-gonic/gin/binding"
	"github.com/go-playground/validator/v10"
)

var (
	// Phone number regular expression
	phoneRegex = regexp.MustCompile(`^1[3-9]\d{9}$`)
	// Email regular expression
	emailRegex = regexp.MustCompile(`^[a-zA-Z0-9._%+-]+@[a-zA-Z0-9.-]+\.[a-zA-Z]{2,}$`)
)

// ValidateStruct validates struct
func ValidateStruct(obj interface{}) error {
	if err := binding.Validator.ValidateStruct(obj); err != nil {
		return formatValidationError(err)
	}
	return nil
}

// formatValidationError formats validation error
func formatValidationError(err error) error {
	if validationErrors, ok := err.(validator.ValidationErrors); ok {
		var messages []string
		for _, fieldError := range validationErrors {
			message := getFieldErrorMessage(fieldError)
			messages = append(messages, message)
		}
		return NewError(CodeInvalidParam, strings.Join(messages, "; "))
	}
	return NewErrorWithErr(CodeInvalidParam, "validation failed", err)
}

// getFieldErrorMessage gets field error message
func getFieldErrorMessage(fieldError validator.FieldError) string {
	field := getFieldName(fieldError)
	tag := fieldError.Tag()
	param := fieldError.Param()

	switch tag {
	case "required":
		return fmt.Sprintf("%s is required", field)
	case "min":
		return fmt.Sprintf("%s must be at least %s", field, param)
	case "max":
		return fmt.Sprintf("%s must be at most %s", field, param)
	case "len":
		return fmt.Sprintf("%s must be exactly %s characters", field, param)
	case "email":
		return fmt.Sprintf("%s must be a valid email address", field)
	case "phone":
		return fmt.Sprintf("%s must be a valid phone number", field)
	case "positive":
		return fmt.Sprintf("%s must be positive", field)
	case "nonnegative":
		return fmt.Sprintf("%s must be non-negative", field)
	case "oneof":
		return fmt.Sprintf("%s must be one of: %s", field, param)
	case "gte":
		return fmt.Sprintf("%s must be greater than or equal to %s", field, param)
	case "lte":
		return fmt.Sprintf("%s must be less than or equal to %s", field, param)
	case "gt":
		return fmt.Sprintf("%s must be greater than %s", field, param)
	case "lt":
		return fmt.Sprintf("%s must be less than %s", field, param)
	case "numeric":
		return fmt.Sprintf("%s must be numeric", field)
	case "alpha":
		return fmt.Sprintf("%s must contain only letters", field)
	case "alphanum":
		return fmt.Sprintf("%s must contain only letters and numbers", field)
	default:
		return fmt.Sprintf("%s validation failed", field)
	}
}

// getFieldName gets field name
func getFieldName(fieldError validator.FieldError) string {
	// Convert camelCase to snake_case
	return camelToSnake(fieldError.Field())
}

// camelToSnake converts camelCase to snake_case
func camelToSnake(s string) string {
	var result strings.Builder
	for i, r := range s {
		if i > 0 && r >= 'A' && r <= 'Z' {
			// Check if the previous character is also uppercase, if so don't add underscore (handle consecutive uppercase letters)
			if i > 1 && s[i-1] >= 'A' && s[i-1] <= 'Z' {
				// If it's the last character and uppercase
				if i == len(s)-1 {
					result.WriteRune('_')
				} else if s[i+1] >= 'a' && s[i+1] <= 'z' {
					result.WriteRune('_')
				}
			} else {
				result.WriteRune('_')
			}
		}
		result.WriteRune(r)
	}
	return strings.ToLower(result.String())
}

// RegisterCustomValidators registers custom validators
func RegisterCustomValidators() {
	if v, ok := binding.Validator.Engine().(*validator.Validate); ok {
		// Register phone number validator
		v.RegisterValidation("phone", validatePhone)
		// Register email validator
		v.RegisterValidation("email", validateEmail)
		// Register positive integer validator
		v.RegisterValidation("positive", validatePositive)
		// Register non-negative integer validator
		v.RegisterValidation("nonnegative", validateNonNegative)
	}

	// Register tag name function
	if v, ok := binding.Validator.Engine().(*validator.Validate); ok {
		v.RegisterTagNameFunc(func(fld reflect.StructField) string {
			name := strings.SplitN(fld.Tag.Get("json"), ",", 2)[0]
			if name == "-" {
				return ""
			}
			return name
		})
	}
}

// validatePhone validates phone number
func validatePhone(fl validator.FieldLevel) bool {
	phone := fl.Field().String()
	return phoneRegex.MatchString(phone)
}

// validateEmail validates email
func validateEmail(fl validator.FieldLevel) bool {
	email := fl.Field().String()
	return emailRegex.MatchString(email)
}

// validatePositive validates positive integer
func validatePositive(fl validator.FieldLevel) bool {
	switch fl.Field().Kind() {
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		return fl.Field().Int() > 0
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
		return fl.Field().Uint() > 0
	case reflect.Float32, reflect.Float64:
		return fl.Field().Float() > 0
	default:
		return false
	}
}

// validateNonNegative validates non-negative number
func validateNonNegative(fl validator.FieldLevel) bool {
	switch fl.Field().Kind() {
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		return fl.Field().Int() >= 0
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
		return true // uint types are inherently non-negative
	case reflect.Float32, reflect.Float64:
		return fl.Field().Float() >= 0
	default:
		return false
	}
}

// ValidateID validates ID parameter
func ValidateID(id string) (int64, error) {
	if id == "" {
		return 0, NewError(CodeInvalidParam, "ID cannot be empty")
	}

	idInt, err := strconv.ParseInt(id, 10, 64)
	if err != nil {
		return 0, NewError(CodeInvalidParam, "ID must be a valid integer")
	}

	if idInt <= 0 {
		return 0, NewError(CodeInvalidParam, "ID must be positive")
	}

	return idInt, nil
}

// ValidatePage validates pagination parameters
func ValidatePage(page, pageSize int) error {
	if page <= 0 {
		return NewError(CodeInvalidParam, "page must be positive")
	}

	if pageSize <= 0 || pageSize > 100 {
		return NewError(CodeInvalidParam, "pageSize must be between 1 and 100")
	}

	return nil
}