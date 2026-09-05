# jjj — jjui as a commit journal

`jjj` is this fork's binary: jjui plus a journal-first layer. Stock `jjui`
still builds from the same tree (`go build ./cmd/jjui`) and is untouched.

## Install

    make install          # → ~/bin/jjj
    make check            # our tests + upstream tests + stock jjui build

## What you get

- Opens on the whole visible history (`all()`), newest first, with a header
  per day and per ISO week. `J` flips to jjui's graph. `{` / `}` jump days.
- Three panels: `[1] Status` (working copy, nearest bookmark ↑/↓ against its
  remote, fetch age, mode + revset), `[4] Journal` (every jjui key works here),
  `[0] Detail` (`jj show --stat --git`; `j`/`k`, `ctrl+d`/`ctrl+u`, `pgup`/`pgdown`
  scroll it when focused). Digits and `tab` move focus; click a panel to focus it.
- Under 120 columns it falls back to jjui's single list.

## Config

`~/.config/jjj/config.toml`, `JJJ_CONFIG_DIR`, and `<repo>/.jjj/config.toml`;
every jjui key works, plus:

    [journal]
    enabled  = true      # start in journal mode
    template = ""        # jj template for a journal row; empty = bundled
    limit    = 5000
    [shell]
    enabled        = true
    narrow_columns = 120
    left_width     = 0.34

Journal mode is delivered through the streaming row source, so jjj forces
`[revisions] log_batching = true` at startup regardless of what's in your
config file. `-n`/`--limit` applies to graph mode; journal mode uses
`[journal] limit` instead. `journal.enabled` only takes effect when
`[shell] enabled` is also true.

## Development

Spec: `docs/superpowers/specs/2026-09-04-jjjournal-design.md`.
Plans: `docs/superpowers/plans/`. Upstream rules and sync: `UPSTREAM.md`.
Smoke test without Docker: `make smoke REPO=/path/to/a/jj/repo`.
