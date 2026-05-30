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
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
)

const (
	launchdLabel   = "com.local.pd-refresh"
	cronMarker     = "# pd-refresh"
	cronSchedule   = "*/15 * * * *"
	launchdPlistPath = "Library/LaunchAgents/com.local.pd-refresh.plist"
)

func setupRefresh() (string, error) {
	binary, err := os.Executable()
	if err != nil {
		return "", fmt.Errorf("could not resolve binary path: %w", err)
	}

	switch runtime.GOOS {
	case "darwin":
		return setupRefreshLaunchd(binary)
	default:
		return setupRefreshCron(binary)
	}
}

func setupRefreshLaunchd(binary string) (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}

	plist := filepath.Join(home, launchdPlistPath)

	if err := os.MkdirAll(filepath.Dir(plist), 0o755); err != nil {
		return "", err
	}

	if err := os.WriteFile(plist, []byte(launchdPlistContent(binary)), 0o644); err != nil {
		return "", err
	}

	uid := fmt.Sprintf("gui/%d", os.Getuid())

	// Unload if already running; ignore failure (not loaded is fine)
	_ = exec.Command("launchctl", "bootout", uid, plist).Run()

	if out, err := exec.Command("launchctl", "bootstrap", uid, plist).CombinedOutput(); err != nil {
		return "", fmt.Errorf("launchctl bootstrap failed: %w\n%s", err, out)
	}

	return fmt.Sprintf("Installed pd refresh schedule via launchd (%s).\nRuns every 15 minutes.", plist), nil
}

func setupRefreshCron(binary string) (string, error) {
	out, err := exec.Command("crontab", "-l").Output()
	if err != nil {
		// crontab -l exits non-zero when there is no crontab yet; treat as empty
		out = []byte{}
	}

	lines := strings.Split(strings.TrimRight(string(out), "\n"), "\n")
	filtered := make([]string, 0, len(lines))
	for _, line := range lines {
		if line == "" || strings.HasSuffix(line, cronMarker) {
			continue
		}
		filtered = append(filtered, line)
	}

	entry := fmt.Sprintf("%s %s --pd-refresh %s", cronSchedule, binary, cronMarker)
	filtered = append(filtered, entry, "")

	cmd := exec.Command("crontab", "-")
	cmd.Stdin = strings.NewReader(strings.Join(filtered, "\n"))
	if out, err := cmd.CombinedOutput(); err != nil {
		return "", fmt.Errorf("crontab write failed: %w\n%s", err, out)
	}

	return fmt.Sprintf("Installed pd refresh schedule via cron (%s).", entry), nil
}

func launchdPlistContent(binary string) string {
	return fmt.Sprintf(`<?xml version="1.0" encoding="UTF-8"?>
<!DOCTYPE plist PUBLIC "-//Apple//DTD PLIST 1.0//EN"
  "http://www.apple.com/DTDs/PropertyList-1.0.dtd">
<plist version="1.0">
<dict>
  <key>Label</key>
  <string>%s</string>
  <key>ProgramArguments</key>
  <array>
    <string>%s</string>
    <string>--pd-refresh</string>
  </array>
  <key>StartInterval</key>
  <integer>900</integer>
  <key>StandardOutPath</key>
  <string>/tmp/pd-refresh.log</string>
  <key>StandardErrorPath</key>
  <string>/tmp/pd-refresh.log</string>
</dict>
</plist>
`, launchdLabel, binary)
}
