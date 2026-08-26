# Browser Chooser

A small rofi-style browser selector for Linux, macOS, and Windows. Point it at
a URL and pick which browser opens it, or copy the link to the clipboard and
share it. Built with [Fyne](https://fyne.io) in Go.

## What it does

A single fixed-size window with 7 slots:

| slot | purpose |
|------|---------|
| 1 | text input (fuzzy filter) |
| 2-6 | browser rows (up to 4, plus a pinned copy row) |
| 7 | Show QR code toggle |

Start typing to fuzzy-filter the list by name; rows are ranked by fuzzy match
then usage frequency/recency (frecency). Browsers are detected from `.desktop`
files on Linux, app bundles on macOS, and standard install paths on Windows.

## Keys

| key | action |
|-----|--------|
| `1`-`4` | open the numbered browser |
| `5` | copy the link to the clipboard |
| `Enter` | open the top match |
| `Ctrl+C` | copy the link and quit |
| `Esc` | quit |
| `F2` | toggle dark/light theme |

## Usage

```sh
browserchooser https://example.com
```

With no argument it opens as an interactive picker. Flags:

| flag | action |
|------|--------|
| `--set-default` | register as the default browser (Linux) |
| `--help` | show usage |

`--set-default` installs the desktop file to `~/.local/share/applications/`
and registers it as the default handler for `text/html`, `http`, `https`, and
`ftp`. The binary must be on `PATH` for it to be invoked with URLs.

## Rules

Optionally route URLs to a specific browser with `rules.json` in the
user config dir (`$XDG_CONFIG_HOME/browserchooser/rules.json`). Each rule is an
[expr](https://expr-lang.github.io/) expression over the variable `link` that
must evaluate to a bool, plus the browser id to use:

```json
[
  {"expr": "link contains \"github.com\"", "browser": "firefox"}
]
```

The first matching rule wins; a match opens its browser without showing the
picker. Browser ids come from the executable basename (e.g. `firefox`,
`google-chrome`).

## Settings

Optional `config.toml` in the user config dir
(`$XDG_CONFIG_HOME/browserchooser/config.toml`). Off by default.

| key | effect |
|-----|--------|
| `firefox.profiles` | list every Firefox profile (from `profiles.ini`) as its own selection, launched with `-profile` |

```toml
[firefox]
profiles = true
```

## Build

```sh
go build ./...
```

The binary is `browserchooser`. It needs a display; Fyne handles the backend
per platform.

## Install (Linux desktop file)

```sh
go build -o browserchooser .
install -Dm755 browserchooser ~/.local/bin/browserchooser
install -Dm644 dev.fishman.browserchooser.desktop \
  ~/.local/share/applications/dev.fishman.browserchooser.desktop
update-desktop-database ~/.local/share/applications
```

## Theme

Nord palette. Follows the system dark/light theme by default; `F2` toggles and
the override persists.

## License

MIT. All dependencies are permissive (MIT / BSD / Apache-2.0), so the project
carries no copyleft obligation.
