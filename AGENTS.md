# BrowserChooser - agent working rules

A single-window Fyne Go app. Keep it small: no speculative layers, no
config for a value that never changes, delete over add. Anything beyond the
minimum code that works needs a reason, not a placeholder.

## Style

- Go-idiomatic, gofmt-clean, stdlib first, clear over clever. A concept
  exists once.
- Self-documenting names preferred; only non-obvious constraints, edge
  cases, or critical tradeoffs get a comment. Never explain basic syntax.
- No unnecessary comments. ASCII only - no unicode dashes or curly quotes
  in code or prose.
- No commit carries an AI marker; the human author answers for every line,
  whether or not AI drafted it. Doc/spec commits carry
  `Co-Authored-By: <model that drafted them>`; review stays with the human
  either way.

## Commits

- Conventional Commits: `type(scope): subject`, brief lowercase imperative.
  One logical change per commit.

## Testing

- Treat output like firmware - assume it fails in production until exercised.
- Non-trivial logic leaves ONE runnable check (`go test`, no framework).
- Regression fixes fail first (TDD): write the failing test, confirm it
  FAILS against current code, then fix. Never commit a regression fix
  without that red run - a test that passed before the fix proves nothing.

## Performance (hard rule)

- Never walk the icon tree. /usr/share/icons is ~2.6GB / ~500k files on a
  real system. iconResource stats only the common pixmaps/hicolor app dirs
  and caches per name; a miss returns nil. Keep it that way.
