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
	"path/filepath"
	"strings"

	homedir "github.com/mitchellh/go-homedir"
)

const (
	setupMarkerStart = "# >>> pd setup >>>"
	setupMarkerEnd   = "# <<< pd setup <<<"
)

func setupShell(shell string) (string, error) {
	shell, err := normalizeSetupShell(shell)
	if err != nil {
		return "", err
	}

	home, err := homedir.Dir()
	if err != nil {
		return "", err
	}

	rcPath := filepath.Join(home, "."+shell+"rc")
	block, err := shellSetupBlock(shell)
	if err != nil {
		return "", err
	}

	current, err := os.ReadFile(rcPath)
	if err != nil && !os.IsNotExist(err) {
		return "", err
	}

	next := withManagedSetupBlock(string(current), block)
	if err := os.WriteFile(rcPath, []byte(next), 0o644); err != nil {
		return "", err
	}

	return rcPath, nil
}

func parseSetupTarget(target string) (string, bool, error) {
	fields := strings.Fields(target)
	if len(fields) == 0 || fields[0] != "--pd-setup-shell" {
		return "", false, nil
	}
	if len(fields) > 2 {
		return "", true, fmt.Errorf("usage: pd --pd-setup-shell [zsh|bash]")
	}
	if len(fields) == 2 {
		return fields[1], true, nil
	}
	return "", true, nil
}

func normalizeSetupShell(shell string) (string, error) {
	shell = strings.TrimSpace(shell)
	if shell == "" {
		shell = filepath.Base(os.Getenv("SHELL"))
	}

	switch shell {
	case "zsh", "bash":
		return shell, nil
	default:
		return "", fmt.Errorf("unsupported shell %q; expected zsh or bash", shell)
	}
}

func shellSetupBlock(shell string) (string, error) {
	switch shell {
	case "zsh":
		return strings.TrimSpace(`
# >>> pd setup >>>
_pd_log_cwd() {
  if [[ -n "${_pd_skip_log_once:-}" ]]; then
    _pd_skip_log_once=
    return
  fi

  pd --pd-log-cwd >/dev/null 2>&1
}

autoload -Uz add-zsh-hook
add-zsh-hook -d chpwd _pd_log_cwd 2>/dev/null
add-zsh-hook chpwd _pd_log_cwd

pd-switch() {
  local dir
  local oldpwd="$PWD"
  zle -I
  dir="$(pd)"
  if [[ -n "$dir" ]]; then
    _pd_skip_log_once=1
    if builtin cd "$dir"; then
      [[ "$PWD" == "$oldpwd" ]] && _pd_skip_log_once=
    else
      _pd_skip_log_once=
    fi
  fi
  zle reset-prompt
}
zle -N pd-switch
bindkey '^h' pd-switch
# <<< pd setup <<<
`) + "\n", nil
	case "bash":
		return strings.TrimSpace(`
# >>> pd setup >>>
_pd_last_pwd="$PWD"

_pd_log_cwd() {
  [[ "$PWD" == "$_pd_last_pwd" ]] && return
  _pd_last_pwd="$PWD"

  if [[ -n "${_pd_skip_log_once:-}" ]]; then
    _pd_skip_log_once=
    return
  fi

  pd --pd-log-cwd >/dev/null 2>&1
}

case ";$PROMPT_COMMAND;" in
  *";_pd_log_cwd;"*) ;;
  *) PROMPT_COMMAND="_pd_log_cwd${PROMPT_COMMAND:+;$PROMPT_COMMAND}" ;;
esac

pd-switch() {
  local dir
  local oldpwd="$PWD"
  dir="$(pd)"
  if [[ -n "$dir" ]]; then
    _pd_skip_log_once=1
    if builtin cd "$dir"; then
      [[ "$PWD" == "$oldpwd" ]] && _pd_skip_log_once=
    else
      _pd_skip_log_once=
    fi
  fi
}
bind -x '"\C-h": pd-switch'
# <<< pd setup <<<
`) + "\n", nil
	default:
		return "", fmt.Errorf("unsupported shell %q; expected zsh or bash", shell)
	}
}

func withManagedSetupBlock(content, block string) string {
	start := strings.Index(content, setupMarkerStart)
	end := strings.Index(content, setupMarkerEnd)
	if start >= 0 && end >= start {
		end += len(setupMarkerEnd)
		if end < len(content) && content[end] == '\n' {
			end++
		}
		return content[:start] + block + content[end:]
	}

	if content == "" {
		return block
	}
	if !strings.HasSuffix(content, "\n") {
		content += "\n"
	}
	return content + "\n" + block
}
