package utils

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestGetCurrentTime(t *testing.T) {
	now := GetCurrentTime()
	assert.NotZero(t, now)
	assert.True(t, time.Since(now) < time.Second)
}

func TestGetCurrentTimeString(t *testing.T) {
	timeStr := GetCurrentTimeString()
	assert.NotEmpty(t, timeStr)
	
	// Parse the time string to verify format
	_, err := time.Parse(TimeFormat, timeStr)
	assert.NoError(t, err)
}

func TestGetCurrentTimestamp(t *testing.T) {
	timestamp := GetCurrentTimestamp()
	assert.Greater(t, timestamp, int64(0))
	
	// Should be close to current time
	now := time.Now().Unix()
	assert.True(t, timestamp-now <= 1)
}

func TestGetCurrentTimestampMilli(t *testing.T) {
	timestamp := GetCurrentTimestampMilli()
	assert.Greater(t, timestamp, int64(0))
	
	// Should be close to current time
	now := time.Now().UnixMilli()
	assert.True(t, timestamp-now <= 1000)
}

func TestParseTime(t *testing.T) {
	tests := []struct {
		input     string
		wantError bool
	}{
		{"2023-01-01 12:00:00", false},
		{"2023-12-31 23:59:59", false},
		{"invalid", true},
		{"", true},
		{"2023-01-01", true}, // Wrong format
	}

	for _, tt := range tests {
		result, err := ParseTime(tt.input)
		if tt.wantError {
			assert.Error(t, err)
		} else {
			assert.NoError(t, err)
			assert.NotZero(t, result)
		}
	}
}

func TestParseDate(t *testing.T) {
	tests := []struct {
		input     string
		wantError bool
	}{
		{"2023-01-01", false},
		{"2023-12-31", false},
		{"invalid", true},
		{"", true},
		{"2023-01-01 12:00:00", true}, // Wrong format
	}

	for _, tt := range tests {
		result, err := ParseDate(tt.input)
		if tt.wantError {
			assert.Error(t, err)
		} else {
			assert.NoError(t, err)
			assert.NotZero(t, result)
		}
	}
}

func TestFormatTime(t *testing.T) {
	testTime := time.Date(2023, 1, 1, 12, 0, 0, 0, time.UTC)
	result := FormatTime(testTime)
	assert.Equal(t, "2023-01-01 12:00:00", result)
}

func TestFormatDate(t *testing.T) {
	testTime := time.Date(2023, 1, 1, 12, 0, 0, 0, time.UTC)
	result := FormatDate(testTime)
	assert.Equal(t, "2023-01-01", result)
}

func TestTimestampToTime(t *testing.T) {
	timestamp := int64(1672574400) // 2023-01-01 12:00:00 UTC
	result := TimestampToTime(timestamp)
	expected := time.Unix(timestamp, 0)
	assert.Equal(t, expected, result)
}

func TestTimestampMilliToTime(t *testing.T) {
	timestamp := int64(1672574400000) // 2023-01-01 12:00:00 UTC in milliseconds
	result := TimestampMilliToTime(timestamp)
	expected := time.UnixMilli(timestamp)
	assert.Equal(t, expected, result)
}

func TestTimeToTimestamp(t *testing.T) {
	testTime := time.Date(2023, 1, 1, 12, 0, 0, 0, time.UTC)
	result := TimeToTimestamp(testTime)
	expected := testTime.Unix()
	assert.Equal(t, expected, result)
}

func TestTimeToTimestampMilli(t *testing.T) {
	testTime := time.Date(2023, 1, 1, 12, 0, 0, 0, time.UTC)
	result := TimeToTimestampMilli(testTime)
	expected := testTime.UnixMilli()
	assert.Equal(t, expected, result)
}

func TestIsTimeInRange(t *testing.T) {
	start := time.Date(2023, 1, 1, 10, 0, 0, 0, time.UTC)
	end := time.Date(2023, 1, 1, 14, 0, 0, 0, time.UTC)
	
	tests := []struct {
		t        time.Time
		expected bool
	}{
		{time.Date(2023, 1, 1, 12, 0, 0, 0, time.UTC), true},  // In range
		{time.Date(2023, 1, 1, 10, 0, 0, 0, time.UTC), true},  // Start time
		{time.Date(2023, 1, 1, 14, 0, 0, 0, time.UTC), true},  // End time
		{time.Date(2023, 1, 1, 9, 0, 0, 0, time.UTC), false},  // Before start
		{time.Date(2023, 1, 1, 15, 0, 0, 0, time.UTC), false}, // After end
	}

	for _, tt := range tests {
		result := IsTimeInRange(tt.t, start, end)
		assert.Equal(t, tt.expected, result)
	}
}

func TestGetStartOfDay(t *testing.T) {
	testTime := time.Date(2023, 1, 1, 15, 30, 45, 0, time.UTC)
	result := GetStartOfDay(testTime)
	expected := time.Date(2023, 1, 1, 0, 0, 0, 0, time.UTC)
	assert.Equal(t, expected, result)
}

func TestGetEndOfDay(t *testing.T) {
	testTime := time.Date(2023, 1, 1, 15, 30, 45, 0, time.UTC)
	result := GetEndOfDay(testTime)
	expected := time.Date(2023, 1, 1, 23, 59, 59, 999999999, time.UTC)
	assert.Equal(t, expected, result)
}

func TestGetStartOfWeek(t *testing.T) {
	// Test with a Wednesday (2023-01-04)
	testTime := time.Date(2023, 1, 4, 15, 30, 45, 0, time.UTC)
	result := GetStartOfWeek(testTime)
	
	// Should be the previous Sunday (2023-01-01)
	expected := time.Date(2023, 1, 1, 0, 0, 0, 0, time.UTC)
	assert.Equal(t, expected, result)
}

func TestGetEndOfWeek(t *testing.T) {
	// Test with a Wednesday (2023-01-04)
	testTime := time.Date(2023, 1, 4, 15, 30, 45, 0, time.UTC)
	result := GetEndOfWeek(testTime)
	
	// Should be the next Saturday (2023-01-07)
	expected := time.Date(2023, 1, 7, 23, 59, 59, 999999999, time.UTC)
	assert.Equal(t, expected, result)
}

func TestGetStartOfMonth(t *testing.T) {
	testTime := time.Date(2023, 1, 15, 15, 30, 45, 0, time.UTC)
	result := GetStartOfMonth(testTime)
	expected := time.Date(2023, 1, 1, 0, 0, 0, 0, time.UTC)
	assert.Equal(t, expected, result)
}

func TestGetEndOfMonth(t *testing.T) {
	testTime := time.Date(2023, 1, 15, 15, 30, 45, 0, time.UTC)
	result := GetEndOfMonth(testTime)
	expected := time.Date(2023, 1, 31, 23, 59, 59, 999999999, time.UTC)
	assert.Equal(t, expected, result)
}

func TestAddDays(t *testing.T) {
	testTime := time.Date(2023, 1, 1, 12, 0, 0, 0, time.UTC)
	result := AddDays(testTime, 5)
	expected := time.Date(2023, 1, 6, 12, 0, 0, 0, time.UTC)
	assert.Equal(t, expected, result)
}

func TestAddHours(t *testing.T) {
	testTime := time.Date(2023, 1, 1, 12, 0, 0, 0, time.UTC)
	result := AddHours(testTime, 5)
	expected := time.Date(2023, 1, 1, 17, 0, 0, 0, time.UTC)
	assert.Equal(t, expected, result)
}

func TestAddMinutes(t *testing.T) {
	testTime := time.Date(2023, 1, 1, 12, 0, 0, 0, time.UTC)
	result := AddMinutes(testTime, 30)
	expected := time.Date(2023, 1, 1, 12, 30, 0, 0, time.UTC)
	assert.Equal(t, expected, result)
}

func TestAddSeconds(t *testing.T) {
	testTime := time.Date(2023, 1, 1, 12, 0, 0, 0, time.UTC)
	result := AddSeconds(testTime, 45)
	expected := time.Date(2023, 1, 1, 12, 0, 45, 0, time.UTC)
	assert.Equal(t, expected, result)
}

func TestDiffDays(t *testing.T) {
	t1 := time.Date(2023, 1, 1, 0, 0, 0, 0, time.UTC)
	t2 := time.Date(2023, 1, 6, 0, 0, 0, 0, time.UTC)
	result := DiffDays(t1, t2)
	assert.Equal(t, -5, result) // t1 is earlier than t2, so result is negative
}

func TestDiffHours(t *testing.T) {
	t1 := time.Date(2023, 1, 1, 10, 0, 0, 0, time.UTC)
	t2 := time.Date(2023, 1, 1, 15, 0, 0, 0, time.UTC)
	result := DiffHours(t1, t2)
	assert.Equal(t, -5, result) // t1 is earlier than t2, so result is negative
}

func TestDiffMinutes(t *testing.T) {
	t1 := time.Date(2023, 1, 1, 12, 0, 0, 0, time.UTC)
	t2 := time.Date(2023, 1, 1, 12, 30, 0, 0, time.UTC)
	result := DiffMinutes(t1, t2)
	assert.Equal(t, -30, result) // t1 is earlier than t2, so result is negative
}

func TestDiffSeconds(t *testing.T) {
	t1 := time.Date(2023, 1, 1, 12, 0, 0, 0, time.UTC)
	t2 := time.Date(2023, 1, 1, 12, 0, 45, 0, time.UTC)
	result := DiffSeconds(t1, t2)
	assert.Equal(t, -45, result) // t1 is earlier than t2, so result is negative
}

func TestIsToday(t *testing.T) {
	now := time.Now()
	assert.True(t, IsToday(now))
	
	yesterday := now.AddDate(0, 0, -1)
	assert.False(t, IsToday(yesterday))
}

func TestIsYesterday(t *testing.T) {
	yesterday := time.Now().AddDate(0, 0, -1)
	assert.True(t, IsYesterday(yesterday))
	
	now := time.Now()
	assert.False(t, IsYesterday(now))
}

func TestIsTomorrow(t *testing.T) {
	tomorrow := time.Now().AddDate(0, 0, 1)
	assert.True(t, IsTomorrow(tomorrow))
	
	now := time.Now()
	assert.False(t, IsTomorrow(now))
}

func TestSleep(t *testing.T) {
	start := time.Now()
	Sleep(100 * time.Millisecond)
	elapsed := time.Since(start)
	assert.True(t, elapsed >= 100*time.Millisecond)
}

func TestSleepSeconds(t *testing.T) {
	start := time.Now()
	SleepSeconds(1)
	elapsed := time.Since(start)
	assert.True(t, elapsed >= time.Second)
}

func TestSleepMilliseconds(t *testing.T) {
	start := time.Now()
	SleepMilliseconds(100)
	elapsed := time.Since(start)
	assert.True(t, elapsed >= 100*time.Millisecond)
}