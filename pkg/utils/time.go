package utils

import (
	"time"
)

const (
	// Time format constants
	TimeFormat     = "2006-01-02 15:04:05"
	DateFormat     = "2006-01-02"
	TimeOnlyFormat = "15:04:05"
	RFC3339Format  = time.RFC3339
)

// GetCurrentTime get current time
func GetCurrentTime() time.Time {
	return time.Now()
}

// GetCurrentTimeString get current time string
func GetCurrentTimeString() string {
	return time.Now().Format(TimeFormat)
}

// GetCurrentTimestamp get current timestamp (seconds)
func GetCurrentTimestamp() int64 {
	return time.Now().Unix()
}

// GetCurrentTimestampMilli get current timestamp (milliseconds)
func GetCurrentTimestampMilli() int64 {
	return time.Now().UnixMilli()
}

// ParseTime parse time string
func ParseTime(timeStr string) (time.Time, error) {
	return time.Parse(TimeFormat, timeStr)
}

// ParseDate parse date string
func ParseDate(dateStr string) (time.Time, error) {
	return time.Parse(DateFormat, dateStr)
}

// FormatTime format time
func FormatTime(t time.Time) string {
	return t.Format(TimeFormat)
}

// FormatDate format date
func FormatDate(t time.Time) string {
	return t.Format(DateFormat)
}

// TimestampToTime timestamp to time
func TimestampToTime(timestamp int64) time.Time {
	return time.Unix(timestamp, 0)
}

// TimestampMilliToTime millisecond timestamp to time
func TimestampMilliToTime(timestamp int64) time.Time {
	return time.UnixMilli(timestamp)
}

// TimeToTimestamp time to timestamp
func TimeToTimestamp(t time.Time) int64 {
	return t.Unix()
}

// TimeToTimestampMilli time to millisecond timestamp
func TimeToTimestampMilli(t time.Time) int64 {
	return t.UnixMilli()
}

// IsTimeInRange check if time is within specified range
func IsTimeInRange(t, start, end time.Time) bool {
	return (t.Equal(start) || t.After(start)) && (t.Equal(end) || t.Before(end))
}

// GetStartOfDay get start time of the day
func GetStartOfDay(t time.Time) time.Time {
	return time.Date(t.Year(), t.Month(), t.Day(), 0, 0, 0, 0, t.Location())
}

// GetEndOfDay get end time of the day
func GetEndOfDay(t time.Time) time.Time {
	return time.Date(t.Year(), t.Month(), t.Day(), 23, 59, 59, 999999999, t.Location())
}

// GetStartOfWeek get start time of the week (Monday)
func GetStartOfWeek(t time.Time) time.Time {
	weekday := int(t.Weekday())
	return GetStartOfDay(t.AddDate(0, 0, -weekday))
}

// GetEndOfWeek get end time of the week (Sunday)
func GetEndOfWeek(t time.Time) time.Time {
	return GetEndOfDay(GetStartOfWeek(t).AddDate(0, 0, 6))
}

// GetStartOfMonth get start time of the month
func GetStartOfMonth(t time.Time) time.Time {
	return time.Date(t.Year(), t.Month(), 1, 0, 0, 0, 0, t.Location())
}

// GetEndOfMonth get end time of the month
func GetEndOfMonth(t time.Time) time.Time {
	return GetStartOfMonth(t).AddDate(0, 1, 0).Add(-time.Nanosecond)
}

// AddDays add days
func AddDays(t time.Time, days int) time.Time {
	return t.AddDate(0, 0, days)
}

// AddHours add hours
func AddHours(t time.Time, hours int) time.Time {
	return t.Add(time.Duration(hours) * time.Hour)
}

// AddMinutes add minutes
func AddMinutes(t time.Time, minutes int) time.Time {
	return t.Add(time.Duration(minutes) * time.Minute)
}

// AddSeconds add seconds
func AddSeconds(t time.Time, seconds int) time.Time {
	return t.Add(time.Duration(seconds) * time.Second)
}

// DiffDays calculate difference in days between two times
func DiffDays(t1, t2 time.Time) int {
	return int(t1.Sub(t2).Hours() / 24)
}

// DiffHours calculate difference in hours between two times
func DiffHours(t1, t2 time.Time) int {
	return int(t1.Sub(t2).Hours())
}

// DiffMinutes calculate difference in minutes between two times
func DiffMinutes(t1, t2 time.Time) int {
	return int(t1.Sub(t2).Minutes())
}

// DiffSeconds calculate difference in seconds between two times
func DiffSeconds(t1, t2 time.Time) int {
	return int(t1.Sub(t2).Seconds())
}

// IsToday check if it's today
func IsToday(t time.Time) bool {
	now := time.Now()
	return t.Year() == now.Year() && t.Month() == now.Month() && t.Day() == now.Day()
}

// IsYesterday check if it's yesterday
func IsYesterday(t time.Time) bool {
	yesterday := time.Now().AddDate(0, 0, -1)
	return t.Year() == yesterday.Year() && t.Month() == yesterday.Month() && t.Day() == yesterday.Day()
}

// IsTomorrow check if it's tomorrow
func IsTomorrow(t time.Time) bool {
	tomorrow := time.Now().AddDate(0, 0, 1)
	return t.Year() == tomorrow.Year() && t.Month() == tomorrow.Month() && t.Day() == tomorrow.Day()
}

// Sleep sleep for specified duration
func Sleep(duration time.Duration) {
	time.Sleep(duration)
}

// SleepSeconds sleep for specified seconds
func SleepSeconds(seconds int) {
	time.Sleep(time.Duration(seconds) * time.Second)
}

// SleepMilliseconds sleep for specified milliseconds
func SleepMilliseconds(milliseconds int) {
	time.Sleep(time.Duration(milliseconds) * time.Millisecond)
}