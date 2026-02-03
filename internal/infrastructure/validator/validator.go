package validator

import (
	"fmt"
	"strconv"
	"strings"
	"unicode"

	"github.com/gin-gonic/gin/binding"
	"github.com/go-playground/validator/v10"
	usecasecontract "github.com/mikiasgoitom/Articulate/internal/usecase/contract"
)

// AppValidator implements the usecase.Validator interface.
type AppValidator struct {
	validate *validator.Validate
}

// NewValidator creates a new validator that implements the usecase.Validator interface.
func NewValidator() usecasecontract.IValidator {
	v := validator.New()
	return &AppValidator{validate: v}
}

// ValidateEmail checks if the email format is valid.
func (av *AppValidator) ValidateEmail(email string) error {
	return av.validate.Var(email, "required,email")
}

// ValidatePasswordStrength checks if the password meets the strength requirements.
func (av *AppValidator) ValidatePasswordStrength(password string) error {
	if len(password) < 8 {
		return fmt.Errorf("password must be at least 8 characters long")
	}
	if !containsUppercase(password) {
		return fmt.Errorf("password must contain at least one uppercase letter")
	}
	if !containsLowercase(password) {
		return fmt.Errorf("password must contain at least one lowercase letter")
	}
	if !containsNumber(password) {
		return fmt.Errorf("password must contain at least one number")
	}
	if !containsSpecial(password) {
		return fmt.Errorf("password must contain at least one special character")
	}
	return nil
}

// RegisterCustomValidators registers custom validation functions with the Gin validator.
func RegisterCustomValidators() {
	if v, ok := binding.Validator.Engine().(*validator.Validate); ok {
		v.RegisterValidation("containsuppercase", containsUppercaseFL)
		v.RegisterValidation("containslowercase", containsLowercaseFL)
		v.RegisterValidation("containsdigit", containsNumberFL)
		v.RegisterValidation("containssymbol", containsSpecialFL)
		v.RegisterValidation("min", customMinLength)
		v.RegisterValidation("max", customMaxLength)
	}
}

// containsUppercase checks if the string contains at least one uppercase letter.
func containsUppercase(s string) bool {
	for _, char := range s {
		if unicode.IsUpper(char) {
			return true
		}
	}
	return false
}
func containsUppercaseFL(fl validator.FieldLevel) bool {
	return containsUppercase(fl.Field().String())
}

// containsLowercase checks if the string contains at least one lowercase letter.
func containsLowercase(s string) bool {
	for _, char := range s {
		if unicode.IsLower(char) {
			return true
		}
	}
	return false
}
func containsLowercaseFL(fl validator.FieldLevel) bool {
	return containsLowercase(fl.Field().String())
}

// containsNumber checks if the string contains at least one number.
func containsNumber(s string) bool {
	for _, char := range s {
		if unicode.IsNumber(char) {
			return true
		}
	}
	return false
}
func containsNumberFL(fl validator.FieldLevel) bool {
	return containsNumber(fl.Field().String())
}

// containsSpecial checks if the string contains at least one special character.
func containsSpecial(s string) bool {
	for _, char := range s {
		if strings.ContainsRune("!@#$%^&*()_+-=[]{};:'\\|,.<>/?", char) {
			return true
		}
	}
	return false
}
func containsSpecialFL(fl validator.FieldLevel) bool {
	return containsSpecial(fl.Field().String())
}

// customMinLength
func customMinLength(fl validator.FieldLevel) bool {
	param := fl.Param()
	minLen, err := strconv.Atoi(param)
	if err != nil {
		return false
	}
	return len(fl.Field().String()) >= minLen
}

// customMaxLength
func customMaxLength(fl validator.FieldLevel) bool {
	param := fl.Param()
	maxLen, err := strconv.Atoi(param)
	if err != nil {
		return false
	}
	return len(fl.Field().String()) <= maxLen
}
