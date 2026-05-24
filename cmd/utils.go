/*
Copyright © 2026 J.M. Romer <h35514n@icloud.com>

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

	http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/
package cmd

import (
	"bytes"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/charmbracelet/glamour"
	homedir "github.com/mitchellh/go-homedir"
	"github.com/spf13/viper"
)

var readmeFilenames = []string{
	"README.md",
	"README.rst",
	"README.txt",
	"README.org",
	"README.adoc",
	"README.asciidoc",
	"README",
}

func ensureDirExists(dir string) {
	if _, err := os.Stat(dir); os.IsNotExist(err) {
		if err := os.MkdirAll(dir, 0o755); err != nil {
			check(err)
		}
	}
}

// exists returns true if the given file or directory exists, else false.
func exists(path string) bool {
	_, err := os.Stat(path)
	return err == nil || !os.IsNotExist(err)
}

// expandPath expands the given path to an absolute path, resolving "~" and
// environment variables. The empty string yields the user's home directory.
func expandPath(path string) string {
	path = strings.TrimSpace(path)
	if path == "" {
		return homeDir()
	}

	path = filepath.Clean(path)

	expanded, err := homedir.Expand(path)
	check(err)

	abspath, err := filepath.Abs(expanded)
	check(err)

	return abspath
}

// homeDir returns the current user's home directory.
func homeDir() string {
	home, err := homedir.Dir()
	check(err)
	return home
}

// workingDir returns the current working directory, or the home directory on
// error.
func workingDir() string {
	pwd, err := os.Getwd()
	if err == nil {
		return pwd
	}
	return homeDir()
}

// configDir returns the pd config directory, creating it if necessary.
func configDir() string {
	xdgDir := os.Getenv("XDG_CONFIG_HOME")
	if xdgDir == "" {
		xdgDir = os.ExpandEnv("$HOME/.config")
	}
	dir := filepath.Join(xdgDir, "pd")
	ensureDirExists(dir)
	return dir
}

// stateDir returns the pd state directory, creating it if necessary.
func stateDir() string {
	xdgDir := os.Getenv("XDG_STATE_HOME")
	if xdgDir == "" {
		xdgDir = os.ExpandEnv("$HOME/.local/state")
	}
	dir := filepath.Join(xdgDir, "pd")
	ensureDirExists(dir)
	return dir
}

// isProject returns true if path is considered a project directory.
func isProject(path string) bool {
	for _, marker := range activeProjectMarkers() {
		if markerMatches(path, marker) {
			return true
		}
	}

	return false
}

func activeProjectMarkers() []string {
	if len(projectMarkers) > 0 {
		return projectMarkers
	}

	return defaultProjectMarkers
}

func markerMatches(path, marker string) bool {
	requireDir := strings.HasSuffix(marker, "/")
	marker = strings.TrimSuffix(marker, "/")
	if marker == "" || filepath.IsAbs(marker) {
		return false
	}

	candidate := filepath.Join(path, filepath.Clean(marker))
	info, err := os.Stat(candidate)
	if err != nil {
		return false
	}

	return !requireDir || info.IsDir()
}

func cleanProjectMarkers(markers []string) []string {
	cleaned := make([]string, 0, len(markers))
	for _, marker := range markers {
		marker = strings.TrimSpace(marker)
		if marker == "" || filepath.IsAbs(marker) {
			continue
		}
		cleaned = append(cleaned, marker)
	}

	return cleaned
}

// check prints an error and exits non-zero.
func check(err error) {
	if err != nil {
		fmt.Println("[pd] Encountered an error:")
		fmt.Println(err.Error())
		os.Exit(1)
	}
}

// checkConfigFile terminates if the config file exists but cannot be read.
func checkConfigFile(err error) {
	if err == nil {
		return
	}

	_, notFound := err.(viper.ConfigFileNotFoundError)
	if notFound {
		return
	}

	check(err)
}

// listFilesEza lists the contents of path using "eza"
func listFilesEza(path, abbreviated string) (string, error) {
	cmd := exec.Command(
		"eza",
		"--tree",
		"-L2",
		"--only-dirs",
		"--color=always",
		"--group-directories-first",
		path,
	)

	return capturedOutput(cmd)
}

// listFilesLs lists the contents of path using "ls".
func listFilesLs(path, _ string) (string, error) {
	fmt.Println(path)
	cmd := exec.Command(
		"ls",
		"--almost-all",
		"--color=always",
		"--group-directories-first",
		"--human-readable",
		"-1",
		path,
	)

	return capturedOutput(cmd)
}

// listFilesTree lists the contents of path using "tree".
func listFilesTree(path string) (string, error) {
	cmd := exec.Command(
		"tree",
		"-C",
		"-L", "2",
		path,
	)

	return capturedOutput(cmd)
}

// capturedOutput runs cmd and returns its stdout as a string.
func capturedOutput(cmd *exec.Cmd) (string, error) {
	var out bytes.Buffer
	cmd.Stdout = &out

	err := cmd.Run()
	if err != nil {
		return "", err
	}

	return out.String(), nil
}

// findReadme returns the path to the first README-style file found in dir,
// or "" if none is present.
func findReadme(dir string) string {
	for _, name := range readmeFilenames {
		p := filepath.Join(dir, name)
		if exists(p) {
			return p
		}
	}
	return ""
}

// renderReadmeGlamour renders a Markdown file to ANSI in-process using glamour.
// Style is controlled by the GLAMOUR_STYLE env var (default: "dark").
func renderReadmeGlamour(path string) (string, error) {
	b, err := os.ReadFile(path)
	if err != nil {
		return "", err
	}

	style := os.Getenv("GLAMOUR_STYLE")
	if style == "" {
		style = "dark"
	}

	r, err := glamour.NewTermRenderer(
		glamour.WithStandardStyle(style),
		glamour.WithWordWrap(previewWidth()),
	)
	if err != nil {
		return "", err
	}

	return r.Render(string(b))
}

// renderReadmeBat renders a file using bat with ANSI color and no decorations.
func renderReadmeBat(path string) (string, error) {
	cmd := exec.Command("bat", "--color=always", "--plain", path)
	return capturedOutput(cmd)
}

// renderReadmePlain reads a file as plain text.
func renderReadmePlain(path string) (string, error) {
	b, err := os.ReadFile(path)
	if err != nil {
		return "", err
	}
	return string(b), nil
}

// previewWidth returns the FZF preview column count from the environment,
// defaulting to 80.
func previewWidth() int {
	if w := os.Getenv("FZF_PREVIEW_COLUMNS"); w != "" {
		if n, err := strconv.Atoi(w); err == nil {
			return n
		}
	}
	return 80
}

// mergeExcludes prepends defaultExcludes to user entries (additive merge).
func mergeExcludes(user []string) []string {
	merged := make([]string, 0, len(defaultExcludes)+len(user))
	merged = append(merged, defaultExcludes...)
	merged = append(merged, user...)
	return merged
}

// classifyExcludes splits exclude entries into path patterns and basename
// patterns. Entries containing '/' are paths (with ~/ expansion); others are
// basename globs. Invalid patterns are dropped with a stderr warning.
func classifyExcludes(entries []string) (paths, basenames []string) {
	for _, entry := range entries {
		entry = strings.TrimSpace(entry)
		if entry == "" {
			continue
		}
		if _, err := filepath.Match(entry, "x"); err != nil {
			fmt.Fprintf(os.Stderr,
				"Warning: invalid excludes pattern %q: %v\n", entry, err)
			continue
		}
		if strings.Contains(entry, "/") {
			expanded, err := homedir.Expand(entry)
			if err != nil {
				fmt.Fprintf(os.Stderr,
					"Warning: could not expand excludes entry %q: %v\n", entry, err)
				continue
			}
			paths = append(paths, filepath.Clean(expanded))
			continue
		}
		basenames = append(basenames, entry)
	}
	return paths, basenames
}

// containsGlobChars reports whether s has any filepath.Match metacharacter.
func containsGlobChars(s string) bool {
	return strings.ContainsAny(s, "*?[")
}

// isExcludedPath reports whether path matches an excludePathPatterns entry.
// Non-glob entries match the path itself and all descendants; glob entries
// use filepath.Match against the full path.
func isExcludedPath(path string) bool {
	path = filepath.Clean(path)
	for _, pattern := range excludePathPatterns {
		if containsGlobChars(pattern) {
			if ok, _ := filepath.Match(pattern, path); ok {
				return true
			}
			continue
		}
		if path == pattern {
			return true
		}
		rel, err := filepath.Rel(pattern, path)
		if err == nil && rel != "." && rel != ".." &&
			!strings.HasPrefix(rel, ".."+string(os.PathSeparator)) {
			return true
		}
	}
	return false
}

// isExcludedBasename reports whether name matches any pattern in
// excludeBasenamePatterns via filepath.Match.
func isExcludedBasename(name string) bool {
	for _, pattern := range excludeBasenamePatterns {
		if ok, _ := filepath.Match(pattern, name); ok {
			return true
		}
	}
	return false
}
