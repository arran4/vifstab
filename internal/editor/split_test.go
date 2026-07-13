package editor

import (
	"reflect"
	"testing"
)

func TestSplitEditorCommand(t *testing.T) {
	tests := []struct {
		name    string
		input   string
		want    []string
		wantErr bool
	}{
		{
			name:  "simple command",
			input: "vi",
			want:  []string{"vi"},
		},
		{
			name:  "command with arguments",
			input: "vi -c '!sh'",
			want:  []string{"vi", "-c", "!sh"},
		},
		{
			name:  "multiple spaces",
			input: "nano   -v",
			want:  []string{"nano", "-v"},
		},
		{
			name:  "double quotes",
			input: "vi \"/etc/shadow\"",
			want:  []string{"vi", "/etc/shadow"},
		},
		{
			name:  "single quotes",
			input: "vi '/etc/shadow'",
			want:  []string{"vi", "/etc/shadow"},
		},
		{
			name:  "escaped spaces",
			input: "vi /tmp/my\\ editor",
			want:  []string{"vi", "/tmp/my editor"},
		},
		{
			name:  "escaped quotes",
			input: "vi \\\"hello\\\"",
			want:  []string{"vi", "\"hello\""},
		},
		{
			name:    "unterminated single quote",
			input:   "vi 'hello",
			wantErr: true,
		},
		{
			name:    "unterminated double quote",
			input:   "vi \"hello",
			wantErr: true,
		},
		{
			name:    "unterminated escape",
			input:   "vi hello\\",
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := SplitEditorCommand(tt.input)
			if (err != nil) != tt.wantErr {
				t.Errorf("SplitEditorCommand() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !tt.wantErr && !reflect.DeepEqual(got, tt.want) {
				t.Errorf("SplitEditorCommand() = %v, want %v", got, tt.want)
			}
		})
	}
}
