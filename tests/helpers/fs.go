// Package helpers - fs provides utility functions for filesystem operations in tests.
package helpers

import (
	"os"
	"path/filepath"
	"testing"
)

type TestFS struct {
	Root string
}

func NewTestFS(t *testing.T) *TestFS {
	t.Helper()
	return &TestFS{Root: t.TempDir()}
}

func (fs *TestFS) Path(parts ...string) string {
	return filepath.Join(append([]string{fs.Root}, parts...)...)
}

func (fs *TestFS) Write(rel string, content string) {
	path := fs.Path(rel)
	os.MkdirAll(filepath.Dir(path), 0755)
	if err := os.WriteFile(path, []byte(content), 0644); err != nil {
		panic(err)
	}
}

func (fs *TestFS) Cleanup() {
	os.RemoveAll(fs.Root)
}
