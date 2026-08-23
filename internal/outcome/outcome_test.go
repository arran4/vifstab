package outcome

import (
	"errors"
	"testing"
)

func TestApplicationOutcome_Error(t *testing.T) {
	tests := []struct {
		name     string
		exitCode int
		err      error
		expected string
	}{
		{
			name:     "with wrapped error",
			exitCode: 1,
			err:      errors.New("some error"),
			expected: "some error",
		},
		{
			name:     "without wrapped error",
			exitCode: 2,
			err:      nil,
			expected: "application exited with code 2",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			e := &ApplicationOutcome{
				ExitCode: tt.exitCode,
				Err:      tt.err,
			}
			if got := e.Error(); got != tt.expected {
				t.Errorf("ApplicationOutcome.Error() = %v, want %v", got, tt.expected)
			}
		})
	}
}

func TestApplicationOutcome_Unwrap(t *testing.T) {
	wrappedErr := errors.New("wrapped error")
	e := &ApplicationOutcome{
		ExitCode: 1,
		Err:      wrappedErr,
	}

	if got := e.Unwrap(); got != wrappedErr {
		t.Errorf("ApplicationOutcome.Unwrap() = %v, want %v", got, wrappedErr)
	}

	eNil := &ApplicationOutcome{
		ExitCode: 1,
		Err:      nil,
	}
	if got := eNil.Unwrap(); got != nil {
		t.Errorf("ApplicationOutcome.Unwrap() = %v, want %v", got, nil)
	}
}

func TestNew(t *testing.T) {
	wrappedErr := errors.New("test error")
	err := New(5, wrappedErr)

	outcome, ok := err.(*ApplicationOutcome)
	if !ok {
		t.Fatalf("New() did not return an *ApplicationOutcome, got %T", err)
	}

	if outcome.ExitCode != 5 {
		t.Errorf("ExitCode = %v, want %v", outcome.ExitCode, 5)
	}

	if outcome.Err != wrappedErr {
		t.Errorf("Err = %v, want %v", outcome.Err, wrappedErr)
	}
}
