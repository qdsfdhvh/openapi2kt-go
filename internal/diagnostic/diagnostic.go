// Package diagnostic defines payload-safe, source-located generator diagnostics.
package diagnostic

// Diagnostic is a stable code and safe message with an optional JSON Pointer.
// A pointer to "" identifies the document root; nil means no schema location.
type Diagnostic struct {
	Code    string  `json:"code"`
	Message string  `json:"message"`
	Pointer *string `json:"pointer,omitempty"`
}

// New creates a diagnostic without a source location.
func New(code, message string) *Diagnostic {
	return &Diagnostic{Code: code, Message: message}
}

// At creates a diagnostic at a JSON Pointer, including an empty root pointer.
func At(code, message, pointer string) *Diagnostic {
	return &Diagnostic{Code: code, Message: message, Pointer: &pointer}
}

func (d Diagnostic) Error() string { return d.Code + ": " + d.Message }

// Diagnostics implements error while retaining structured diagnostic entries.
// Messages must not embed input payloads, filenames, or raw operating-system errors.
type Diagnostics []Diagnostic

func (ds Diagnostics) Error() string {
	if len(ds) == 0 {
		return "generation failed"
	}
	return ds[0].Error()
}
