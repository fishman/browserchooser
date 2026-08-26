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

There are established link-picker tools; here is what pushed this one into
existence and keeps it small.

| tool | model | misses for us |
|------|-------|---------------|
| Browserosaurus / Browseratops | default-browser hook that pops a menu | Electron (Node + WebKit) runtime; macOS/Windows only, no Linux/Wayland |
| `@browsers`-style launcher scripts | rofi shell/JS scripts that parse `profiles.ini` | pulled in a pile of deps; miss **modern** Firefox profile-group profiles, which live in `Profile Groups` sqlite DBs, not `profiles.ini`; rofi sits on X11, not Wayland |
| linkquisition | Go browser chooser with routing rules | far larger codebase to audit for a one-purpose tool |

browserchooser keeps three properties the above split across them:

- **One static binary, no runtime stack.** Go + Fyne only; no Electron,
  Node, WebKit, or script interpreter to ship or keep patched.
- **Modern Firefox profiles work.** Real names are read from the `Profile
  Groups` sqlite DBs (via the `sqlite3` CLI), not `profiles.ini`, so the
  profiles that exist after Firefox 137 resolve to their actual names.
- **Cross-platform and Wayland-native.** Fyne draws directly on Wayland (and
  X11), and the same binary runs on Linux, macOS, and Windows. The dedicated
  pickers above are each locked to one OS.

The codebase stays small enough to read end to end, which matters for a tool
that sits at the default-browser boundary.

## Theme

Nord palette. The scheme is set in `config.toml`: `theme = "auto"` (default)
follows the system's dark/light, while `theme = "light"` or `theme = "dark"`
forces a variant.

## License

MIT. All dependencies are permissive (MIT / BSD / Apache-2.0), so the project
carries no copyleft obligation.
