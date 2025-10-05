package utils

import (
	"regexp"
	"strconv"
	"strings"
)

// ValidateID validates ID string and returns int64
func ValidateID(idStr string) (int64, error) {
	if idStr == "" {
		return 0, NewError(CodeInvalidParam, "ID cannot be empty")
	}

	id, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		return 0, NewError(CodeInvalidParam, "invalid ID format")
	}

	if id <= 0 {
		return 0, NewError(CodeInvalidParam, "ID must be positive")
	}

	return id, nil
}

// ValidatePage validates pagination parameters
func ValidatePage(page, size int) error {
	if page <= 0 {
		return NewError(CodeInvalidParam, "page must be positive")
	}

	if size <= 0 {
		return NewError(CodeInvalidParam, "page size must be positive")
	}

	if size > 100 {
		return NewError(CodeInvalidParam, "page size cannot exceed 100")
	}

	return nil
}

// ValidateEmail validates email format
func ValidateEmail(email string) error {
	if email == "" {
		return NewError(CodeInvalidParam, "email cannot be empty")
	}

	emailRegex := regexp.MustCompile(`^[a-zA-Z0-9._%+-]+@[a-zA-Z0-9.-]+\.[a-zA-Z]{2,}$`)
	if !emailRegex.MatchString(email) {
		return NewError(CodeInvalidParam, "invalid email format")
	}

	return nil
}

// ValidatePhone validates phone number format
func ValidatePhone(phone string) error {
	if phone == "" {
		return NewError(CodeInvalidParam, "phone cannot be empty")
	}

	phoneRegex := regexp.MustCompile(`^1[3-9]\d{9}$`)
	if !phoneRegex.MatchString(phone) {
		return NewError(CodeInvalidParam, "invalid phone format")
	}

	return nil
}

// ValidatePassword validates password strength
func ValidatePassword(password string) error {
	if password == "" {
		return NewError(CodeInvalidParam, "password cannot be empty")
	}

	if len(password) < 6 {
		return NewError(CodeInvalidParam, "password must be at least 6 characters")
	}

	if len(password) > 20 {
		return NewError(CodeInvalidParam, "password cannot exceed 20 characters")
	}

	return nil
}

// camelToSnake converts camelCase to snake_case
func camelToSnake(s string) string {
	var result strings.Builder
	for i, r := range s {
		if i > 0 && r >= 'A' && r <= 'Z' {
			// Check if previous character is also uppercase
			prevRune := rune(s[i-1])
			if prevRune < 'A' || prevRune > 'Z' {
				result.WriteRune('_')
			}
		}
		if r >= 'A' && r <= 'Z' {
			result.WriteRune(r - 'A' + 'a')
		} else {
			result.WriteRune(r)
		}
	}
	return result.String()
}