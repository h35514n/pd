p/d
===

A project / directory manager and [FZF][]-powered fuzzy-selector.

Use `pd` in tandem with `cd` to change directories using fuzzy-search, search
for version-controlled projects in your home directory, and keep track of your
most-visited `cd` targets.

It's written in [Go][], and it _zooms_.

[![asciicast](https://asciinema.org/a/330578.svg)](https://asciinema.org/a/330578)

<!-- markdown-toc start - Don't edit this section. Run M-x markdown-toc-refresh-toc -->
**Contents**

- [Recommended setup](#recommended-setup)
- [Usage](#usage)
- [Installation](#installation)
- [License](#license)
- [Acknowledgements](#acknowledgements)

<!-- markdown-toc end -->

Recommended setup
-----------------

<details>
<summary><strong>Zsh: full-feature without shadowing cd</strong></summary>

**Zsh** (preferred): bind `pd` to a ZLE widget and use a `chpwd` hook to log
ordinary directory changes without overriding `cd`:

```sh
# ~/.zshrc

_pd_log_cwd() {
  if [[ -n "${_pd_skip_log_once:-}" ]]; then
    _pd_skip_log_once=
    return
  fi

  pd --pd-log-cwd >/dev/null 2>&1
}

autoload -Uz add-zsh-hook
add-zsh-hook chpwd _pd_log_cwd

pd-switch() {
  local dir
  local oldpwd="$PWD"
  zle -I               # suspend ZLE input handling
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
```
</details>

<details>
<summary><strong>Bash: full-feature without shadowing cd</strong></summary>

**Bash**: bind `pd` with Readline and use `PROMPT_COMMAND` to log ordinary
directory changes without overriding `cd`:

```sh
# ~/.bashrc

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

PROMPT_COMMAND="_pd_log_cwd${PROMPT_COMMAND:+;$PROMPT_COMMAND}"

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
```
</details>

<details>
<summary><strong>Portable full-feature fallback</strong></summary>

**Portable fallback** (any shell): wrap `cd` to delegate to `pd`. This is more
intrusive because it shadows the builtin, but it gives the full p/d experience
in shells without a convenient directory-change hook:

```sh
# ~/.bashrc or ~/.zshrc

# wrap built-in cd to:
# 1. fuzzy-select a directory to visit when given no argument
# 2. retain built-in dir-stack-related behavior when given a -/+ numeric arg
# 3. log a directory visit when given any other arg

cd() {
    builtin cd "$(pd "$1")" || return
}
```

</details>

<details>
<summary><strong>Selector-only setup</strong></summary>

If you only bind `pd` to a key without a shell hook or `cd` wrapper, fuzzy
selection works but arbitrary `cd` targets are not logged or counted.

```sh
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
</details>


Usage
-----
```
pd [directory name]
```

Given a file path, print its absolute form (resolving symlinks) and save to
history. If a path to a non-directory is given, use its containing
directory instead.

Examples:

```sh
pd ~/Documents/projects/my-project
pd ~/my-other-project
pd ./projects/my-project/some-file.txt
```


Given a position on the directory stack, no-op. Print that back out to leave the
behavior of cd unchanged.

Examples:

``` sh
pd -2
pd +1
pd -
```

Given no arguments, open FZF to allow fuzzy-selecting a directory to cd into.

Given `--pd-log-cwd`, silently add the current working directory to the history
file. This is intended for shell hooks; it prints nothing.

Given `--pd-refresh`, rescan $HOME for Git, Projectile, and marker-detected
projects, merge with visit history, and rewrite the history file. Use this to
pre-warm the directory list after installing or when new projects have been
added.

Project markers are configured in `~/.config/pd/config.yaml` with the
`project_markers` key. A directory is considered a project when it contains any
marker. Markers ending in `/` must be directories; other markers can be files or
directories. The default markers are:

```yaml
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

```yaml
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

`skip_dirs` prevents noisy system and cache directories from being scanned. By
default, `pd` skips common macOS and Linux locations such as `~/Library`,
`~/.Trash`, `~/.cache`, `~/.local/share/Trash`, `~/.var`, and package manager
caches like `~/.npm`, `~/.cargo/registry`, `~/.gradle`, and `~/.m2/repository`.
Set `skip_dirs` in your config to add more directories to skip.

Running `pd --pd-refresh` periodically keeps the directory list current as
projects are created or cloned. Two common approaches:

<details>
<summary><strong>cron</strong></summary>

**cron** — add an entry with `crontab -e`:

```cron
# Every 15 minutes
*/15 * * * * /usr/local/bin/pd --pd-refresh
```
</details>

<details>
<summary><strong>launchd</strong></summary>

**launchd** (preferred over cron on macOS) — save a plist to

`~/Library/LaunchAgents/com.local.pd-refresh.plist`:

```xml
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


Then register it with launchd:

```sh
launchctl bootstrap gui/$(id -u) ~/Library/LaunchAgents/com.local.pd-refresh.plist
```

To stop and unregister it:

```sh
launchctl bootout gui/$(id -u) ~/Library/LaunchAgents/com.local.pd-refresh.plist
```

</details>

**Note:** Find the `pd` binary path with `which pd` and substitute it above if
it differs from `/usr/local/bin/pd` (e.g. `/opt/homebrew/bin/pd` on Apple
Silicon, or `~/go/bin/pd` if installed via `go install`).

Installation
------------

Clone and build with `go build && go install`.

License
-------

Apache

Acknowledgements
----------------

p/d is written in [Go][] based on a prototype in [Ruby][].
It builds upon prior art by [junegunn][] ([fzf][]) and [b4b4r07][]
([go-finder][]).

[b4b4r07]: https://github.com/b4b4r07
[fzf]: https://github.com/junegunn/fzf
[go-finder]: https://github.com/b4b4r07/go-finder
[go]: https://golang.org/doc
[junegunn]: https://github.com/junegunn
[ruby]: https://ruby-doc.org/
