package errors

import (
	"errors"
	"fmt"
	"os"
	"syscall"
)

// ExitCode represents application exit codes
type ExitCode int

const (
	ExitSuccess ExitCode = iota
	ExitGeneralError
	ExitInvalidKeyFormat
	ExitConfigError
	ExitNetworkError
	ExitPermissionError
)

// AppError represents an application error with exit code
type AppError struct {
	Message  string
	ExitCode ExitCode
	Err      error
}

func (e *AppError) Error() string {
	if e.Err != nil {
		return fmt.Sprintf("%s: %v", e.Message, e.Err)
	}
	return e.Message
}

func (e *AppError) Unwrap() error {
	return e.Err
}

// NewAppError creates a new application error
func NewAppError(message string, exitCode ExitCode, err error) *AppError {
	return &AppError{
		Message:  message,
		ExitCode: exitCode,
		Err:      err,
	}
}

// ExitWithCode exits the application with the given exit code
func ExitWithCode(code ExitCode) {
	os.Exit(int(code))
}

// ExitWithError exits the application with the error's exit code
func ExitWithError(err error) {
	var appErr *AppError
	if errors.As(err, &appErr) {
		os.Exit(int(appErr.ExitCode))
	}
	os.Exit(int(ExitGeneralError))
}

// HandleInvalidKey handles invalid key format by terminating with SIGTERM
// This implements "fail secure" behavior
func HandleInvalidKey(key string, err error) {
	// Log the error before terminating
	fmt.Fprintf(os.Stderr, "ERROR: Invalid SSH key format: %q: %v\n", key, err)
	fmt.Fprintf(os.Stderr, "Terminating due to invalid key format (fail secure)\n")
	
	// Send SIGTERM to ourselves
	process, err := os.FindProcess(os.Getpid())
	if err == nil {
		process.Signal(syscall.SIGTERM)
	}
	
	// Fallback: exit with error code
	os.Exit(int(ExitInvalidKeyFormat))
}

// IsInvalidKeyError checks if an error is related to invalid key format
func IsInvalidKeyError(err error) bool {
	var appErr *AppError
	if errors.As(err, &appErr) {
		return appErr.ExitCode == ExitInvalidKeyFormat
	}
	return false
}

