package cmd

import (
	"os"
	"path/filepath"
	"testing"
)

func TestIsProjectMatchesDefaultMarkers(t *testing.T) {
	projectMarkers = nil
	t.Cleanup(func() { projectMarkers = nil })

	tests := []struct {
		name  string
		setup func(string) error
	}{
		{
			name: ".git directory",
			setup: func(dir string) error {
				return os.Mkdir(filepath.Join(dir, ".git"), 0o755)
			},
		},
		{
			name: ".git file",
			setup: func(dir string) error {
				return os.WriteFile(filepath.Join(dir, ".git"), []byte("gitdir: ../.git/modules/repo\n"), 0o644)
			},
		},
		{
			name: ".projectile",
			setup: func(dir string) error {
				return os.WriteFile(filepath.Join(dir, ".projectile"), nil, 0o644)
			},
		},
		{
			name: "Makefile",
			setup: func(dir string) error {
				return os.WriteFile(filepath.Join(dir, "Makefile"), nil, 0o644)
			},
		},
		{
			name: "src directory",
			setup: func(dir string) error {
				return os.Mkdir(filepath.Join(dir, "src"), 0o755)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			dir := t.TempDir()
			if err := tt.setup(dir); err != nil {
				t.Fatal(err)
			}

			if !isProject(dir) {
				t.Fatalf("expected %s to be detected as a project", dir)
			}
		})
	}
}

func TestIsProjectRequiresDirectoryForDirectoryMarkers(t *testing.T) {
	projectMarkers = []string{"src/"}
	t.Cleanup(func() { projectMarkers = nil })

	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "src"), nil, 0o644); err != nil {
		t.Fatal(err)
	}

	if isProject(dir) {
		t.Fatalf("expected file marker with trailing slash to be ignored")
	}
}

func TestIsProjectDoesNotMatchUnknownDirectory(t *testing.T) {
	projectMarkers = nil
	t.Cleanup(func() { projectMarkers = nil })

	if isProject(t.TempDir()) {
		t.Fatalf("expected empty directory not to be detected as a project")
	}
}

func TestIsProjectMatchesCustomMarker(t *testing.T) {
	projectMarkers = []string{"config.toml"}
	t.Cleanup(func() { projectMarkers = nil })

	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "config.toml"), nil, 0o644); err != nil {
		t.Fatal(err)
	}

	if !isProject(dir) {
		t.Fatalf("expected custom marker to be detected as a project")
	}
}
