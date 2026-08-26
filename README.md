# Browser Chooser

A small rofi-style browser selector for Linux, macOS, and Windows. Point it at
a URL and pick which browser opens it, or copy the link to the clipboard and
share it. Built with [Fyne](https://fyne.io) in Go.

## Design principles

browserchooser stays deliberately small:

- **Simple tool.** One job, one window: pick a browser for a link.
- **Few dependencies.** Go + Fyne and nothing else at runtime; no Electron,
  Node, or script interpreter to ship or keep patched. Detecting profiles
  shells out to tools already on the system (`sqlite3`) instead of pulling in
  bindings.
- **Easy-to-read config and link matcher.** All configuration lives in one
  plain `config.toml`: theme, which browser profiles to list, and routing
  rules written as short, declarative expressions.
- **Frecency without a dialog.** Match ordering is derived from how often and
  how recently each browser was used - no configuration dialog, no preference
  pane.
- **No complex behaviour.** No daemon, no background state beyond a usage
  counter, no plugins.

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

Route a URL to a specific browser without opening the picker. Rules live in
`config.toml` as `[[rules]]`. Each rule pairs an
[expr](https://expr-lang.github.io/) expression that must evaluate to a bool
with the browser id to open:

```toml
[[rules]]
expr = 'link contains "github.com"'
browser = "firefox"
```

- `link` is the URL being opened.
- Expressions are ordinary
  [expr](https://expr-lang.github.io/lang/) booleans over `link`: string
  operators (`contains`, `startsWith`, `endsWith`, `matches` for regex),
  `in`, comparisons, and `and` / `or` / `not`. The full expression language
  applies.
- Rules are evaluated **top to bottom; the first match wins** and opens its
  browser immediately. If nothing matches, or there are no rules, the picker
  appears as usual.

```toml
[[rules]]
expr = 'link matches "^(https?://)?github\\.com/"'
browser = "firefox-work"   # a modern Firefox profile by real name

[[rules]]
expr = 'link contains "youtube.com" and link startsWith "https://"'
browser = "google-chrome"
```

**Browser ids** used in `browser`:

| browser kind | id | example |
|--------------|----|---------|
| standalone binary | executable basename | `firefox`, `google-chrome` |
| Firefox profile (classic or modern) | `firefox-<name>` | `firefox-work`, `firefox-profile-4` |
| Chrome profile | `chrome-<dir>` | `chrome-default`, `chrome-profile-1` |

Modern Firefox ids come from the real profile name stored in the `Profile
Groups` sqlite DBs; classic and Chrome ids use the sanitized profile/directory
name. Ids are lowercased with non-alphanumeric characters turned into `-`.

## Settings

Optional `config.toml` in the user config dir
(`$XDG_CONFIG_HOME/browserchooser/config.toml`).

| key | effect |
|-----|--------|
| `theme` | color scheme: `auto` (default) follows the system, `light` and `dark` force a variant |
| `firefox.profiles` | list every Firefox profile as its own selection, launched with `-profile`. **On by default**; set to `false` to disable. Covers classic `profiles.ini` profiles and **modern profile-group profiles**, whose real names are read from the `Profile Groups` sqlite DBs (via the `sqlite3` CLI; falls back to the directory name when that is unavailable). Works on Linux, macOS, and Windows |
| `chrome.profiles` | list every Google Chrome profile (from `Local State`) as its own selection, launched with `--profile-directory`. **On by default**; set to `false` to disable |
| `rules` | an array of `expr` + `browser` pairs routing URLs; see Rules |

```toml
theme = "dark"

[firefox]
profiles = false

[chrome]
profiles = true

[[rules]]
expr = 'link contains "github.com"'
browser = "firefox"
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

## Why not an existing browser chooser?

Existing link-picker tools, and the differences that matter:

| tool | model | misses for us |
|------|-------|---------------|
| Browserosaurus / Browseratops | default-browser hook that pops a menu | Electron (Node + WebKit) runtime; macOS/Windows only, no Linux/Wayland |
| Browsers (`browsers.software`) | Rust context-menu picker on the druid GUI toolkit (GTK3 on Linux) | hard to extend; the toolkit is the older GTK3, not GTK4; heavier to build than a single Go binary |
| linkquisition | Go browser chooser with routing rules | far larger codebase to audit for a one-purpose tool |

browserchooser keeps three properties the above split across them:

- **One static binary, no runtime stack.** Go + Fyne only; no Electron,
  Node, WebKit, or script interpreter to ship or keep patched.
- **Modern Firefox profiles work.** Real names are read from the `Profile
  Groups` sqlite DBs (via the `sqlite3` CLI), not `profiles.ini`, so the
  profiles that exist after Firefox 137 resolve to their actual names.
- **Cross-platform and Wayland-native.** The same binary runs on Linux, macOS,
  and Windows. Fyne renders its own widgets on Wayland (and X11), rather than
  wrapping a legacy native toolkit like the GTK3-backed Browsers or the
  Electron runtime of the Browserosaurus forks.

## Theme

Nord palette. The scheme is set in `config.toml`: `theme = "auto"` (default)
follows the system's dark/light, while `theme = "light"` or `theme = "dark"`
forces a variant.

## License

MIT. All dependencies are permissive (MIT / BSD / Apache-2.0), so the project
carries no copyleft obligation.
