package boltimore

import "fmt"

type statusCodeError struct {
	statusCode int
	message    string
}

func (s statusCodeError) Error() string {
	return fmt.Sprintf("%d: %s", s.statusCode, s.message)
}

var _ error = statusCodeError{}

func StatusCodeErr(statusCode int, message string) error {
	return statusCodeError{
		statusCode: statusCode,
		message:    message,
	}
}
