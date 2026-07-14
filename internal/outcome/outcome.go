package outcome

import "fmt"

// ApplicationOutcome represents a known error condition with a designated exit code.
type ApplicationOutcome struct {
	ExitCode int
	Err      error
}

func (e *ApplicationOutcome) Error() string {
	if e.Err != nil {
		return e.Err.Error()
	}
	return fmt.Sprintf("application exited with code %d", e.ExitCode)
}

func (e *ApplicationOutcome) Unwrap() error {
	return e.Err
}

// New creates a new ApplicationOutcome with the given exit code and wrapped error.
func New(exitCode int, err error) error {
	return &ApplicationOutcome{
		ExitCode: exitCode,
		Err:      err,
	}
}
