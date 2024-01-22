package validator

import (
	"regexp"
)

var (
	EmailRX = regexp.MustCompile(`^[a-z0-9._%+\-]+@[a-z0-9.\-]+\.[a-z]{2,16}$`)
)

type Validator struct {
	Errors map[string]string
}

// New returns a new instance of Validator.
func New() *Validator {
	return &Validator{Errors: make(map[string]string)}
}

// Valid returns true if the errors map doesn't contain any entries, indicating
func (v *Validator) Valid() bool {
	return len(v.Errors) == 0
}

// AddError adds an error message to the map (so long as no entry already exists for the given key).
func (v *Validator) AddError(key, message string) {
	if _, ok := v.Errors[key]; !ok {
		v.Errors[key] = message
	}
}

// Check checks if the form field is empty and adds an error message to the errors map if it is.
func (v *Validator) Check(ok bool, field, value string) {
	if !ok {
		v.AddError(field, value)
	}
}

// In checks if a string value is in a list of strings.
func In(value string, list ...string) bool {
	for i := range list {
		if value == list[i] {
			return true
		}
	}
	return false
}

// Matches checks if a string value matches a regular expression.
func (v *Validator) Matches(field string, value string, rx *regexp.Regexp) {
	if !rx.MatchString(value) {
		v.AddError(field, "This field is invalid")
	}
}

// Unique checks if a string value is contained in a slice of strings.
func (v *Validator) Unique(values []string) bool {
	uniqueValues := make(map[string]bool)

	for _, value := range values {
		uniqueValues[value] = true
	}

	return len(uniqueValues) == len(values)
}
