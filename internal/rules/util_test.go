package rules

import (
	"reflect"
	"testing"
)

func TestSplitOptions(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected []string
	}{
		{"empty", "", nil},
		{"defaults", "defaults", []string{"defaults"}},
		{"single", "ro", []string{"ro"}},
		{"multiple", "ro,noatime", []string{"ro", "noatime"}},
		{"quotes", "context=\"system_u:object_r:tmpfs_t:s0\",ro", []string{"context=\"system_u:object_r:tmpfs_t:s0\"", "ro"}},
		{"quotes_with_comma", "context=\"a,b\",ro", []string{"context=\"a,b\"", "ro"}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := SplitOptions(tt.input)
			if !reflect.DeepEqual(got, tt.expected) {
				t.Errorf("SplitOptions(%q) = %v, want %v", tt.input, got, tt.expected)
			}
		})
	}
}
