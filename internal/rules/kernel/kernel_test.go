package kernel

import (
	"testing"
)

func TestParseKernelVersion(t *testing.T) {
	maj, min, p := parseKernelVersionStr("Linux version 6.18.35-gentoo-dist")
	if maj != 6 || min != 18 || p != 35 {
		t.Errorf("Expected 6.18.35, got %d.%d.%d", maj, min, p)
	}

	maj, min, p = parseKernelVersionStr("6.1.1-arch1-1")
	if maj != 6 || min != 1 || p != 1 {
		t.Errorf("Expected 6.1.1, got %d.%d.%d", maj, min, p)
	}
}
