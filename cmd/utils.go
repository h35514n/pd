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
	"strings"

	homedir "github.com/mitchellh/go-homedir"
	"github.com/spf13/viper"
)

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

// listFilesEza lists the contents of path using "eza", printing the abbreviated
// path header (e.g. "~" for home).
func listFilesEza(path, abbreviated string) (string, error) {
	fmt.Println(abbreviated)

	cmd := exec.Command(
		"eza",
		"--all",
		"--color=always",
		"--group-directories-first",
		"-1",
		path,
	)

	return capturedOutput(cmd)
}

// listFilesLs lists the contents of path using "ls".
func listFilesLs(path, _ string) (string, error) {
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

// toSkipDirSet converts a slice of directory patterns to an expanded set.
func toSkipDirSet(strs []string) map[string]bool {
	set := make(map[string]bool, len(strs))

	for _, s := range strs {
		s = strings.TrimSpace(s)
		if s == "" {
			continue
		}

		expanded, err := homedir.Expand(s)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Warning: could not expand skip_dirs entry '%s': %v\n", s, err)
			continue
		}
		set[filepath.Clean(expanded)] = true
	}

	return set
}

func mergeSkipDirPatterns(userSkipDirs []string) []string {
	merged := make([]string, 0, len(defaultSkipDirs)+len(userSkipDirs))
	merged = append(merged, defaultSkipDirs...)
	merged = append(merged, userSkipDirs...)
	return merged
}

func isSkippedDir(path string) bool {
	path = filepath.Clean(path)
	for skipDir := range skipDirs {
		if path == skipDir {
			return true
		}

		rel, err := filepath.Rel(skipDir, path)
		if err == nil && rel != "." && rel != ".." && !strings.HasPrefix(rel, ".."+string(os.PathSeparator)) {
			return true
		}
	}

	return false
}
