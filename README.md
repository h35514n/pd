p/d
===

A project / directory manager and
[FZF](https://github.com/junegunn/fzf)-powered fuzzy-selector.

Use `pd` in tandem with `cd` to change directories using fuzzy-search,
search for projects in your home directory, and keep track of your
most-visited `cd` targets.

It's written in [Go](https://golang.org/doc), and it *zooms*.

[![demo](./docs/assets/demo.gif)](https://asciinema.org/a/330647)

Quick start
-----------

Clone and install:

``` sh
go install
```

Install the recommended shell integration:

``` sh
pd --pd-setup
```

`pd --pd-setup` detects zsh or bash from `$SHELL`, writes a managed
block to your shell rc file, and installs:

- a key binding for fuzzy selection: <kbd>Ctrl</kbd>+<kbd>h</kbd>
- a directory-change hook so ordinary `cd`, `pushd`, and `popd` targets
  are logged and ranked

Restart your shell, or source the file named by the setup command.

Usage and setup
---------------

See [docs/usage.md](docs/usage.md) for command usage, shell setup
details, fallback setup options, project discovery config, and
cron/launchd refresh examples.

License
-------

Apache

Acknowledgements
----------------

p/d is written in [Go](https://golang.org/doc) based on a prototype in
[Ruby](https://ruby-doc.org/). It builds upon prior art by
[junegunn](https://github.com/junegunn)
([fzf](https://github.com/junegunn/fzf)) and
[b4b4r07](https://github.com/b4b4r07)
([go-finder](https://github.com/b4b4r07/go-finder)).
