# jjj — jjui as a commit journal

`jjj` is this fork's binary: jjui plus a journal-first layer. Stock `jjui`
still builds from the same tree (`go build ./cmd/jjui`) and is untouched.

## Install

    make install          # → ~/bin/jjj
    make check            # our tests + upstream tests + stock jjui build

## What you get

- Opens on the whole visible history (`all()`), newest first, with a header
  per day and per ISO week. `J` flips to jjui's graph. `{` / `}` jump days.
- Six panels. Left column: `[1] Status` (working copy, nearest bookmark ↑/↓
  against its remote, fetch age, mode + revset), `[2] Files` (the files changed
  in the selected revision, with upstream's file-details keys; `enter` shows a
  file's diff in Detail), `[3] Bookmarks` (upstream's bookmark pane, kept open,
  plus `a`/`A` advance), `[4] Journal` (every jjui key works here). Right
  column: `[0] Detail` (`jj show --stat --git`, or the file diff opened from
  Files; scrollable when focused) above `[6] Notes` (the selected revision's
  full description; `enter` describes it inline).
- The left column is an accordion: the focused panel grows. `+` / `_` zoom
  the focused panel to its column, then the whole screen. Digits and `tab`
  move focus; click a panel to focus it. Focus stays put while a panel has a
  confirmation or filter open. `[shell] hidden = ["files", ...]` removes
  Status, Files, Bookmarks or Notes.
- Under 120 columns it shows jjui's single list while Journal is focused and
  the focused panel full-screen otherwise.

## Keys

Every jjj key is an action: rebind it in `~/.config/jjj/config.toml` like any
jjui binding, e.g. `bindings = [{ key = "x", action = "jjj.shell.focus_files", scope = "jjj.shell" }]`.
Files keeps upstream's file-details keys (space select, r restore, s split,
S squash, A absorb, d diff, / filter, * revisions touching the file);
Bookmarks keeps upstream's bookmark-pane keys (c create, m move, d delete,
r rename, t/u track, P push, F fetch, e edit, n new, s show).

| Group | Key | Action |
|---|---|---|
| Shell | `?` | help: common tasks with explanations, then every key (`/` searches) |
| Shell | `f1` | upstream's compact expanded status help |
| Shell | `0` `1` `2` `3` `4` `6` | focus Detail, Status, Files, Bookmarks, Journal, Notes |
| Shell | `tab` / `shift+tab` | next / previous panel |
| Shell | `+` / `_` | zoom focused panel in / out |
| Shell | `J` | journal / graph |
| Shell | `{` / `}` | previous / next day |
| Shell | `esc` (Status) | back to journal |
| Detail | `j` `k` `up` `down` | scroll |
| Detail | `ctrl+d` / `ctrl+u` | half page down / up |
| Detail | `pgdown` / `pgup` | page down / up |
| Detail | `esc` | back to journal |
| Notes | `j` `k` `up` `down` | scroll |
| Notes | `enter` | describe (inline, on the Journal row) |
| Notes | `esc` | back to journal |
| Files | `enter` | diff in Detail |
| Files | `e` | diffedit |
| Files | `o` | open in $EDITOR |
| Files | `b` | annotate |
| Files | `M` / `shift+m` | resolve |
| Files | `tab` / `shift+tab` | next / previous panel |
| Bookmarks | `a` | advance closest bookmark to the selected revision |
| Bookmarks | `A` / `shift+a` | advance to @ and push |

Bookmarks rows: pinned bookmarks (`[bookmarks] pinned`, default `main`,
`master`, `trunk`) come first, tagged `trunk`. Each local bookmark shows its
tracked remotes with a sync marker: `=` in sync, `↑n` local has n commits the
remote lacks (push), `↓n` the remote has n commits local lacks, `⇅a/b`
diverged. `enter` expands a bookmark to one row per remote, in words.

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
    [bookmarks]
    pinned = ["main", "master", "trunk"]   # shown first, tagged trunk

Journal mode is delivered through the streaming row source, so jjj forces
`[revisions] log_batching = true` at startup regardless of what's in your
config file. `-n`/`--limit` applies to graph mode; journal mode uses
`[journal] limit` instead. `journal.enabled` only takes effect when
`[shell] enabled` is also true.

## Development

Spec: `docs/superpowers/specs/2026-09-04-jjjournal-design.md`.
Plans: `docs/superpowers/plans/`. Upstream rules and sync: `UPSTREAM.md`.
Smoke test without Docker: `make smoke REPO=/path/to/a/jj/repo`.
