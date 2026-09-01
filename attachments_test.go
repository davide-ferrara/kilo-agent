package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestFileReferencesStayInsideProject(t *testing.T) {
	root := t.TempDir()
	if err := os.WriteFile(filepath.Join(root, "note.txt"), []byte("hello"), 0o600); err != nil {
		t.Fatal(err)
	}
	path, ok := safeProjectPath(root, "note.txt")
	if !ok || !strings.HasSuffix(path, "note.txt") {
		t.Fatalf("safe path = %q, %v", path, ok)
	}
	if _, ok := safeProjectPath(root, "../escape.txt"); ok {
		t.Fatal("parent traversal accepted")
	}
}

func TestReferencedPathsSupportsSpaces(t *testing.T) {
	paths := referencedPaths("compare @{docs/user guide.md} with @README.md")
	if len(paths) != 2 || paths[0] != "docs/user guide.md" || paths[1] != "README.md" {
		t.Fatalf("paths = %#v", paths)
	}
}
