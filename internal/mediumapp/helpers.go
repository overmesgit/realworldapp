package mediumapp

import (
	"github.com/go-playground/validator/v10"
	"strings"
)

func GetErrorMap(validationError error) map[string][]string {
	var errMap map[string][]string
	if validationError != nil {
		errMap = make(map[string][]string)
		for _, err := range validationError.(validator.ValidationErrors) {
			lowerField := strings.ToLower(err.Field())
			errMap[lowerField] = append(errMap[lowerField], err.Tag())
		}
	}
	return errMap
}
