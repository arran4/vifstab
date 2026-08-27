import re

with open("internal/editor/systemd_test.go", "r") as f:
    content = f.read()

# Replace failingWriteFS to implement CreateTemp and return our specific error
content = content.replace(
"""type failingWriteFS struct{ *mockFS }

func (f *failingWriteFS) WriteFile(string, []byte, os.FileMode) error {
	return errors.New("write failed")
}""",
"""type failingWriteFS struct{ *mockFS }

func (f *failingWriteFS) CreateTemp(dir, pattern string) (TempFile, error) {
	if dir == "/etc" {
		return nil, errors.New("failed to replace destination file")
	}
	return nil, errors.New("fallback error")
}"""
)

with open("internal/editor/systemd_test.go", "w") as f:
    f.write(content)

with open("internal/editor/editor.go", "r") as f:
    content = f.read()

# We need to add CreateTemp to our editor mockFS as well
with open("internal/editor/editor_test.go", "r") as f:
    test_content = f.read()

if "func (m *mockFS) CreateTemp" not in test_content:
    test_content = test_content.replace(
"""type mockFS struct {
	mfs fstest.MapFS

	statErr  error
	writeErr error
}""",
"""type mockFS struct {
	mfs fstest.MapFS

	statErr  error
	writeErr error
}

func (m *mockFS) CreateTemp(dir, pattern string) (TempFile, error) {
	return nil, fmt.Errorf("mockFS CreateTemp not implemented")
}"""
    )
    with open("internal/editor/editor_test.go", "w") as f:
        f.write(test_content)
