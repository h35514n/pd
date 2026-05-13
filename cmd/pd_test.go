package cmd

import (
	"os"
	"path/filepath"
	"sort"
	"testing"

	homedir "github.com/mitchellh/go-homedir"
)

func TestCollectUserProjectsIncludesDeclaredSubmodules(t *testing.T) {
	home := configureProjectCollectionTest(t)

	repo := filepath.Join(home, "repo")
	submodule := filepath.Join(repo, "modules", "one")
	nested := filepath.Join(submodule, "deps", "two")

	mkdirAll(t, filepath.Join(repo, ".git"))
	mkdirAll(t, submodule)
	mkdirAll(t, nested)
	writeFile(t, filepath.Join(submodule, ".git"), "gitdir: ../../.git/modules/one\n")
	writeFile(t, filepath.Join(nested, ".git"), "gitdir: ../../../.git/modules/two\n")
	writeFile(t, filepath.Join(repo, ".gitmodules"), "[submodule \"one\"]\n\tpath = modules/one\n")
	writeFile(t, filepath.Join(submodule, ".gitmodules"), "[submodule \"two\"]\n\tpath = deps/two\n")

	projects := collectUserProjects()
	sort.Strings(projects)

	want := []string{repo, submodule, nested}
	if !equalStrings(projects, want) {
		t.Fatalf("projects = %#v, want %#v", projects, want)
	}
}

func TestCollectUserProjectsIncludesDotProjectSubmodules(t *testing.T) {
	home := configureProjectCollectionTest(t)

	repo := filepath.Join(home, ".dotfiles")
	submodule := filepath.Join(repo, "shell", "share", "pd")
	nested := filepath.Join(submodule, "deps", "one")

	mkdirAll(t, filepath.Join(repo, ".git"))
	mkdirAll(t, submodule)
	mkdirAll(t, nested)
	writeFile(t, filepath.Join(submodule, ".git"), "gitdir: ../../../.git/modules/pd\n")
	writeFile(t, filepath.Join(nested, ".git"), "gitdir: ../../../../.git/modules/one\n")
	writeFile(t, filepath.Join(repo, ".gitmodules"), "[submodule \"pd\"]\n\tpath = shell/share/pd\n")
	writeFile(t, filepath.Join(submodule, ".gitmodules"), "[submodule \"one\"]\n\tpath = deps/one\n")

	projects := collectUserProjects()
	sort.Strings(projects)

	want := []string{repo, submodule, nested}
	if !equalStrings(projects, want) {
		t.Fatalf("projects = %#v, want %#v", projects, want)
	}
}

func TestCollectUserProjectsSkipsDefaultSystemDirs(t *testing.T) {
	home := configureProjectCollectionTest(t)

	trash := filepath.Join(home, ".Trash")
	libraryProject := filepath.Join(home, "Library", "Caches", "repo")
	cacheProject := filepath.Join(home, ".cache", "tool", "repo")
	mkdirAll(t, filepath.Join(trash, "lib"))
	mkdirAll(t, filepath.Join(libraryProject, ".git"))
	mkdirAll(t, filepath.Join(cacheProject, ".git"))

	skipDirs = toSkipDirSet(defaultSkipDirs)

	projects := collectUserProjects()
	if len(projects) != 0 {
		t.Fatalf("projects = %#v, want none", projects)
	}
}

func TestCollectUserProjectsMergesDefaultAndUserSkipDirs(t *testing.T) {
	home := configureProjectCollectionTest(t)

	trash := filepath.Join(home, ".Trash")
	userSkipped := filepath.Join(home, "work", "generated")
	mkdirAll(t, filepath.Join(trash, "lib"))
	mkdirAll(t, filepath.Join(userSkipped, ".git"))

	skipDirs = toSkipDirSet(mergeSkipDirPatterns([]string{userSkipped}))

	projects := collectUserProjects()
	if len(projects) != 0 {
		t.Fatalf("projects = %#v, want none", projects)
	}
}

func TestCollectUserProjectsRespectsSkipDirsForSubmodules(t *testing.T) {
	home := configureProjectCollectionTest(t)

	repo := filepath.Join(home, "repo")
	submodule := filepath.Join(repo, "modules", "one")

	mkdirAll(t, filepath.Join(repo, ".git"))
	mkdirAll(t, submodule)
	writeFile(t, filepath.Join(submodule, ".git"), "gitdir: ../../.git/modules/one\n")
	writeFile(t, filepath.Join(repo, ".gitmodules"), "[submodule \"one\"]\n\tpath = modules/one\n")

	skipDirs = toSkipDirSet([]string{filepath.Join(repo, "modules") + string(os.PathSeparator)})

	projects := collectUserProjects()
	if !equalStrings(projects, []string{repo}) {
		t.Fatalf("projects = %#v, want %#v", projects, []string{repo})
	}
}

func TestCollectUserProjectsRespectsTrailingSlashSkipDirs(t *testing.T) {
	home := configureProjectCollectionTest(t)

	libraryProject := filepath.Join(home, "Library", "Caches", "repo")
	mkdirAll(t, filepath.Join(libraryProject, ".git"))

	skipDirs = toSkipDirSet([]string{filepath.Join(home, "Library") + string(os.PathSeparator)})

	projects := collectUserProjects()
	if len(projects) != 0 {
		t.Fatalf("projects = %#v, want none", projects)
	}
}

func configureProjectCollectionTest(t *testing.T) string {
	t.Helper()

	home := t.TempDir()
	t.Setenv("HOME", home)

	oldDisableCache := homedir.DisableCache
	homedir.DisableCache = true
	oldSkipDirs := skipDirs
	oldProjectMarkers := projectMarkers
	skipDirs = make(map[string]bool)
	projectMarkers = nil

	t.Cleanup(func() {
		homedir.DisableCache = oldDisableCache
		skipDirs = oldSkipDirs
		projectMarkers = oldProjectMarkers
	})

	return home
}

func mkdirAll(t *testing.T, path string) {
	t.Helper()
	if err := os.MkdirAll(path, 0o755); err != nil {
		t.Fatal(err)
	}
}

func writeFile(t *testing.T, path, content string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
}

func equalStrings(got, want []string) bool {
	if len(got) != len(want) {
		return false
	}

	for i := range got {
		if got[i] != want[i] {
			return false
		}
	}

	return true
}
