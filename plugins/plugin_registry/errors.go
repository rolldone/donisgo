package plugin_registry

import "fmt"

// ══════════════════════════════════════════════════════════════════════════
// ERROR BERSAMA ANTAR PLUGIN
// ══════════════════════════════════════════════════════════════════════════
// Tipe error yang dibagikan antar plugin didefinisikan di sini supaya plugin
// lain bisa mendeteksinya dengan errors.As tanpa saling import.
//
// Contoh pemakaian di handler:
//
//	var le *plugin_registry.ContractError
//	if errors.As(err, &le) {
//		// beri respons terstruktur berdasarkan le.Code
//	}
//
// ══════════════════════════════════════════════════════════════════════════

// ContractError adalah contoh tipe error bersama antar plugin. Plugin dapat
// mendeteksi tipe ini untuk memberi respons terstruktur. Sesuaikan field-nya
// dengan kebutuhan nyata (misal error limit/kuota).
type ContractError struct {
	Code    string `json:"code"`
	Message string `json:"message"`
}

// Error implements the error interface.
func (e *ContractError) Error() string {
	return fmt.Sprintf("[%s] %s", e.Code, e.Message)
}

// NewContractError membuat ContractError baru.
func NewContractError(code, message string) *ContractError {
	return &ContractError{Code: code, Message: message}
}
