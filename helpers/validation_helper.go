package helpers

import (
	// "fmt"
	"regexp"
	"github.com/google/uuid"
)

func IsValidEmail(email string) bool {
	if len(email) < 3 || len(email) > 254 {
		return false
	}
	// create regex for email validation
	regex := `^[a-zA-Z0-9._%+-]+@[a-zA-Z0-9.-]+\.[a-zA-Z]{2,}$`
	re := regexp.MustCompile(regex)
	if(!re.MatchString(email)) {
		return false
	}

	return true
}

func IsValidPhoneNumber(phone string) bool {
	if len(phone) < 10 || len(phone) > 15 {
		return false
	}
	// create regex for phone number validation (only digits, optional + at the start)
	regex := `^\+?[0-9]{10,15}$`
	re := regexp.MustCompile(regex)
	if(!re.MatchString(phone)) {
		return false
	}
	return true
}

// validate password strength (min 8 characters, at least one number, one uppercase letter, one lowercase letter, one special character)
func IsStrongPassword(password string) bool {
	if len(password) < 8 {
		return false
	}
	// create regex for password strength validation
	var (
		hasUpper   = regexp.MustCompile(`[A-Z]`)
		hasLower   = regexp.MustCompile(`[a-z]`)
		hasNumber  = regexp.MustCompile(`[0-9]`)
		hasSpecial = regexp.MustCompile(`[!@#~$%^&*()+|_.,<>?/{}\-]`)
	)

	if !hasUpper.MatchString(password) {
		return false
	}
	if !hasLower.MatchString(password) {
		return false
	}
	if !hasNumber.MatchString(password) {
		return false
	}
	if !hasSpecial.MatchString(password) {
		return false
	}

	return true
}

// ParseUUID parses a string to UUID
func ParseUUID(uuidStr string) uuid.UUID {
	parsedUUID, err := uuid.Parse(uuidStr)
	if err != nil {
		return uuid.Nil
	}
	return parsedUUID
}

// ParseUUIDPtr parses a *string to *uuid.UUID, returning nil if input is nil or invalid
func ParseUUIDPtr(uuidStr *string) *uuid.UUID {
	if uuidStr == nil || *uuidStr == "" {
		return nil
	}
	parsedUUID, err := uuid.Parse(*uuidStr)
	if err != nil {
		return nil
	}
	return &parsedUUID
}
