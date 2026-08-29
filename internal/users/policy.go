package users

import (
	"fmt"
	"strings"
	"unicode"

	"github.com/raven-clown/idpforge/internal/config"
)

// ValidatePassword checks a self-chosen password against the configured
// complexity policy. It does not apply to DefaultPassword, which is an
// operator-supplied config value, not user input.
func ValidatePassword(password string, policy config.PasswordPolicyConfig) error {
	if len(password) < policy.MinLength {
		return fmt.Errorf("password must be at least %d characters", policy.MinLength)
	}

	var hasUpper, hasLower, hasNumber, hasSpecial bool
	for _, r := range password {
		switch {
		case unicode.IsUpper(r):
			hasUpper = true
		case unicode.IsLower(r):
			hasLower = true
		case unicode.IsNumber(r):
			hasNumber = true
		case strings.ContainsRune("!@#$%^&*()_+-=[]{}|;:'\",.<>/?`~\\", r):
			hasSpecial = true
		}
	}

	if policy.RequireUppercase && !hasUpper {
		return fmt.Errorf("password must contain at least one uppercase letter")
	}
	if policy.RequireLowercase && !hasLower {
		return fmt.Errorf("password must contain at least one lowercase letter")
	}
	if policy.RequireNumber && !hasNumber {
		return fmt.Errorf("password must contain at least one number")
	}
	if policy.RequireSpecial && !hasSpecial {
		return fmt.Errorf("password must contain at least one special character")
	}
	return nil
}
