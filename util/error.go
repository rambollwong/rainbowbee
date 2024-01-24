package util

import (
	"errors"
	"net"
	"strings"
)

// NetError checks if the provided error implements the net.Error interface.
// It returns the net.Error value and true if the error implements the interface.
// Otherwise, it returns nil and false.
func NetError(err error) (net.Error, bool) {
	var ne net.Error
	ok := errors.As(err, &ne)
	if ok {
		return ne, true
	}
	return nil, false
}

// IsNetErrorTimeout checks if the provided error is a network error and specifically a timeout error.
// It uses the NetError function to obtain the net.Error value and checks if the error's Timeout() method returns true.
// It returns true if the error represents a network timeout, and false otherwise.
func IsNetErrorTimeout(err error) bool {
	ne, ok := NetError(err)
	if ok {
		return ne.Timeout()
	}
	return false
}

// IsConnClosedError checks if the provided error corresponds to a closed connection error.
// It checks if the error string contains specific substrings that commonly indicate a closed connection.
// It returns true if the error represents a closed connection, and false otherwise.
func IsConnClosedError(err error) bool {
	// todo
	errStr := err.Error()
	return strings.Contains(errStr, "Application error 0x0") || strings.Contains(errStr, "connection closed")
}
