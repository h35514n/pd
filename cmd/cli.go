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
	"fmt"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

const (
	defaultHistoryFilename = "history"

	configFileName = "config"
	configFileType = "yaml"

	configKeyHistoryFile    = "history_filepath"
	configKeyDebug          = "debug"
	configKeyExcludes       = "excludes"
	configKeyProjectMarkers = "project_markers"
)

var (
	historyFile             string
	excludePathPatterns     []string
	excludeBasenamePatterns []string
	projectMarkers          []string
	debug                   bool

	dirStackPattern = regexp.MustCompile(`\A[-+][0-9]*\z`)
)

var defaultProjectMarkers = []string{
	".git",
	".projectile",
	"Makefile",
	"go.mod",
	"package.json",
	"pyproject.toml",
	"Cargo.toml",
	"src/",
	"lib/",
}

// defaultExcludes is a flat list of patterns excluded from project / home
// walks. Entries containing '/' are treated as paths (with ~/ expansion);
// entries without '/' are treated as basename globs (filepath.Match).
var defaultExcludes = []string{
	// Paths
	"~/Library",
	"~/.Trash",
	"~/.cache",
	"~/.local/share/Trash",
	"~/.local/share/containers",
	"~/.local/share/flatpak",
	"~/.var",
	"~/.npm",
	"~/.cargo/registry",
	"~/.rustup",
	"~/.gradle",
	"~/.m2/repository",
	"~/.gem",
	"~/.bundle",
	"~/.pyenv",
	"~/.rbenv",
	"~/.nvm",
	"~/.mozilla",
	"~/.thunderbird",

	// Basename globs
	".git",
	".github",
	".svn",
	".hg",
	".bzr",
	"node_modules",
	"__pycache__",
	".venv",
	"vendor",
}

var help = `
p/d

Usage:

  pd [directory name]

Intended to be used in tandem with cd as follows:

  cd $(pd ~/Documents)

Given a file path, print its absolute form (resolving symlinks) and save to
history. If a path to a non-directory is given, use its containing directory.
The given path can be absolute or relative, but its history will contain the
absolute path to the final directory.

Examples:
  pd my-project
  pd ~/my-other-project
  pd ./projects/my-project/some-file.txt

Given a position on the directory stack, no-op. Print that back out to leave the
behavior of cd unchanged.

Examples:
  pd -2
  pd +1
  pd -

Given no arguments, open FZF to allow fuzzy-selecting a directory to cd into.

Given --pd-log-cwd, silently log the current working directory. This is intended
for shell hooks that track ordinary cd, pushd, and popd directory changes.

Given --pd-setup, install the recommended shell setup for zsh or bash. Pass zsh
or bash explicitly to override automatic shell detection.
`

var rootCmd = &cobra.Command{
	Use:                "pd",
	Short:              "A project / directory manager and FZF-powered fuzzy-selector.",
	DisableFlagParsing: true,
	SilenceErrors:      true,
	SilenceUsage:       true,
	Args:               cobra.ArbitraryArgs,
	Run: func(cmd *cobra.Command, args []string) {
		target := strings.TrimSpace(strings.Join(args, " "))
		setupShellArg, isSetup, setupErr := parseSetupTarget(target)

		switch {
		case len(target) == 0:
			// No arguments → fuzzy-select a directory
			SelectProject()

		case target == "--help":
			// Explicit help text
			fmt.Println(help)

		case strings.HasPrefix(target, "--fzf-preview"):
			// Preview for FZF
			label := strings.TrimSpace(strings.TrimPrefix(target, "--fzf-preview"))
			FzfPreview(label)

		case target == "--pd-refresh":
			// Force a full refresh of project listing
			RefreshLog(true)

		case strings.HasPrefix(target, "--home-picker"):
			// Fallback picker: every directory under $HOME
			query := strings.TrimSpace(strings.TrimPrefix(target, "--home-picker"))
			HomePicker(query)

		case target == "--pd-log-cwd":
			// Silently log the current working directory for shell hooks
			LogCurrentDirectory()

		case isSetup:
			// Install managed shell setup block
			check(setupErr)
			rcPath, err := setupShell(setupShellArg)
			check(err)
			fmt.Println("Installed pd shell setup in", rcPath)
			fmt.Println("Restart your shell or source the file to activate it.")

		case dirStackPattern.MatchString(target):
			// Dir stack position → pass through unchanged
			fmt.Println(target)

		default:
			// Treat as directory path to change into
			ChangeDirectory(target)
		}
	},
}

func init() {
	cobra.OnInitialize(initConfig)
}

// Execute is called by main.main().
func Execute() {
	if err := rootCmd.Execute(); err != nil {
		check(err)
	}
}

// initConfig reads config file and environment variables, then initializes
// global configuration (historyFile, excludes, debug).
func initConfig() {
	cfgDir := configDir()
	statePath := stateDir()

	// Config defaults
	viper.SetDefault(configKeyHistoryFile, filepath.Join(statePath, defaultHistoryFilename))
	viper.SetDefault(configKeyDebug, false)
	viper.SetDefault(configKeyProjectMarkers, defaultProjectMarkers)

	// Config file
	viper.AddConfigPath(cfgDir)
	viper.SetConfigName(configFileName)
	viper.SetConfigType(configFileType)

	err := viper.ReadInConfig()
	checkConfigFile(err)

	// Effective configuration
	debug = viper.GetBool(configKeyDebug)
	historyFile = expandPath(viper.GetString(configKeyHistoryFile))
	excludePathPatterns, excludeBasenamePatterns = classifyExcludes(
		mergeExcludes(viper.GetStringSlice(configKeyExcludes)),
	)
	projectMarkers = cleanProjectMarkers(viper.GetStringSlice(configKeyProjectMarkers))
}
