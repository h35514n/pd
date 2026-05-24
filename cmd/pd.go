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
	"bufio"
	"fmt"
	"math"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"time"

	finder "github.com/b4b4r07/go-finder"
	"github.com/b4b4r07/go-finder/source"
	"github.com/logrusorgru/aurora"
)

// SelectProject uses FZF to select a project directory and prints the selected
// directory's absolute path to stdout.
func SelectProject() {
	fzf, err := finder.New(
		"fzf",
		"--ansi",
		"--bind 'ctrl-b:preview-up'",
		"--bind 'ctrl-f:preview-down'",
		"--bind 'ctrl-h:become(pd --home-picker)'",
		"--cycle",
		"--exact",
		"--no-multi",
		"--no-sort",
		"--preview-window='right,60%,<60(top,80%)'",
		"--reverse",
		"--tiebreak=index",
		`--preview="pd --fzf-preview "{+}""`,
	)
	check(err)

	if !exists(historyFile) {
		RefreshLog(true)
	}

	projects := currentlyLoggedProjects()
	if len(projects) == 0 {
		fmt.Println(workingDir())
		return
	}

	listingEntries, listingIndex := searchListing(projects)
	fzf.Read(listingEntries)

	selection, err := fzf.Run()
	check(err)

	// Bail if selection is canceled
	if len(selection) == 0 {
		fmt.Println(workingDir())
		return
	}

	// The selected label is stripped of ANSI color codes; use listingIndex to
	// retrieve the associated absolute path. If the selection isn't in the
	// index, it came from --home-picker as an already-absolute path.
	sel := selection[0]
	abspath, ok := listingIndex[sel]
	if !ok {
		abspath = sel
	}
	fmt.Println(abspath)
	addLogEntry(abspath)

	RefreshLog(false)
}

// FzfPreview shows a README when one is found in the project directory,
// falling back to a directory listing (eza → tree → ls) otherwise.
// This is intended for the FZF preview window.
func FzfPreview(label string) {
	abspath := projectLabelToAbsPath(label)
	abbreviated := strings.Replace(abspath, homeDir(), "~", 1)

	if readme := findReadme(abspath); readme != "" {
		fmt.Println(abbreviated)

		ext := strings.ToLower(filepath.Ext(readme))
		var content string
		var err error

		if ext == ".md" {
			content, err = renderReadmeGlamour(readme)
		}
		if content == "" || err != nil {
			content, err = renderReadmeBat(readme)
		}
		if content == "" || err != nil {
			content, _ = renderReadmePlain(readme)
		}

		if content != "" {
			fmt.Print(content)
			return
		}
	}

	list, err := listFilesEza(abspath, abbreviated)

	switch {
	case err != nil:
		list, err = listFilesTree(abspath)
		fallthrough
	case err != nil:
		list, err = listFilesLs(abspath, abbreviated)
		fallthrough
	case err == nil && len(list) > 0:
		fmt.Println(list)
	case len(list) == 0:
		fmt.Println("Empty")
	case !exists(abspath):
		fmt.Println("Directory does not exist.")
	default:
		fmt.Println("Could not list contents.")
	}
}

// RefreshLog rewrites the history file.
//
// When searchForProjects is true, it rescans $HOME for marker-detected
// projects, merges them with existing log entries, and rewrites history. When
// false, it simply rewrites the existing entries (coalescing counts and
// re-sorting) without scanning the filesystem.
func RefreshLog(searchForProjects bool) {
	var projects []string
	if searchForProjects {
		projects = collectUserProjects()
	} else {
		projects = nil
	}

	logEntries := currentlyLoggedProjects()
	entries := collectEntries(projects, logEntries)

	sort.Sort(ByName(entries))
	sort.Sort(ByCount(entries))

	writeLogEntries(entries)
}

// ChangeDirectory resolves target to an absolute directory (using the file's
// parent directory if target is a file), prints the directory, logs it, and
// refreshes the history.
func ChangeDirectory(target string) {
	projectPath := findDirectory(target)
	fmt.Println(projectPath)
	addLogEntry(projectPath)
	RefreshLog(false)
}

// LogCurrentDirectory silently appends the current working directory to the
// history file. It is intended for shell hooks that track ordinary cd usage.
func LogCurrentDirectory() {
	addLogEntry(workingDir())
}

// addLogEntry appends a log entry for the given absolute path to the history
// file, skipping the home directory (home is always shown separately).
func addLogEntry(abspath string) {
	if abspath == homeDir() {
		return
	}

	ensureDirExists(filepath.Dir(expandPath(historyFile)))

	f, err := os.OpenFile(
		expandPath(historyFile),
		os.O_APPEND|os.O_CREATE|os.O_WRONLY,
		0600,
	)
	check(err)
	defer f.Close()

	entry := buildLogEntry(abspath)
	entry.WriteLogLine(f)
}

// findDirectory returns the absolute directory for the given path, using the
// containing directory if path is a file.
func findDirectory(path string) string {
	target := expandPath(path)

	if stat, err := os.Stat(target); err == nil && !stat.IsDir() {
		target = filepath.Dir(target)
	}

	return target
}

// collectUserProjects scans the user's home directory and returns a slice of
// absolute paths for directories considered "projects".
func collectUserProjects() []string {
	if debug {
		fmt.Println("Finding project directories...")
	}

	var projects []string
	seen := make(map[string]bool)

	_ = filepath.Walk(
		homeDir(),
		func(path string, info os.FileInfo, err error) error {
			switch {
			case err != nil:
				// Skip paths we can't stat
				return nil
			case !info.IsDir():
				// Skip non-directories
				return nil
			case isExcludedPath(path) || isExcludedBasename(info.Name()):
				// Do not recurse into excluded directories
				return filepath.SkipDir
			case isProject(path):
				// Record project and do not recurse further into it
				addProjectPath(&projects, seen, path)
				if markerMatches(path, ".git") {
					addGitSubmoduleProjects(&projects, seen, path)
				}
				return filepath.SkipDir
			case strings.HasPrefix(info.Name(), "."):
				// Do not recurse into dot-directories unless they are projects
				return filepath.SkipDir
			default:
				return nil
			}
		},
	)

	return projects
}

func addProjectPath(projects *[]string, seen map[string]bool, path string) {
	if seen[path] {
		return
	}

	seen[path] = true
	*projects = append(*projects, path)
}

func addGitSubmoduleProjects(projects *[]string, seen map[string]bool, path string) {
	for _, submodulePath := range gitSubmodulePaths(path) {
		if seen[submodulePath] ||
			isExcludedPath(submodulePath) ||
			isExcludedBasename(filepath.Base(submodulePath)) {
			continue
		}

		info, err := os.Stat(submodulePath)
		if err != nil || !info.IsDir() {
			continue
		}

		addProjectPath(projects, seen, submodulePath)
		if markerMatches(submodulePath, ".git") {
			addGitSubmoduleProjects(projects, seen, submodulePath)
		}
	}
}

func gitSubmodulePaths(path string) []string {
	fp, err := os.Open(filepath.Join(path, ".gitmodules"))
	if err != nil {
		return nil
	}
	defer fp.Close()

	var paths []string
	scanner := bufio.NewScanner(fp)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		key, value, ok := strings.Cut(line, "=")
		if !ok || strings.TrimSpace(key) != "path" {
			continue
		}

		submodulePath := strings.TrimSpace(value)
		if submodulePath == "" || filepath.IsAbs(submodulePath) {
			continue
		}

		paths = append(paths, filepath.Join(path, filepath.Clean(submodulePath)))
	}

	return paths
}

// currentlyLoggedProjects reads the history file and returns a map keyed by
// absolute path to aggregated LogEntry values.
func currentlyLoggedProjects() map[string]LogEntry {
	entries := make(map[string]LogEntry)

	if !exists(historyFile) {
		return entries
	}

	fp, err := os.Open(historyFile)
	check(err)
	defer fp.Close()

	scanner := bufio.NewScanner(fp)
	for scanner.Scan() {
		fields := strings.Split(scanner.Text(), ",")
		if len(fields) < 4 {
			continue
		}

		count, err := strconv.Atoi(fields[0])
		if err != nil {
			continue
		}

		abspath := fields[1]
		name := fields[2]
		path := fields[3]

		if existing, ok := entries[abspath]; ok {
			count += existing.Count
		}

		entries[abspath] = LogEntry{
			Count:   count,
			AbsPath: abspath,
			Name:    name,
			Path:    path,
		}
	}

	if err := scanner.Err(); err != nil {
		check(err)
	}
	return entries
}

// collectEntries merges discovered project paths with existing log entries,
// returning a flat slice of LogEntry values.
func collectEntries(foundPaths []string, currEntries map[string]LogEntry) (entries []LogEntry) {
	// Keep all current entries
	for _, entry := range currEntries {
		entries = append(entries, entry)
	}

	// Add new entries
	for _, abspath := range foundPaths {
		if _, alreadyLogged := currEntries[abspath]; !alreadyLogged {
			entries = append(entries, buildLogEntry(abspath))
		}
	}

	return entries
}

// writeLogEntries overwrites the history file with the given entries.
func writeLogEntries(entries []LogEntry) {
	if debug {
		fmt.Println("Refreshing project listing...")
	}

	f, err := os.Create(expandPath(historyFile))
	check(err)
	defer f.Close()

	for _, entry := range entries {
		entry.WriteLogLine(f)
	}

	if debug {
		fmt.Println("Completed at", time.Now().Format("Mon Jan 2 15:04:05 MST 2006"))
	}
}

// LogEntry represents a single row in the pd history file: a count, abs path,
// and human-readable project label pieces.
type LogEntry struct {
	Count   int
	AbsPath string
	Name    string
	Path    string
}

// LabelFormatted returns the colored label used in the FZF interface.
func (e LogEntry) LabelFormatted() string {
	name := aurora.Blue(e.Name).String()
	path := aurora.Gray(11, e.Path).String()
	return strings.Join([]string{name, path}, " ")
}

// Label returns the uncolored label used as a key into the listing index.
func (e LogEntry) Label() string {
	return strings.Join([]string{e.Name, e.Path}, " ")
}

// LogLine returns the CSV representation used in the history file.
func (e LogEntry) LogLine() string {
	return fmt.Sprintf("%d,%s,%s,%s\n", e.Count, e.AbsPath, e.Name, e.Path)
}

// WriteLogLine writes a single log line to file if the path still exists.
func (e LogEntry) WriteLogLine(file *os.File) {
	if exists(e.AbsPath) {
		if _, err := file.WriteString(e.LogLine()); err != nil {
			check(err)
		}
	}
}

// ByCount sorts LogEntry by descending Count, then by name (case-insensitive).
type ByCount []LogEntry

func (a ByCount) Len() int      { return len(a) }
func (a ByCount) Swap(i, j int) { a[i], a[j] = a[j], a[i] }
func (a ByCount) Less(i, j int) bool {
	if a[j].Count == a[i].Count {
		return strings.ToLower(a[i].Name) < strings.ToLower(a[j].Name)
	}
	return a[j].Count < a[i].Count
}

// ByName sorts LogEntry by Name (case-insensitive), then by Path.
type ByName []LogEntry

func (a ByName) Len() int      { return len(a) }
func (a ByName) Swap(i, j int) { a[i], a[j] = a[j], a[i] }
func (a ByName) Less(i, j int) bool {
	if a[i].Name == a[j].Name {
		return a[i].Path < a[j].Path
	}
	return strings.ToLower(a[i].Name) < strings.ToLower(a[j].Name)
}

// buildLogEntry constructs a single LogEntry from an absolute path.
func buildLogEntry(abspath string) LogEntry {
	path := strings.Replace(abspath, homeDir(), "~", -1)
	components := strings.Split(path, "/")
	last := len(components) - 1

	location := strings.Join(components[0:last], "/")

	return LogEntry{
		Count:   1,
		AbsPath: abspath,
		Name:    components[last],
		Path:    location,
	}
}

// buildHomeLogEntry returns a special LogEntry for the user's home directory,
// with a very large Count so it always sorts to the top.
func buildHomeLogEntry() LogEntry {
	return LogEntry{
		Count:   math.MaxInt32,
		AbsPath: homeDir(),
		Name:    "~",
		Path:    "",
	}
}

// projectLabelToAbsPath resolves a project label (as displayed in FZF) to an
// absolute path on disk.
func projectLabelToAbsPath(label string) string {
	label = strings.TrimSpace(label)

	if label == "~" {
		return homeDir()
	}

	// Labels under $HOME, e.g. "my-project ~Documents/projects"
	comps := strings.Split(label, " ~")
	if len(comps) > 1 {
		projName := comps[0]
		pathLabel := comps[1]
		path := fmt.Sprintf("%s/%s", homeDir(), pathLabel)
		return filepath.Join(path, projName)
	}

	// Labels starting at root, e.g. "my-project /usr/local/src"
	comps = strings.Split(label, " /")
	if len(comps) > 1 {
		projName := comps[0]
		pathLabel := comps[1]
		path := fmt.Sprintf("/%s", pathLabel)
		return filepath.Join(path, projName)
	}

	return ""
}

// searchListing creates the FZF listing source and index map from the current
// set of project log entries.
func searchListing(projectIndex map[string]LogEntry) (source.Source, map[string]string) {
	logEntries := make([]LogEntry, 0, len(projectIndex)+1)

	logEntries = append(logEntries, buildHomeLogEntry())
	for _, entry := range projectIndex {
		logEntries = append(logEntries, entry)
	}

	sort.Sort(ByName(logEntries))
	sort.Sort(ByCount(logEntries))

	listing := make([]string, 0, len(logEntries))
	index := make(map[string]string, len(logEntries))

	for _, entry := range logEntries {
		label := entry.Label()
		index[label] = entry.AbsPath
		listing = append(listing, entry.LabelFormatted())
	}

	return source.Slice(listing), index
}
