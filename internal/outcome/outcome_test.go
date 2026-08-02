package outcome

import (
	"errors"
	"fmt"
	"testing"
)

func TestNew(t *testing.T) {
	err := errors.New("underlying error")
	appErr := New(1, err)

	var target *ApplicationOutcome
	if !errors.As(appErr, &target) {
		t.Fatalf("expected New to return an error that can be cast to *ApplicationOutcome")
	}

	if target.ExitCode != 1 {
		t.Errorf("expected ExitCode 1, got %d", target.ExitCode)
	}
	if target.Err != err {
		t.Errorf("expected Err to be %v, got %v", err, target.Err)
	}
}

func TestError(t *testing.T) {
	tests := []struct {
		name     string
		exitCode int
		err      error
		expected string
	}{
		{
			name:     "with underlying error",
			exitCode: 2,
			err:      errors.New("something went wrong"),
			expected: "something went wrong",
		},
		{
			name:     "without underlying error",
			exitCode: 3,
			err:      nil,
			expected: "application exited with code 3",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			appErr := New(tc.exitCode, tc.err)
			if got := appErr.Error(); got != tc.expected {
				t.Errorf("expected %q, got %q", tc.expected, got)
			}
		})
	}
}

func TestUnwrap(t *testing.T) {
	err := errors.New("underlying error")
	appErr := New(4, err)

	var target *ApplicationOutcome
	if !errors.As(appErr, &target) {
		t.Fatalf("expected error to be *ApplicationOutcome")
	}

	if got := target.Unwrap(); got != err {
		t.Errorf("expected Unwrap to return %v, got %v", err, got)
	}
}

func TestErrorsIs(t *testing.T) {
	baseErr := errors.New("base error")
	appErr := New(5, baseErr)
	wrappedErr := fmt.Errorf("wrapped: %w", appErr)

	if !errors.Is(appErr, baseErr) {
		t.Errorf("expected errors.Is(appErr, baseErr) to be true")
	}

	if !errors.Is(wrappedErr, baseErr) {
		t.Errorf("expected errors.Is(wrappedErr, baseErr) to be true")
	}
}

func TestErrorsAs(t *testing.T) {
	baseErr := errors.New("base error")
	appErr := New(6, baseErr)
	wrappedErr := fmt.Errorf("wrapped: %w", appErr)

	var target *ApplicationOutcome
	if !errors.As(wrappedErr, &target) {
		t.Fatalf("expected errors.As to succeed")
	}

	if target.ExitCode != 6 {
		t.Errorf("expected ExitCode 6, got %d", target.ExitCode)
	}
}
