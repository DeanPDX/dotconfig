package dotconfig

import (
	"fmt"
	"strings"
)

// Simple wrapper for multiple errors. This definition matches the uderlying
// type that [errors.Join] introduced in go 1.20:
//   - https://cs.opensource.google/go/go/+/refs/tags/go1.23.1:src/errors/join.go;l=40
type joinError struct {
	errs []error
}

// HasErrors will return true if any of the errors in underlying
// errs slice are non-nil.
func (je *joinError) HasErrors() bool {
	for _, err := range je.errs {
		if err != nil {
			return true
		}
	}
	return false
}

// Add will append err to the errors slice if non-nil.
func (je *joinError) Add(err error) {
	if err != nil {
		je.errs = append(je.errs, err)
	}
}

// Error implements the error interface
func (je joinError) Error() string {
	// We have no errors
	if len(je.errs) == 0 {
		return ""
	}
	// If we have a single error, just return it.
	if len(je.errs) == 1 {
		return je.errs[0].Error()
	}
	// We have multiple errors, so build up a nice error string.
	errorStrings := make([]string, len(je.errs))
	for i, err := range je.errs {
		errorStrings[i] = err.Error()
	}
	return fmt.Sprintf("multiple errors:\n- %s", strings.Join(errorStrings, "\n- "))
}

// Errors returns a slice containing zero or more errors that the supplied
// error is composed of. If the error is nil, a nil slice is returned.
//
// Example usage:
//
//	type myconfig struct{/*...*/}
//	conf, err := dotconfig.FromFileName[myconfig](".env")
//	// If we want to extract errors as a slice:
//	errors := dotconfig.Errors(err)
func Errors(err error) []error {
	return extractErrors(err)
}

func extractErrors(err error) []error {
	if err == nil {
		return nil
	}
	// check if err is a joinError .
	eg, ok := err.(joinError)
	if !ok {
		return []error{err}
	}

	return eg.errs
}
