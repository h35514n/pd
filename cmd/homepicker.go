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
	"io"
	"io/fs"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

const homePickerMaxDepth = 10

// HomePicker launches a fallback FZF picker over every directory under $HOME
// (up to homePickerMaxDepth), honoring excludes. The walk runs concurrently
// with fzf and streams entries to stdin as it discovers them, so the picker
// is responsive immediately. On selection, prints the absolute path to stdout.
// On cancellation, exits silently so the calling pd sees an empty selection.
//
// Note: this bypasses go-finder. go-finder runs its source goroutine to
// completion *before* starting fzf, which deadlocks on inputs that exceed
// the OS pipe buffer (~64KB). The home walk easily produces megabytes of
// paths, so we drive exec.Cmd directly.
func HomePicker() {
	cmd := exec.Command(
		"fzf",
		"--ansi",
		"--cycle",
		"--exact",
		"--no-multi",
		"--no-sort",
		"--reverse",
		"--tiebreak=index",
		"--prompt=~> ",
		"--header=$HOME subdirectories",
	)
	cmd.Stderr = os.Stderr

	stdin, err := cmd.StdinPipe()
	check(err)

	var stdout bytes.Buffer
	cmd.Stdout = &stdout

	check(cmd.Start())

	go func() {
		defer stdin.Close()
		streamHomeDirectories(homePickerMaxDepth, stdin)
	}()

	if err := cmd.Wait(); err != nil {
		// Exit 130 means the user cancelled (Esc / Ctrl-C). Exit silently
		// so the parent pd sees an empty selection.
		if exitErr, ok := err.(*exec.ExitError); ok && exitErr.ExitCode() == 130 {
			return
		}
		check(err)
	}

	sel := strings.TrimSpace(stdout.String())
	if sel == "" {
		return
	}

	fmt.Println(filepath.Join(homeDir(), sel))
}

// streamHomeDirectories walks $HOME and writes each discovered directory
// (relative to $HOME) to out as a separate line. Honors excludes. Caps
// recursion at maxDepth. Stops walking early if the consumer closes its end
// of the pipe.
func streamHomeDirectories(maxDepth int, out io.Writer) {
	home := homeDir()
	sep := string(os.PathSeparator)

	_ = filepath.WalkDir(home, func(path string, d fs.DirEntry, err error) error {
		if err != nil || !d.IsDir() {
			return nil
		}
		if isExcludedPath(path) || isExcludedBasename(d.Name()) {
			return filepath.SkipDir
		}

		rel, err := filepath.Rel(home, path)
		if err != nil || rel == "." {
			return nil
		}

		if _, werr := fmt.Fprintln(out, rel); werr != nil {
			// fzf closed stdin (user selected or cancelled). Stop walking.
			return filepath.SkipAll
		}

		depth := strings.Count(rel, sep) + 1
		if depth >= maxDepth {
			return filepath.SkipDir
		}
		return nil
	})
}
