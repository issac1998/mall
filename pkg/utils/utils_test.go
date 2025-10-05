package utils

import (
	"errors"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
)

// TestResponse test response utilities
func TestResponse(t *testing.T) {
	gin.SetMode(gin.TestMode)

	t.Run("Success", func(t *testing.T) {
		// Test response structure
		resp := Response{
			Code:      0,
			Message:   "success",
			Data:      "test data",
			Timestamp: 1234567890,
		}
		assert.Equal(t, 0, resp.Code)
		assert.Equal(t, "success", resp.Message)
		assert.Equal(t, "test data", resp.Data)
		assert.Equal(t, int64(1234567890), resp.Timestamp)
	})

	t.Run("Error", func(t *testing.T) {
		resp := Response{
			Code:      1,
			Message:   "error",
			Timestamp: 1234567890,
		}
		assert.Equal(t, 1, resp.Code)
		assert.Equal(t, "error", resp.Message)
		assert.Equal(t, int64(1234567890), resp.Timestamp)
	})

	t.Run("PageResponse", func(t *testing.T) {
		pageResp := PageResponse{
			List:  []string{"item1", "item2"},
			Total: 100,
			Page:  1,
			Size:  10,
		}
		assert.Equal(t, []string{"item1", "item2"}, pageResp.List)
		assert.Equal(t, int64(100), pageResp.Total)
		assert.Equal(t, 1, pageResp.Page)
		assert.Equal(t, 10, pageResp.Size)
	})
}

// TestAppError test application error
func TestAppError(t *testing.T) {
	t.Run("NewError", func(t *testing.T) {
		err := NewError(CodeInvalidParam, "test error")
		assert.Equal(t, CodeInvalidParam, err.Code)
		assert.Equal(t, "test error", err.Message)
		assert.Nil(t, err.Err)
		assert.Equal(t, "code: 1001, message: test error", err.Error())
	})

	t.Run("NewErrorWithErr", func(t *testing.T) {
		originalErr := errors.New("original error")
		err := NewErrorWithErr(CodeDatabaseError, "database error", originalErr)
		assert.Equal(t, CodeDatabaseError, err.Code)
		assert.Equal(t, "database error", err.Message)
		assert.Equal(t, originalErr, err.Err)
		assert.Contains(t, err.Error(), "original error")
	})

	t.Run("WrapError", func(t *testing.T) {
		originalErr := errors.New("original error")
		err := WrapError(originalErr, CodeServiceError, "service error")
		assert.Equal(t, CodeServiceError, err.Code)
		assert.Equal(t, "service error", err.Message)
		assert.Equal(t, originalErr, err.Err)
	})

	t.Run("IsAppError", func(t *testing.T) {
		appErr := NewError(CodeInvalidParam, "test error")
		normalErr := errors.New("normal error")

		// Test application error
		if err, ok := IsAppError(appErr); ok {
			assert.Equal(t, CodeInvalidParam, err.Code)
		} else {
			t.Error("Expected app error")
		}

		// Test normal error
		if _, ok := IsAppError(normalErr); ok {
			t.Error("Expected not app error")
		}
	})

	t.Run("GetErrorCode", func(t *testing.T) {
		appErr := NewError(CodeInvalidParam, "test error")
		normalErr := errors.New("normal error")

		assert.Equal(t, CodeInvalidParam, GetErrorCode(appErr))
		assert.Equal(t, CodeInternalError, GetErrorCode(normalErr))
	})

	t.Run("GetErrorMessage", func(t *testing.T) {
		appErr := NewError(CodeInvalidParam, "test error")
		normalErr := errors.New("normal error")

		assert.Equal(t, "test error", GetErrorMessage(appErr))
		assert.Equal(t, "normal error", GetErrorMessage(normalErr))
	})
}

// TestValidator test validator
func TestValidator(t *testing.T) {
	t.Run("ValidateID", func(t *testing.T) {
		// Test valid ID
		id, err := ValidateID("123")
		assert.NoError(t, err)
		assert.Equal(t, int64(123), id)

		// Test empty ID
		_, err = ValidateID("")
		assert.Error(t, err)
		assert.Equal(t, CodeInvalidParam, GetErrorCode(err))

		// Test invalid format
		_, err = ValidateID("abc")
		assert.Error(t, err)
		assert.Equal(t, CodeInvalidParam, GetErrorCode(err))

		// Test negative number
		_, err = ValidateID("-1")
		assert.Error(t, err)
		assert.Equal(t, CodeInvalidParam, GetErrorCode(err))

		// Test zero
		_, err = ValidateID("0")
		assert.Error(t, err)
		assert.Equal(t, CodeInvalidParam, GetErrorCode(err))
	})

	t.Run("ValidatePage", func(t *testing.T) {
		// Test valid pagination
		err := ValidatePage(1, 10)
		assert.NoError(t, err)

		// Test invalid page number
		err = ValidatePage(0, 10)
		assert.Error(t, err)
		assert.Equal(t, CodeInvalidParam, GetErrorCode(err))

		err = ValidatePage(-1, 10)
		assert.Error(t, err)

		// Test invalid page size
		err = ValidatePage(1, 0)
		assert.Error(t, err)

		err = ValidatePage(1, -1)
		assert.Error(t, err)

		err = ValidatePage(1, 101)
		assert.Error(t, err)
	})

	t.Run("CamelToSnake", func(t *testing.T) {
		assert.Equal(t, "user_id", camelToSnake("UserID"))
		assert.Equal(t, "user_name", camelToSnake("UserName"))
		assert.Equal(t, "id", camelToSnake("ID"))
		assert.Equal(t, "name", camelToSnake("Name"))
		assert.Equal(t, "apikey", camelToSnake("APIKey"))
		assert.Equal(t, "httpurl", camelToSnake("HTTPURL"))
	})
}

// TestTimeUtils test time utilities
func TestTimeUtils(t *testing.T) {
	t.Run("GetCurrentTime", func(t *testing.T) {
		now := GetCurrentTime()
		assert.True(t, time.Since(now) < time.Second)
	})

	t.Run("GetCurrentTimestamp", func(t *testing.T) {
		timestamp := GetCurrentTimestamp()
		assert.True(t, timestamp > 0)
		assert.True(t, time.Now().Unix()-timestamp < 1)
	})

	t.Run("ParseTime", func(t *testing.T) {
		timeStr := "2023-01-01 12:00:00"
		parsedTime, err := ParseTime(timeStr)
		assert.NoError(t, err)
		assert.Equal(t, 2023, parsedTime.Year())
		assert.Equal(t, time.January, parsedTime.Month())
		assert.Equal(t, 1, parsedTime.Day())
		assert.Equal(t, 12, parsedTime.Hour())
	})

	t.Run("FormatTime", func(t *testing.T) {
		testTime := time.Date(2023, 1, 1, 12, 0, 0, 0, time.UTC)
		formatted := FormatTime(testTime)
		assert.Equal(t, "2023-01-01 12:00:00", formatted)
	})

	t.Run("TimestampConversion", func(t *testing.T) {
		now := time.Now()
		timestamp := TimeToTimestamp(now)
		convertedTime := TimestampToTime(timestamp)
		assert.Equal(t, now.Unix(), convertedTime.Unix())
	})

	t.Run("IsTimeInRange", func(t *testing.T) {
		start := time.Date(2023, 1, 1, 0, 0, 0, 0, time.UTC)
		end := time.Date(2023, 1, 2, 0, 0, 0, 0, time.UTC)
		middle := time.Date(2023, 1, 1, 12, 0, 0, 0, time.UTC)
		before := time.Date(2022, 12, 31, 0, 0, 0, 0, time.UTC)
		after := time.Date(2023, 1, 3, 0, 0, 0, 0, time.UTC)

		assert.True(t, IsTimeInRange(middle, start, end))
		assert.True(t, IsTimeInRange(start, start, end))
		assert.True(t, IsTimeInRange(end, start, end))
		assert.False(t, IsTimeInRange(before, start, end))
		assert.False(t, IsTimeInRange(after, start, end))
	})

	t.Run("GetStartEndOfDay", func(t *testing.T) {
		testTime := time.Date(2023, 1, 1, 12, 30, 45, 0, time.UTC)
		startOfDay := GetStartOfDay(testTime)
		endOfDay := GetEndOfDay(testTime)

		assert.Equal(t, 0, startOfDay.Hour())
		assert.Equal(t, 0, startOfDay.Minute())
		assert.Equal(t, 0, startOfDay.Second())

		assert.Equal(t, 23, endOfDay.Hour())
		assert.Equal(t, 59, endOfDay.Minute())
		assert.Equal(t, 59, endOfDay.Second())
	})

	t.Run("DiffCalculations", func(t *testing.T) {
		t1 := time.Date(2023, 1, 2, 12, 0, 0, 0, time.UTC)
		t2 := time.Date(2023, 1, 1, 10, 0, 0, 0, time.UTC)

		assert.Equal(t, 1, DiffDays(t1, t2))
		assert.Equal(t, 26, DiffHours(t1, t2))
		assert.Equal(t, 1560, DiffMinutes(t1, t2))
	})

	t.Run("IsToday", func(t *testing.T) {
		now := time.Now()
		assert.True(t, IsToday(now))

		yesterday := now.AddDate(0, 0, -1)
		assert.False(t, IsToday(yesterday))
	})
}

// TestCryptoUtils test crypto utilities
func TestCryptoUtils(t *testing.T) {
	t.Run("HashFunctions", func(t *testing.T) {
		data := "test data"

		md5Hash := MD5(data)
		assert.Len(t, md5Hash, 32)
		assert.Equal(t, MD5(data), md5Hash) // Same input should produce same output

		sha1Hash := SHA1(data)
		assert.Len(t, sha1Hash, 40)
		assert.Equal(t, SHA1(data), sha1Hash)

		sha256Hash := SHA256(data)
		assert.Len(t, sha256Hash, 64)
		assert.Equal(t, SHA256(data), sha256Hash)
	})

	t.Run("GenerateRandomString", func(t *testing.T) {
		str1 := GenerateRandomString(10)
		str2 := GenerateRandomString(10)

		assert.Len(t, str1, 10)
		assert.Len(t, str2, 10)
		assert.NotEqual(t, str1, str2) // Random strings should be different
	})

	t.Run("GenerateRandomNumber", func(t *testing.T) {
		num := GenerateRandomNumber(6)
		assert.Len(t, num, 6)

		// Verify it contains only digits
		for _, char := range num {
			assert.True(t, char >= '0' && char <= '9')
		}
	})

	t.Run("GenerateUUID", func(t *testing.T) {
		uuid1 := GenerateUUID()
		uuid2 := GenerateUUID()

		assert.NotEqual(t, uuid1, uuid2)
		assert.Contains(t, uuid1, "-")
		assert.Len(t, uuid1, 36) // 8-4-4-4-12 + 4 hyphens
	})

	t.Run("PasswordHashing", func(t *testing.T) {
		password := "testpassword"
		salt := GenerateSalt()

		hashedPassword := HashPassword(password, salt)
		assert.NotEmpty(t, hashedPassword)
		assert.NotEqual(t, password, hashedPassword)

		// Verify password
		assert.True(t, VerifyPassword(password, salt, hashedPassword))
		assert.False(t, VerifyPassword("wrongpassword", salt, hashedPassword))
	})

	t.Run("MaskString", func(t *testing.T) {
		str := "1234567890"
		masked := MaskString(str, 2, 2, '*')
		assert.Equal(t, "12******90", masked)

		// Test short string
		shortStr := "123"
		maskedShort := MaskString(shortStr, 2, 2, '*')
		assert.Equal(t, "***", maskedShort)
	})

	t.Run("MaskPhone", func(t *testing.T) {
		phone := "13812345678"
		masked := MaskPhone(phone)
		assert.Equal(t, "138****5678", masked)

		// Test invalid phone number
		invalidPhone := "123"
		assert.Equal(t, "123", MaskPhone(invalidPhone))
	})

	t.Run("MaskEmail", func(t *testing.T) {
		email := "test@example.com"
		masked := MaskEmail(email)
		assert.Equal(t, "t**t@example.com", masked)

		// Test short username
		shortEmail := "ab@example.com"
		maskedShort := MaskEmail(shortEmail)
		assert.Equal(t, "**@example.com", maskedShort)

		// Test invalid email
		invalidEmail := "invalid"
		assert.Equal(t, "invalid", MaskEmail(invalidEmail))
	})

	t.Run("MaskIDCard", func(t *testing.T) {
		idCard := "123456789012345678"
		masked := MaskIDCard(idCard)
		assert.Equal(t, "123456********5678", masked)

		// Test invalid ID card number
		invalidID := "123"
		assert.Equal(t, "123", MaskIDCard(invalidID))
	})
}