package outcome_test

import (
	"errors"
	"testing"

	"github.com/arran4/vifstab/internal/outcome"
)

func TestApplicationOutcome_Error(t *testing.T) {
	tests := []struct {
		name     string
		exitCode int
		err      error
		want     string
	}{
		{
			name:     "wrapped error present",
			exitCode: 1,
			err:      errors.New("custom error message"),
			want:     "custom error message",
		},
		{
			name:     "wrapped error is nil",
			exitCode: 2,
			err:      nil,
			want:     "application exited with code 2",
		},
		{
			name:     "wrapped error is nil zero exit code",
			exitCode: 0,
			err:      nil,
			want:     "application exited with code 0",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			appErr := outcome.New(tt.exitCode, tt.err)
			if got := appErr.Error(); got != tt.want {
				t.Errorf("ApplicationOutcome.Error() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestApplicationOutcome_Unwrap(t *testing.T) {
	underlying := errors.New("underlying cause")

	tests := []struct {
		name     string
		exitCode int
		err      error
		want     error
	}{
		{
			name:     "with underlying error",
			exitCode: 1,
			err:      underlying,
			want:     underlying,
		},
		{
			name:     "with nil underlying error",
			exitCode: 0,
			err:      nil,
			want:     nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := outcome.New(tt.exitCode, tt.err)
			got := errors.Unwrap(err)
			if got != tt.want {
				t.Errorf("errors.Unwrap() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestApplicationOutcome_NewAndFields(t *testing.T) {
	innerErr := errors.New("inner error")
	exitCode := 42

	err := outcome.New(exitCode, innerErr)

	var appOutcome *outcome.ApplicationOutcome
	if !errors.As(err, &appOutcome) {
		t.Fatalf("expected errors.As to match *outcome.ApplicationOutcome")
	}

	if appOutcome.ExitCode != exitCode {
		t.Errorf("ExitCode = %d, want %d", appOutcome.ExitCode, exitCode)
	}

	if appOutcome.Err != innerErr {
		t.Errorf("Err = %v, want %v", appOutcome.Err, innerErr)
	}

	if !errors.Is(err, innerErr) {
		t.Errorf("errors.Is(err, innerErr) = false, want true")
	}
}
