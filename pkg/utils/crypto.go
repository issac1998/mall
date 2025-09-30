package utils

import (
	"crypto/md5"
	"crypto/rand"
	"crypto/sha1"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"math/big"
	"strings"
)

// MD5 calculate MD5 hash value
func MD5(data string) string {
	hash := md5.Sum([]byte(data))
	return hex.EncodeToString(hash[:])
}

// SHA1 calculate SHA1 hash value
func SHA1(data string) string {
	hash := sha1.Sum([]byte(data))
	return hex.EncodeToString(hash[:])
}

// SHA256 calculate SHA256 hash value
func SHA256(data string) string {
	hash := sha256.Sum256([]byte(data))
	return hex.EncodeToString(hash[:])
}

// GenerateRandomString generate random string
func GenerateRandomString(length int) string {
	const charset = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789"
	result := make([]byte, length)
	for i := range result {
		num, _ := rand.Int(rand.Reader, big.NewInt(int64(len(charset))))
		result[i] = charset[num.Int64()]
	}
	return string(result)
}

// GenerateRandomNumber generate random number string
func GenerateRandomNumber(length int) string {
	const charset = "0123456789"
	result := make([]byte, length)
	for i := range result {
		num, _ := rand.Int(rand.Reader, big.NewInt(int64(len(charset))))
		result[i] = charset[num.Int64()]
	}
	return string(result)
}

// GenerateUUID generate simple UUID (not standard UUID)
func GenerateUUID() string {
	return fmt.Sprintf("%s-%s-%s-%s-%s",
		GenerateRandomString(8),
		GenerateRandomString(4),
		GenerateRandomString(4),
		GenerateRandomString(4),
		GenerateRandomString(12))
}

// HashPassword password hash (simple implementation, recommend using bcrypt in production)
func HashPassword(password, salt string) string {
	return SHA256(password + salt)
}

// GenerateSalt generate salt value
func GenerateSalt() string {
	return GenerateRandomString(16)
}

// VerifyPassword verify password
func VerifyPassword(password, salt, hashedPassword string) bool {
	return HashPassword(password, salt) == hashedPassword
}

// MaskString mask string (for sensitive information display)
func MaskString(str string, start, end int, mask rune) string {
	if len(str) <= start+end {
		return strings.Repeat(string(mask), len(str))
	}

	runes := []rune(str)
	for i := start; i < len(runes)-end; i++ {
		runes[i] = mask
	}
	return string(runes)
}

// MaskPhone mask phone number
func MaskPhone(phone string) string {
	if len(phone) != 11 {
		return phone
	}
	return MaskString(phone, 3, 4, '*')
}

// MaskEmail mask email
func MaskEmail(email string) string {
	parts := strings.Split(email, "@")
	if len(parts) != 2 {
		return email
	}

	username := parts[0]
	domain := parts[1]

	if len(username) <= 2 {
		return strings.Repeat("*", len(username)) + "@" + domain
	}

	maskedUsername := MaskString(username, 1, 1, '*')
	return maskedUsername + "@" + domain
}

// MaskIDCard mask ID card number
func MaskIDCard(idCard string) string {
	if len(idCard) != 18 {
		return idCard
	}
	return MaskString(idCard, 6, 4, '*')
}