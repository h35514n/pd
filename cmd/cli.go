/*
Copyright © 2023 Jake Romer <jmromer@tensorconclave.com>

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

	configKeyHistoryFile = "history_filepath"
	configKeyDebug       = "debug"
	configKeySkipDirs    = "skip_dirs"
)

var (
	historyFile string
	skipDirs    map[string]bool
	debug       bool

	dirStackPattern = regexp.MustCompile(`\A[-+][0-9]*\z`)
)

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
`

var rootCmd = &cobra.Command{
	Use:   "pd",
	Short: "A project / directory manager and FZF-powered fuzzy-selector.",
	DisableFlagParsing: true,
	Args: cobra.ArbitraryArgs,
	Run: func(cmd *cobra.Command, args []string) {
		target := strings.TrimSpace(strings.Join(args, " "))

		switch {
		case len(target) == 0:
			// No arguments → fuzzy-select a directory
			SelectProject()

		case target == "--help":
			// Explicit help text
			fmt.Println(help)

		case strings.HasPrefix(target, "--fzf-preview"):
			// Preview for FZF
			label := strings.TrimPrefix(target, "--fzf-preview")
			FzfPreview(label)

		case target == "--pd-refresh":
			// Force a full refresh of project listing
			RefreshLog(true)

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
// global configuration (historyFile, skipDirs, debug).
func initConfig() {
	cfgDir := configDir()
	stateDir := stateDir()

	// Config defaults
	viper.SetDefault(configKeyHistoryFile, filepath.Join(stateDir, defaultHistoryFilename))
	viper.SetDefault(configKeyDebug, false)
	viper.SetDefault(configKeySkipDirs, []string{"~/Library/"})

	// Config file
	viper.AddConfigPath(cfgDir)
	viper.SetConfigName(configFileName)
	viper.SetConfigType(configFileType)

	err := viper.ReadInConfig()
	checkConfigFile(err)

	// Effective configuration
	debug = viper.GetBool(configKeyDebug)
	historyFile = expandPath(viper.GetString(configKeyHistoryFile))
	skipDirs = toSkipDirSet(viper.GetStringSlice(configKeySkipDirs))
}
