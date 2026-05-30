p/d usage and setup
===================

Command usage
-------------

``` sh
pd [directory name]
```

Given a file path, `pd` prints its absolute form and saves it to
history. If the path is a file, `pd` uses its containing directory.

``` sh
pd ~/Documents/projects/my-project
pd ~/my-other-project
pd ./projects/my-project/some-file.txt
```

Given a position on the directory stack, `pd` prints it back unchanged
so shell wrappers can preserve normal `cd` behavior.

``` sh
pd -2
pd +1
pd -
```

Given no arguments, `pd` opens FZF to select a directory.

``` sh
pd
```

`pd --pd-log-cwd` silently logs the current working directory. It is
intended for shell hooks and prints nothing.

`pd --pd-refresh` rescans `$HOME` for Git, Projectile, and
marker-detected projects, merges them with visit history, and rewrites
the history file.

Automated shell setup
---------------------

Run:

``` sh
pd --pd-setup-shell
```

The setup command detects zsh or bash from `$SHELL`. You can override
detection:

``` sh
pd --pd-setup-shell zsh
pd --pd-setup-shell bash
```

The command writes a managed block between these markers:

``` sh
# >>> pd setup >>>
# <<< pd setup <<<
```

Re-running setup replaces that block instead of appending duplicates.
Restart your shell, or source the file named by the setup command.

### Zsh setup block

`pd --pd-setup-shell zsh` writes this block to `~/.zshrc`:

``` sh
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
```

This binds fuzzy selection to <kbd>Ctrl</kbd>+<kbd>h</kbd> and
logs ordinary directory changes through `chpwd`.

### Bash setup block

`pd --pd-setup-shell bash` writes this block to `~/.bashrc`:

``` sh
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
```

This binds fuzzy selection to
<kbd>Ctrl</kbd>+<kbd>h</kbd> and
logs ordinary directory changes through `PROMPT_COMMAND`.

Fallback setup
--------------

If you want a portable full-feature setup for another shell, wrap `cd`
to delegate to `pd`. This shadows the shell builtin, but it preserves
fuzzy selection, arbitrary directory logging, and directory ranking.

``` sh
# ~/.bashrc, ~/.zshrc, or another shell startup file

cd() {
    builtin cd "$(pd "$1")" || return
}
```

Selector-only setup
-------------------

If you only bind `pd` to a key without a directory-change hook or `cd`
wrapper, fuzzy selection works but arbitrary `cd` targets are not logged
or counted.

``` sh
# Example: zsh selector-only binding
pd-switch() {
  local dir
  zle -I
  dir="$(pd)"
  [[ -n "$dir" ]] && builtin cd "$dir"
  zle reset-prompt
}
zle -N pd-switch
bindkey '^h' pd-switch
```

Automated refresh setup
-----------------------

Run:

``` sh
pd --pd-setup-refresh
```

On macOS this writes a launchd plist to
`~/Library/LaunchAgents/com.local.pd-refresh.plist` and registers it with
`launchctl`. On other platforms it adds an entry to your crontab.

Both run `pd --pd-refresh` every 15 minutes. Re-running the command replaces
the existing schedule rather than adding a duplicate.

To stop and unregister the launchd agent on macOS:

``` sh
launchctl bootout gui/$(id -u) ~/Library/LaunchAgents/com.local.pd-refresh.plist
```

Project discovery
-----------------

Project markers are configured in `~/.config/pd/config.yaml` with the
`project_markers` key. A directory is considered a project when it
contains any marker. Markers ending in `/` must be directories; other
markers can be files or directories.

Default markers:

``` yaml
project_markers:
  - .git
  - .projectile
  - Makefile
  - go.mod
  - package.json
  - pyproject.toml
  - Cargo.toml
  - src/
  - lib/
```

Add your own markers to tune discovery:

``` yaml
project_markers:
  - .git
  - .projectile
  - Makefile
  - go.mod
  - package.json
  - pyproject.toml
  - Cargo.toml
  - src/
  - lib/
  - config.toml
  - env/
```

`excludes` prevents directories from being walked. Each entry is
auto-classified: entries containing `/` are matched as paths (with `~/`
expansion); entries with no `/` are matched as `filepath.Match` glob
patterns against each directory's name, anywhere in the tree.

Defaults cover common macOS/Linux system locations (`~/Library`,
`~/.Trash`, `~/.cache`, and package manager caches like `~/.npm`,
`~/.cargo/registry`, `~/.gradle`, `~/.m2/repository`) and noisy
directory names (`node_modules`, `.git`, `__pycache__`, `vendor`,
and similar). User entries are additive — they extend rather than
replace the defaults.

Examples:

``` yaml
excludes:
  - ~/Library            # path: exact match + descendants
  - ~/.dotfiles/cache    # path: matches this dir and below
  - ~/Code/*/build       # path glob: any project's build/ dir
  - node_modules         # basename: any dir named node_modules
  - "*.egg-info"         # basename glob
```

Periodic refresh
----------------

Running `pd --pd-refresh` periodically keeps the directory list current
as projects are created or cloned. Use `pd --pd-setup-refresh` to install
a schedule automatically, or set one up manually as shown below.

### cron

Add an entry with `crontab -e`:

``` cron
# Every 15 minutes
*/15 * * * * /usr/local/bin/pd --pd-refresh
```

### launchd

On macOS, save a plist to
`~/Library/LaunchAgents/com.local.pd-refresh.plist`:

``` xml
<?xml version="1.0" encoding="UTF-8"?>
<!DOCTYPE plist PUBLIC "-//Apple//DTD PLIST 1.0//EN"
  "http://www.apple.com/DTDs/PropertyList-1.0.dtd">
<plist version="1.0">
<dict>
  <key>Label</key>
  <string>com.local.pd-refresh</string>
  <key>ProgramArguments</key>
  <array>
    <string>/usr/local/bin/pd</string>
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
```

Register it with launchd:

``` sh
launchctl bootstrap gui/$(id -u) ~/Library/LaunchAgents/com.local.pd-refresh.plist
```

Stop and unregister it:

``` sh
launchctl bootout gui/$(id -u) ~/Library/LaunchAgents/com.local.pd-refresh.plist
```

Find the `pd` binary path with `which pd` and substitute it above if it
differs from `/usr/local/bin/pd`.
