package cmd

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	homedir "github.com/mitchellh/go-homedir"
)

func TestSetupShellInstallsZshBlock(t *testing.T) {
	home := configureSetupTest(t, "/bin/bash")

	rcPath, err := setupShell("zsh")
	if err != nil {
		t.Fatal(err)
	}

	wantPath := filepath.Join(home, ".zshrc")
	if rcPath != wantPath {
		t.Fatalf("rcPath = %q, want %q", rcPath, wantPath)
	}

	content := readFile(t, rcPath)
	assertContains(t, content, setupMarkerStart)
	assertContains(t, content, "add-zsh-hook chpwd _pd_log_cwd")
	assertContains(t, content, "bindkey '^h' pd-switch")
}

func TestSetupShellInstallsBashBlock(t *testing.T) {
	home := configureSetupTest(t, "/bin/zsh")

	rcPath, err := setupShell("bash")
	if err != nil {
		t.Fatal(err)
	}

	wantPath := filepath.Join(home, ".bashrc")
	if rcPath != wantPath {
		t.Fatalf("rcPath = %q, want %q", rcPath, wantPath)
	}

	content := readFile(t, rcPath)
	assertContains(t, content, setupMarkerStart)
	assertContains(t, content, `PROMPT_COMMAND="_pd_log_cwd${PROMPT_COMMAND:+;$PROMPT_COMMAND}"`)
	assertContains(t, content, `*";_pd_log_cwd;"*) ;;`)
	assertContains(t, content, `bind -x '"\C-h": pd-switch'`)
}

func TestSetupShellDetectsShellFromEnv(t *testing.T) {
	home := configureSetupTest(t, "/usr/local/bin/zsh")

	rcPath, err := setupShell("")
	if err != nil {
		t.Fatal(err)
	}

	if rcPath != filepath.Join(home, ".zshrc") {
		t.Fatalf("rcPath = %q, want zsh rc in %q", rcPath, home)
	}
}

func TestSetupShellReplacesExistingManagedBlock(t *testing.T) {
	home := configureSetupTest(t, "/bin/zsh")
	rcPath := filepath.Join(home, ".zshrc")
	writeFile(t, rcPath, "before\n"+setupMarkerStart+"\nold setup\n"+setupMarkerEnd+"\nafter\n")

	if _, err := setupShell("zsh"); err != nil {
		t.Fatal(err)
	}
	if _, err := setupShell("zsh"); err != nil {
		t.Fatal(err)
	}

	content := readFile(t, rcPath)
	if strings.Count(content, setupMarkerStart) != 1 {
		t.Fatalf("setup start marker count = %d, want 1\n%s", strings.Count(content, setupMarkerStart), content)
	}
	if strings.Count(content, setupMarkerEnd) != 1 {
		t.Fatalf("setup end marker count = %d, want 1\n%s", strings.Count(content, setupMarkerEnd), content)
	}
	assertContains(t, content, "before\n")
	assertContains(t, content, "after\n")
	if strings.Contains(content, "old setup") {
		t.Fatalf("old managed block was not replaced:\n%s", content)
	}
}

func TestSetupShellRejectsUnsupportedShell(t *testing.T) {
	configureSetupTest(t, "/bin/fish")

	if _, err := setupShell(""); err == nil {
		t.Fatal("expected unsupported shell error")
	}
}

func configureSetupTest(t *testing.T, shell string) string {
	t.Helper()

	home := t.TempDir()
	t.Setenv("HOME", home)
	t.Setenv("SHELL", shell)

	oldDisableCache := homedir.DisableCache
	homedir.DisableCache = true
	t.Cleanup(func() {
		homedir.DisableCache = oldDisableCache
	})

	return home
}

func readFile(t *testing.T, path string) string {
	t.Helper()

	content, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}

	return string(content)
}

func assertContains(t *testing.T, content, want string) {
	t.Helper()
	if !strings.Contains(content, want) {
		t.Fatalf("expected content to contain %q:\n%s", want, content)
	}
}
