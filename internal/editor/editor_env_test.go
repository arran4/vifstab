package editor

import (
	"reflect"
	"testing"
)

func TestEditorCommandFromEnvironment(t *testing.T) {
	tests := []struct {
		name       string
		visual     string
		editor     string
		sudoEditor string
		want       []string
		wantErr    bool
	}{
		{
			name:       "VISUAL precedence",
			visual:     "nvim -f",
			editor:     "nano",
			sudoEditor: "/tmp/vi",
			want:       []string{"nvim", "-f"},
		},
		{
			name:       "Arbitrary editor and arguments",
			visual:     "",
			editor:     "micro -w",
			sudoEditor: "",
			want:       []string{"micro", "-w"},
		},
		{
			name:       "SUDO_EDITOR ignored",
			visual:     "",
			editor:     "",
			sudoEditor: "nvim",
			want:       []string{"vi"},
		},
		{
			name:       "VISUAL over EDITOR",
			visual:     "vim",
			editor:     "nano",
			sudoEditor: "",
			want:       []string{"vim"},
		},
		{
			name:       "Literal -- not allowed",
			visual:     "vi --",
			editor:     "",
			sudoEditor: "",
			wantErr:    true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Setenv("VISUAL", tt.visual)
			t.Setenv("EDITOR", tt.editor)
			t.Setenv("SUDO_EDITOR", tt.sudoEditor)

			got, err := editorCommandFromEnvironment()
			if (err != nil) != tt.wantErr {
				t.Errorf("editorCommandFromEnvironment() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !tt.wantErr && !reflect.DeepEqual(got, tt.want) {
				t.Errorf("editorCommandFromEnvironment() = %v, want %v", got, tt.want)
			}
		})
	}
}
