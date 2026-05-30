p/d
===

`pd` is a frecency-ranked project switcher for the terminal. It tracks every
directory you `cd` into, auto-discovers projects under `$HOME`, and presents
the whole list in an FZF fuzzy-finder you can invoke without leaving your
current shell.

Directories you visit rise to the top of the list via frecency ranking (visit
frequency × recency decay). The list is seeded by auto-discovery of projects
under `$HOME` and grows passively as you work through a lightweight shell hook.
The picker shows a rich preview for each entry: rendered README when available
in any of seven formats, directory tree or file listing otherwise. A fallback
home-directory search is a keypress away
inside the picker.

Project discovery, exclusion rules, and frecency tuning are
[configurable](docs/usage.md#project-discovery). Config and state paths conform
to the XDG Base Directory Specification. Supports scheduled background refreshes
via cron or launchd — see [docs/usage.md](docs/usage.md) for setup details and
fallback shell integration options.

[![demo](./docs/assets/demo.gif)](https://asciinema.org/a/330647)

Quick start
-----------

Clone and build:

``` sh
go build [-o <target path in your PATH>]
```

Install the recommended shell integration:

``` sh
pd --pd-setup
```

Restart your shell, or source the file named by the setup command.

Usage and setup
---------------

See [docs/usage.md](docs/usage.md) for command usage, shell setup
details, fallback setup options, project discovery config, and
cron/launchd refresh examples.

License
-------

Apache
