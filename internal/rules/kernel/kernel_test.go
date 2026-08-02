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

func TestVer_IsZero(t *testing.T) {
	tests := []struct {
		name string
		v    Ver
		want bool
	}{
		{"zero", Ver{0, 0, 0}, true},
		{"major non-zero", Ver{1, 0, 0}, false},
		{"minor non-zero", Ver{0, 1, 0}, false},
		{"patch non-zero", Ver{0, 0, 1}, false},
		{"all non-zero", Ver{1, 2, 3}, false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.v.IsZero(); got != tt.want {
				t.Errorf("Ver.IsZero() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestVer_AtLeast(t *testing.T) {
	tests := []struct {
		name  string
		v     Ver
		major int
		minor int
		patch int
		want  bool
	}{
		{"zero version", Ver{0, 0, 0}, 1, 0, 0, false},
		{"exact match", Ver{5, 10, 15}, 5, 10, 15, true},
		{"greater major", Ver{6, 0, 0}, 5, 10, 15, true},
		{"lesser major", Ver{4, 20, 20}, 5, 10, 15, false},
		{"equal major, greater minor", Ver{5, 11, 0}, 5, 10, 15, true},
		{"equal major, lesser minor", Ver{5, 9, 20}, 5, 10, 15, false},
		{"equal major, equal minor, greater patch", Ver{5, 10, 16}, 5, 10, 15, true},
		{"equal major, equal minor, lesser patch", Ver{5, 10, 14}, 5, 10, 15, false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.v.AtLeast(tt.major, tt.minor, tt.patch); got != tt.want {
				t.Errorf("Ver.AtLeast(%v, %v, %v) = %v, want %v", tt.major, tt.minor, tt.patch, got, tt.want)
			}
		})
	}
}

func TestVer_LessThan(t *testing.T) {
	tests := []struct {
		name  string
		v     Ver
		major int
		minor int
		patch int
		want  bool
	}{
		{"zero version", Ver{0, 0, 0}, 1, 0, 0, false},
		{"exact match", Ver{5, 10, 15}, 5, 10, 15, false},
		{"lesser major", Ver{4, 20, 20}, 5, 10, 15, true},
		{"greater major", Ver{6, 0, 0}, 5, 10, 15, false},
		{"equal major, lesser minor", Ver{5, 9, 20}, 5, 10, 15, true},
		{"equal major, greater minor", Ver{5, 11, 0}, 5, 10, 15, false},
		{"equal major, equal minor, lesser patch", Ver{5, 10, 14}, 5, 10, 15, true},
		{"equal major, equal minor, greater patch", Ver{5, 10, 16}, 5, 10, 15, false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.v.LessThan(tt.major, tt.minor, tt.patch); got != tt.want {
				t.Errorf("Ver.LessThan(%v, %v, %v) = %v, want %v", tt.major, tt.minor, tt.patch, got, tt.want)
			}
		})
	}
}

func TestVer_String(t *testing.T) {
	tests := []struct {
		name string
		v    Ver
		want string
	}{
		{"zero", Ver{0, 0, 0}, "0.0.0"},
		{"typical", Ver{5, 15, 20}, "5.15.20"},
		{"large numbers", Ver{99, 999, 9999}, "99.999.9999"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.v.String(); got != tt.want {
				t.Errorf("Ver.String() = %v, want %v", got, tt.want)
			}
		})
	}
}
