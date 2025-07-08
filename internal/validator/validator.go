package validator

import (
	"regexp"
	"unicode"
)

var loginRegex = regexp.MustCompile(`^[a-zA-Z0-9]{8,}$`)

func IsValidLogin(login string) bool {
	return loginRegex.MatchString(login)
}

func IsValidPassword(password string) bool {
	if len(password) < 8 {
		return false
	}

	var hasLower, hasUpper, hasDigit, hasSymbol bool

	for _, r := range password {
		switch {
		case unicode.IsLower(r):
			hasLower = true
		case unicode.IsUpper(r):
			hasUpper = true
		case unicode.IsDigit(r):
			hasDigit = true
		case unicode.IsSymbol(r), unicode.IsPunct(r):
			hasSymbol = true
		}
	}

	return hasLower && hasUpper && hasDigit && hasSymbol
}
