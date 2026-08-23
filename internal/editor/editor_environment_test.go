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
			name:       "visual takes precedence and sudo editor is ignored",
			visual:     "nvim -f",
			editor:     "nano",
			sudoEditor: "/tmp/vi",
			want:       []string{"nvim", "-f"},
		},
		{
			name:       "editor supports arbitrary executable and arguments",
			editor:     "micro -w",
			sudoEditor: "vim",
			want:       []string{"micro", "-w"},
		},
		{
			name:       "sudo editor alone does not affect selection",
			sudoEditor: "nvim",
			want:       []string{"vi"},
		},
		{
			name:    "separator argument remains rejected",
			editor:  "vim -- -f",
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Setenv("VISUAL", tt.visual)
			t.Setenv("EDITOR", tt.editor)
			t.Setenv("SUDO_EDITOR", tt.sudoEditor)

			got, err := editorCommandFromEnvironment()
			if (err != nil) != tt.wantErr {
				t.Fatalf("editorCommandFromEnvironment() error = %v, wantErr %v", err, tt.wantErr)
			}
			if !tt.wantErr && !reflect.DeepEqual(got, tt.want) {
				t.Fatalf("editorCommandFromEnvironment() = %v, want %v", got, tt.want)
			}
		})
	}
}
