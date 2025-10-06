package utils

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestValidateID(t *testing.T) {
	tests := []struct {
		input     string
		expected  int64
		wantError bool
	}{
		{"123", 123, false},
		{"0", 0, true},   // ID must be positive
		{"-1", 0, true},  // ID must be positive
		{"", 0, true},    // ID cannot be empty
		{"abc", 0, true}, // Invalid format
		{"123.45", 0, true}, // Invalid format
	}

	for _, tt := range tests {
		result, err := ValidateID(tt.input)
		if tt.wantError {
			assert.Error(t, err)
		} else {
			assert.NoError(t, err)
			assert.Equal(t, tt.expected, result)
		}
	}
}

func TestValidatePage(t *testing.T) {
	tests := []struct {
		page      int
		size      int
		wantError bool
	}{
		{1, 10, false},
		{1, 100, false},
		{0, 10, true},    // Page must be positive
		{-1, 10, true},   // Page must be positive
		{1, 0, true},     // Size must be positive
		{1, -1, true},    // Size must be positive
		{1, 101, true},   // Size cannot exceed 100
	}

	for _, tt := range tests {
		err := ValidatePage(tt.page, tt.size)
		if tt.wantError {
			assert.Error(t, err)
		} else {
			assert.NoError(t, err)
		}
	}
}

func TestValidateEmail(t *testing.T) {
	tests := []struct {
		email     string
		wantError bool
	}{
		{"test@example.com", false},
		{"user.name@domain.co.uk", false},
		{"user+tag@example.org", false},
		{"", true},                    // Empty email
		{"invalid-email", true},       // No @ symbol
		{"@example.com", true},        // No username
		{"test@", true},               // No domain
		{"test@.com", true},           // Invalid domain
		{"test@example", true},        // No TLD
		{"test..test@example.com", false}, // Double dots - actually valid in regex
	}

	for _, tt := range tests {
		err := ValidateEmail(tt.email)
		if tt.wantError {
			assert.Error(t, err)
		} else {
			assert.NoError(t, err)
		}
	}
}

func TestValidatePhone(t *testing.T) {
	tests := []struct {
		phone     string
		wantError bool
	}{
		{"13812345678", false},
		{"15987654321", false},
		{"18612345678", false},
		{"", true},           // Empty phone
		{"1234567890", true}, // Not 11 digits
		{"138123456789", true}, // Too many digits
		{"abc12345678", true}, // Contains letters
		{"12812345678", true}, // Invalid prefix
	}

	for _, tt := range tests {
		err := ValidatePhone(tt.phone)
		if tt.wantError {
			assert.Error(t, err)
		} else {
			assert.NoError(t, err)
		}
	}
}

func TestValidatePassword(t *testing.T) {
	tests := []struct {
		password  string
		wantError bool
	}{
		{"Password123!", false},
		{"MyPass1@", false},
		{"123456", false},    // Minimum length
		{"", true},           // Empty password
		{"short", true},      // Too short
		{"thispasswordistoolongtobevalid", true}, // Too long
	}

	for _, tt := range tests {
		err := ValidatePassword(tt.password)
		if tt.wantError {
			assert.Error(t, err)
		} else {
			assert.NoError(t, err)
		}
	}
}