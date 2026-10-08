package app

type Severity uint8

const (
	Info Severity = iota
	Warning
	Error
	Success
)

// Diagnostic retains plain application data. Terminal styling/encoding belongs
// to presentation, and callers may preserve an underlying error for inspection.
type Diagnostic struct {
	Severity                    Severity
	Summary, Details, Operation string
	Cause                       error
}
