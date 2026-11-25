package plugin_registry

import "fmt"

// ContractError is a sample shared error type. Plugins can detect it (with
// errors.As) to return a structured response without importing each other:
//
//	var ce *plugin_registry.ContractError
//	if errors.As(err, &ce) {
//		// build a structured response from ce.Code
//	}
//
// Adjust its fields to your real needs (for example limit/quota errors).
type ContractError struct {
	// Code is a machine-readable error code (e.g. "quota_exceeded").
	Code string `json:"code"`
	// Message is a human-readable description.
	Message string `json:"message"`
}

// Error implements the error interface.
func (e *ContractError) Error() string {
	return fmt.Sprintf("[%s] %s", e.Code, e.Message)
}

// NewContractError returns a ContractError with the given code and message.
func NewContractError(code, message string) *ContractError {
	return &ContractError{Code: code, Message: message}
}
