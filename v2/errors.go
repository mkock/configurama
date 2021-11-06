package configurama

import "fmt"

// NoKeyError represents unknown keys when required via the Option Require.
type NoKeyError string

// Error returns the error message for NoKeyError.
func (k NoKeyError) Error() string {
	return fmt.Sprintf("no such key: %q", string(k))
}

// ValidationError represents an error with value validation against a regular expression.
type ValidationError string

// Error returns the error message for ValidationError.
func (v ValidationError) Error() string {
	return fmt.Sprintf("validation failed for key: %q", string(v))
}

// ConversionError represents keys and values that can't be converted into the desired type.
type ConversionError struct {
	key, value, datatype string
}

// Error returns the error message for ConversionError.
func (c ConversionError) Error() string {
	return fmt.Sprintf("unable to convert value %q for key %q into %s", c.value, c.key, c.datatype)
}
