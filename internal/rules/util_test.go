package rules

import (
	"testing"
)

func TestResolveDevicePath(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"UUID=1234-5678", "/dev/disk/by-uuid/1234-5678"},
		{"LABEL=mydisk", "/dev/disk/by-label/mydisk"},
		{"PARTUUID=abcd-efgh", "/dev/disk/by-partuuid/abcd-efgh"},
		{"PARTLABEL=mypart", "/dev/disk/by-partlabel/mypart"},
		{"ID=wwn-0x123", "/dev/disk/by-id/wwn-0x123"},
		{"uuid=1234-5678", "/dev/disk/by-uuid/1234-5678"},
		{" LABEL = mydisk ", "/dev/disk/by-label/mydisk"},
		{"UUID=\"1234-5678\"", "/dev/disk/by-uuid/1234-5678"},
		{"UUID='1234-5678'", "/dev/disk/by-uuid/1234-5678"},
		{"/dev/sda1", "/dev/sda1"},
		{"invalid=xyz", "invalid=xyz"},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			got := ResolveDevicePath(tt.input)
			if got != tt.expected {
				t.Errorf("ResolveDevicePath(%q) = %q; expected %q", tt.input, got, tt.expected)
			}
		})
	}
}
