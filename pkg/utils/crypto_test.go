package utils

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestGenerateSalt(t *testing.T) {
	// Test default length (16)
	salt := GenerateSalt()
	assert.Equal(t, 16, len(salt))

	// Test uniqueness
	salt2 := GenerateSalt()
	assert.NotEqual(t, salt, salt2)
}

func TestHashPassword(t *testing.T) {
	password := "testpassword"
	salt := GenerateSalt()
	
	hash := HashPassword(password, salt)
	assert.NotEmpty(t, hash)
	assert.NotEqual(t, password, hash)
	
	// Same password and salt should produce same hash
	hash2 := HashPassword(password, salt)
	assert.Equal(t, hash, hash2)
	
	// Different salt should produce different hash
	salt2 := GenerateSalt()
	hash3 := HashPassword(password, salt2)
	assert.NotEqual(t, hash, hash3)
}

func TestVerifyPassword(t *testing.T) {
	password := "testpassword"
	salt := GenerateSalt()
	hash := HashPassword(password, salt)
	
	// Correct password should verify
	assert.True(t, VerifyPassword(password, salt, hash))
	
	// Wrong password should not verify
	assert.False(t, VerifyPassword("wrongpassword", salt, hash))
	
	// Wrong salt should not verify
	wrongSalt := GenerateSalt()
	assert.False(t, VerifyPassword(password, wrongSalt, hash))
}

func TestMD5(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"hello", "5d41402abc4b2a76b9719d911017c592"},
		{"", "d41d8cd98f00b204e9800998ecf8427e"},
		{"test", "098f6bcd4621d373cade4e832627b4f6"},
	}

	for _, tt := range tests {
		result := MD5(tt.input)
		assert.Equal(t, tt.expected, result)
	}
}

func TestSHA1(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"hello", "aaf4c61ddcc5e8a2dabede0f3b482cd9aea9434d"},
		{"", "da39a3ee5e6b4b0d3255bfef95601890afd80709"},
		{"test", "a94a8fe5ccb19ba61c4c0873d391e987982fbbd3"},
	}

	for _, tt := range tests {
		result := SHA1(tt.input)
		assert.Equal(t, tt.expected, result)
	}
}

func TestSHA256(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"hello", "2cf24dba5fb0a30e26e83b2ac5b9e29e1b161e5c1fa7425e73043362938b9824"},
		{"", "e3b0c44298fc1c149afbf4c8996fb92427ae41e4649b934ca495991b7852b855"},
		{"test", "9f86d081884c7d659a2feaa0c55ad015a3bf4f1b2b0b822cd15d6c15b0f00a08"},
	}

	for _, tt := range tests {
		result := SHA256(tt.input)
		assert.Equal(t, tt.expected, result)
	}
}

func TestGenerateRandomString(t *testing.T) {
	// Test different lengths
	lengths := []int{8, 16, 32}
	for _, length := range lengths {
		str := GenerateRandomString(length)
		assert.Equal(t, length, len(str))
		assert.NotEmpty(t, str)
	}

	// Test uniqueness
	str1 := GenerateRandomString(16)
	str2 := GenerateRandomString(16)
	assert.NotEqual(t, str1, str2)
}

func TestGenerateRandomNumber(t *testing.T) {
	// Test different lengths
	lengths := []int{4, 6, 8}
	for _, length := range lengths {
		num := GenerateRandomNumber(length)
		assert.Equal(t, length, len(num))
		assert.NotEmpty(t, num)
		
		// Check if all characters are digits
		for _, char := range num {
			assert.True(t, char >= '0' && char <= '9')
		}
	}

	// Test uniqueness
	num1 := GenerateRandomNumber(8)
	num2 := GenerateRandomNumber(8)
	assert.NotEqual(t, num1, num2)
}

func TestGenerateUUID(t *testing.T) {
	uuid := GenerateUUID()
	assert.NotEmpty(t, uuid)
	
	// UUID format: 8-4-4-4-12
	parts := []int{8, 4, 4, 4, 12}
	uuidParts := []string{}
	start := 0
	for i, length := range parts {
		if i > 0 {
			start++ // Skip the dash
		}
		uuidParts = append(uuidParts, uuid[start:start+length])
		start += length
	}
	
	assert.Equal(t, 5, len(uuidParts))
	for i, part := range uuidParts {
		assert.Equal(t, parts[i], len(part))
	}

	// Test uniqueness
	uuid2 := GenerateUUID()
	assert.NotEqual(t, uuid, uuid2)
}

func TestMaskString(t *testing.T) {
	tests := []struct {
		input    string
		start    int
		end      int
		mask     rune
		expected string
	}{
		{"1234567890", 3, 3, '*', "123****890"},
		{"hello", 1, 1, '*', "h***o"},
		{"ab", 1, 1, '*', "**"},
		{"", 0, 0, '*', ""},
	}

	for _, tt := range tests {
		result := MaskString(tt.input, tt.start, tt.end, tt.mask)
		assert.Equal(t, tt.expected, result)
	}
}

func TestMaskPhone(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"13812345678", "138****5678"},
		{"1234567890", "1234567890"}, // Not 11 digits
		{"", ""},
	}

	for _, tt := range tests {
		result := MaskPhone(tt.input)
		assert.Equal(t, tt.expected, result)
	}
}

func TestMaskEmail(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"test@example.com", "t**t@example.com"},
		{"a@example.com", "*@example.com"},
		{"ab@example.com", "**@example.com"},
		{"invalid-email", "invalid-email"},
		{"", ""},
	}

	for _, tt := range tests {
		result := MaskEmail(tt.input)
		assert.Equal(t, tt.expected, result)
	}
}

func TestMaskIDCard(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"123456789012345678", "123456********5678"},
		{"12345678901234567", "12345678901234567"}, // Not 18 digits
		{"", ""},
	}

	for _, tt := range tests {
		result := MaskIDCard(tt.input)
		assert.Equal(t, tt.expected, result)
	}
}