# jjjournal (`jjj`) — Design

- **Date:** 2026-09-04
- **Status:** approved design, not yet planned
- **Upstream:** [idursun/jjui](https://github.com/idursun/jjui) (MIT), forked at main `b4378e9b` (post-v0.10.9)
- **Fork:** https://github.com/humdrum-tiv/jjjournal → `~/Developer/Home/jjjournal` (jj-colocated; remotes `origin` = fork, `upstream` = jjui)
- **Command:** `jjj`

## 1. Purpose

Commit history read as a work journal. `jj log` and `jj st` show only the
working-copy neighbourhood; jjui opens on the same narrow revset with two-line
rows and no detail pane. jjj is a fork of jjui that is *informative* — where a
project has been (journal, stats) and where it is going (Backlog.md tasks) —
and *functional* — every jjui operation (describe, commit, squash, absorb,
rebase, bookmarks, push, fetch) stays available in the same screen.

The shape is lazygit's: several panels visible at once (status, files,
bookmarks, commits, tasks on the left; detail, notes and stats on the right),
numbered, focus-cycled, with the focused side panel expanding.

## 2. Decisions already made

| Topic | Decision |
|---|---|
| Strategy | **Tracking fork.** Upstream stays a remote; our commits sit on top of upstream `main`; we rebase onto each upstream release. |
| Name | Repo `jjjournal`, binary and command `jjj`. |
| Hosting | GitHub fork under `humdrum-tiv` (public, MIT, keeps the fork link). |
| Module path | Unchanged (`github.com/idursun/jjui`). The new binary is a second entrypoint, `cmd/jjj`. |
| Stock jjui | Never touched. `jjui` keeps building from the same tree and is the regression check that upstream is intact. |
| Notes panel | The selected change's **full description body**. |
| Cross-repo timeline | **Phase 2, separate feature.** Not in this spec. |

The `jjournal` wrapper (jjui under `JJUI_CONFIG_DIR=~/.config/jjournal`) and
the `jj journal` alias are the stopgap until `jjj` ships; the wrapper retires
when P1 (§14) lands.

## 3. Shape and layout

### 3.1 Panels

Two columns. Left ≈ ⅓ of the width, right the rest. Panels are numbered the
lazygit way.

```
┌[1] Status──────────────┐┌[0] Detail──────────────────────────────────┐
│ jjjournal  @ mzpuwrwr  ││ Commit ID / Change ID / Bookmarks / Author  │
│ main ↑2 ↓0 · fetched 3h││ full message                                │
├[2] Files───────────────┤│ 2 files changed, 128 insertions(+), 1 (-)   │
│ M internal/ui/ui.go    ││ diff --git a/… (git-style, coloured)        │
│ A internal/ui/shell/…  ││                                             │
├[3] Bookmarks───────────┤│                                             │
│ main      ↑2 origin    ││                                             │
│ journal   (local)      ││                                             │
├[4] Journal─────────────┤├[6] Notes────────────────────────────────────┤
│══ Week 36 · Aug 31–Sep 6│ full description body of the selected change│
│── Today · Fri 2026-09-04│                                             │
│ xmqwnktn 16:30 main feat├[7] Stats────────────────────────────────────┤
│ nwvzkyty 10:11      docs│ today 3 · week 11 · streak 4d  ▁▃▅▂▇▃▁ …    │
├[5] Tasks───────────────┤│                                             │
│ In progress            ││                                             │
│  TASK-006 M jjui journa││                                             │
└────────────────────────┘└─────────────────────────────────────────────┘
 status line: keys for the focused panel · flash messages
```

| # | Panel | Source | Shown when |
|---|---|---|---|
| 1 | **Status** | repo name, working-copy change id, current bookmark, ahead/behind vs. its remote counterpart, age of last fetch | always |
| 2 | **Files** | files changed in the selected change (working copy when `@` is selected) | always |
| 3 | **Bookmarks** | local + remote bookmarks with ahead/behind markers (upstream `bookmarkpane`) | always |
| 4 | **Journal** | the commit list; journal mode by default, graph mode on toggle (upstream `revisions` + our row source) | always; tallest panel |
| 5 | **Tasks** | Backlog.md tasks grouped by status | only when `<repo>/backlog/` exists |
| 0 | **Detail** | `jj show --stat --git` of the selected change; a single file's diff when Files is focused; the task body when Tasks is focused | always |
| 6 | **Notes** | full description of the selected change, wrapped; `(no description)` when empty | always |
| 7 | **Stats** | compact: today / this week / streak / mini heatmap. Zoomed: full heatmap + weekday histogram (§9) | always |

Panels 6 and 7 are not in lazygit; they are the "informative" half. Panel
numbers are stable so keys can be documented.

### 3.2 Focus, sizing, keys

- **Focus:** digits `0`–`7` jump to a panel; `tab` / `shift+tab` cycle. The
  focused panel has a highlighted border; only the focused panel receives
  non-shell keys.
- **Accordion:** the focused *left* panel takes ~50 % of the column height;
  the others share the rest (minimum 3 rows each). Right column: Detail gets
  the space not used by Notes (max 8 rows) and Stats (3 rows compact).
- **Zoom:** `+` cycles the focused panel through normal → half (its column) →
  full screen; `_` reverses. Zoom is how Stats shows its full heatmap and how
  Journal becomes the classic single list.
- **Panel-local keys:** inside Journal every jjui builtin binding works
  unchanged (describe, new, squash, rebase, absorb, bookmarks, git menu,
  revset editor, quick search…). Files, Bookmarks, Tasks, Notes, Stats have
  their own small keymaps (§5–§9).
- **Shell keys** (defaults; if one collides with a jjui builtin binding at the
  pinned upstream, the shell key wins inside jjj and the collision is listed in
  `UPSTREAM.md`):

  | Key | Action |
  |---|---|
  | `0`–`7` | focus panel |
  | `tab` / `shift+tab` | next / previous panel |
  | `+` / `_` | zoom in / out |
  | `J` | Journal panel: toggle journal ↔ graph mode |
  | `E` | Journal panel: export the selected day to the daily note (§10) |
  | `?` | help for the focused panel (upstream help model, filtered) |
  | `q` | quit (when no operation is open) |

- **Mouse:** click focuses a panel and selects a row; wheel scrolls the panel
  under the cursor (upstream's interaction layer already supports this).

### 3.3 Narrow fallback

Below 120 columns (configurable) the shell renders the classic jjui layout:
Journal full width with the Detail pane toggled by `p`. All panels remain
reachable by number (they open zoomed). This keeps jjj usable in a narrow
tmux split.

## 4. Journal mode (panel 4)

Journal mode is the default list. Graph mode is stock jjui's list with our
one-line dated row template; `J` toggles.

**Ordering.** Rows are sorted by **committer timestamp, newest first**, not by
topology. Multiple heads interleave chronologically — that is the point. The
working copy `@` is always the first row (its snapshot time is always the
newest anyway). Hidden and abandoned changes are not shown (same visibility as
`jj log`).

**Headers.** A day header precedes the first row of each calendar day (local
time); a week header precedes the first day of each ISO week (Monday start).
Header rows are not selectable; cursor movement skips them; `{` / `}` jump to
the previous / next day header.

```
══ Week 36 · Aug 31 – Sep 6 ═══════════════════════════════
── Today · Fri 2026-09-04 ───────────────────────  3 changes
── Yesterday · Thu 2026-09-03 ───────────────────  1 change
── Mon 2026-08-31 ───────────────────────────────  7 changes
```

**Row template (journal mode).** Time only; the date is in the header.

```
change_id(8, unique prefix highlighted)  HH:MM  bookmarks tags  [conflict] [(empty)]  subject
```

**Row template (graph mode).** Same, with `%a %Y-%m-%d %H:%M` instead of
`HH:MM` (this is today's `log_journal` alias, bundled so jjj does not depend on
the user's jj config).

Both templates are compiled-in defaults under `[revisions] template` and
`[journal] template`; users can override either. Author is deliberately not in
the row (the user's repos carry several of their own identities; it is in
Detail).

**Revset.** Default `all()`. The revset editor (`L` in jjui) works as usual
and applies to journal mode too; headers are recomputed from whatever rows the
revset returns. `-r` on the command line still overrides for one run.

**Loading.** Journal mode needs the whole revset before it can sort, so it
loads without upstream's 50-row batching, up to `[journal] limit` (default
5000; a status-line note says when the limit was hit). Graph mode keeps
upstream's streaming/batching untouched.

**Operations.** Because rows are upstream `parser.Row` values with real change
ids, every upstream operation works on a journal row exactly as in graph mode.
Header rows carry no commit and are ignored by operations and by multi-select.

**Working copy row.** In journal mode the `@` row shows `(working copy)` after
the time when it is empty and undescribed, instead of `(no description set)`.

## 5. Detail (panel 0) and Notes (panel 6)

**Detail** is upstream's preview model with jjj defaults:

- Journal / Bookmarks / Status focused → `jj show --color always -r $change_id --stat --git`
- Files focused → `jj diff --color always -r $change_id --git $file`
- Tasks focused → the task's Markdown body (rendered plainly; no external renderer)

The three commands stay configurable through upstream's `[preview]` keys
(`revision_command`, `file_command`) plus a new `task_command`-free path (the
task body is read from disk, not from a command).

**Notes** shows the selected change's full description, soft-wrapped to the
panel width, first line bold. It updates on every selection change in Journal.
It is read-only; `enter` on Notes opens upstream's describe operation for that
change so the body can be edited in place.

## 6. Files (panel 2)

Lists files changed in the selected change: status letter (`A` `M` `D` `R`),
path, and `+n −m` from `jj log -r <rev> --summary` / `--stat` (upstream's
`details` operation already knows how to fetch and parse this; the panel
reuses that code as a standalone model rather than as a modal over the list).
For `@` it lists working-copy changes and refreshes on jjui's normal refresh
tick. `enter` opens the file diff full-screen (upstream diff model); moving the
cursor updates Detail (§5). Upstream's details keys that make sense per file
(`r` restore, `s` split, `d` diff) are kept.

## 7. Bookmarks (panel 3) and Status (panel 1)

**Bookmarks** is upstream's `bookmarkpane` hosted persistently instead of
opened on demand. Each row: name, `↑n ↓m` against its tracked remote, remote
name, and a marker when the bookmark is at the selected change. Upstream keys
(create, move, delete, track, push) are unchanged.

**Status** is a three-line panel:

```
<repo dir name>   @ <change id>   op: <short op id>
<current bookmark or "(no bookmark)">  ↑<ahead> ↓<behind> <remote>   fetched <age>
<revset in effect>
```

- *Current bookmark*: the nearest bookmark at or below `@` (`closest_bookmark`
  in the working-copy's ancestry).
- *Ahead/behind*: counts of `<remote>..<bookmark>` and `<bookmark>..<remote>`
  from one `jj log --no-graph -T ''` per side; recomputed on refresh.
- *Fetched*: modification time of `.git/FETCH_HEAD` (colocated repos) or
  `.jj/repo/store/git/FETCH_HEAD`; shown as `3h ago`, `never` if absent.

## 8. Tasks (panel 5) — Backlog.md

Shown only when `<repo root>/backlog/` exists.

**Data.** Parsed directly from `backlog/tasks/*.md` YAML frontmatter — `id`,
`title`, `status`, `priority`, `labels` — and grouped in the order given by
`statuses:` in `backlog/config.yml` (fallback: Backlog, In progress, Paused,
Done). The task id prefix comes from `task_prefix` in the same file (default
`TASK`). No `backlog` CLI call at runtime; the files are the source of truth.
The panel re-reads on jjui's refresh tick when a task file's mtime changed.

**Rows.** `TASK-006  M  jjui journal view: full history …` with priority as a
single letter (`H` `M` `L`) and labels dimmed after the title. The Done group
is collapsed by default; `enter` on a group header expands it.

**Linking.**

- A Journal row whose description contains a task id (`\bTASK-\d+\b`, prefix
  from config) gets a dim ` ⋄TASK-006` suffix.
- Selecting such a row highlights the matching task(s) in Tasks.
- In Tasks, `f` filters Journal to `description(substr:"TASK-006")`; `f` again
  (or the revset editor) restores the previous revset.
- `enter` on a task shows its body in Detail (§5).

**Read-only in this phase.** Status changes still go through the `backlog`
CLI (the house rule: never hand-edit task files). A Lua action may call it;
jjj does not write task files.

## 9. Stats (panel 7)

**Data.** One `jj log --no-graph -r <active revset> ~ root() -T 'committer.timestamp().local().format("%Y-%m-%d") ++ "\n"'`
per refresh, bucketed by day in Go. Empty undescribed changes are excluded.
Respects the active revset, so `mine()` or a date range filters the stats too.

**Compact (3 rows):**

```
today 3 · this week 11 · this month 42 · all 618
streak 4 days · busiest Wed
▁▃▅▂▇▃▁▂▄▆▃▁▁▂▅▇▂  (last 17 weeks, one cell per week)
```

**Zoomed (`+`):** GitHub-style heatmap, 7 rows × `[stats] weeks` (default 26)
columns, five intensity levels, month labels along the top, weekday labels on
the left; below it a weekday histogram and the same totals. `h` / `l` move a
cursor over the heatmap; `enter` sets the revset to that day
(`committer_date(after:"<day>", before:"<day+1>")`), which also drives Journal.

*Streak*: consecutive days with ≥1 change, ending today or yesterday.

## 10. Push to daily note

`E` in Journal, on a day header or any row of that day, appends the day's
changes for this repo to the Obsidian daily note.

- **Path:** `[journal] daily_note = "~/Humdrum/Daily/{{date}}.md"` (Go
  template; `{{date}}` = `YYYY-MM-DD`).
- **Block:** an Obsidian callout wrapped in markers so re-exporting replaces
  rather than duplicates:

  ```markdown
  <!-- jjj:jjjournal:2026-09-04 -->
  > [!journal] jjjournal · Fri 2026-09-04
  > - 16:30 `xmqwnktn` feat(TASK-006): jjui commit-journal view — all() revset…
  > - 10:11 `nwvzkyty` docs(TASK-004): correct final repo count to 61 under ~/Developer
  <!-- /jjj -->
  ```

  Callout type is `[journal] callout` (default `journal`). Repo name is the
  repo directory basename. Empty undescribed changes are skipped.
- **Placement:** if the note contains a ```` ```base ```` fence, the block is
  inserted above the first one (house convention); otherwise appended.
- **Missing note:** created with the bare daily frontmatter
  (`---\ntags: [daily]\n---\n\n#daily\n`) unless `[journal] create_note = false`.
- **Feedback:** flash message `exported 3 changes → Daily/2026-09-04.md`; `E`
  on a day with no changes flashes and does nothing.

Tests write to a temp directory; nothing in the test suite touches the vault.

## 11. Configuration

- **Global:** `~/.config/jjj/config.toml` (+ `config.lua`), overridable with
  `JJJ_CONFIG_DIR`. **Per repo:** `.jjj/config.toml` and `.jjj/config.lua` at
  the repo root. jjj never reads `~/.config/jjui`, `.jjui/`, or
  `JJUI_CONFIG_DIR`, so both tools coexist.
- **Loader:** upstream's, with the directory names swapped through constants
  set by `cmd/jjj`. Every upstream key keeps working (themes, bindings,
  actions, preview, revisions…).
- **jjj defaults** (compiled in; all overridable):

  ```toml
  [revisions]
  revset   = "all()"
  template = "<bundled graph-mode template>"

  [journal]
  enabled    = true            # start in journal mode
  template   = "<bundled journal-mode template>"
  limit      = 5000
  daily_note = "~/Humdrum/Daily/{{date}}.md"
  callout    = "journal"
  create_note = true

  [preview]
  show_at_start    = true
  revision_command = ["show", "--color", "always", "-r", "$change_id", "--stat", "--git"]
  file_command     = ["diff", "--color", "always", "-r", "$change_id", "--git", "$file"]

  [shell]
  enabled        = true        # false → classic jjui layout with jjj defaults
  narrow_columns = 120
  left_width     = 0.34
  notes_rows     = 8
  hidden         = []          # e.g. ["files", "bookmarks"]

  [tasks]
  enabled     = true
  show_done   = false

  [stats]
  weeks = 26
  ```

## 12. Architecture and upstream tracking

### 12.1 Packages

New (ours; never conflict with upstream):

| Package | Role |
|---|---|
| `cmd/jjj` | entrypoint: config dir constants, flags, boots the shell |
| `internal/ui/shell` | layout, focus, zoom, panel registry, narrow fallback |
| `internal/ui/files` | standalone Files panel (adapted from `operations/details`) |
| `internal/ui/tasks` | Tasks panel |
| `internal/ui/notes` | Notes panel |
| `internal/ui/stats` | Stats panel (compact + zoomed) |
| `internal/ui/statuspanel` | the three-line Status panel (§7) |
| `internal/journal` | pure logic, no UI: sort + group rows, header text, heatmap buckets, streak, daily-note block build/replace |
| `internal/backlog` | frontmatter + `config.yml` parsing, task-id regex |

Reused as-is: `revisions`, `bookmarkpane`, `preview`, `diff`, `status` (the
bottom status line), `help`, `flash`, `actions`, `scripting`, `parser`, `jj`.

### 12.2 Boot path and the row-source seam

`cmd/jjj/main.go` is a copy of `cmd/jjui/main.go` with the config constants
and a call to `shell.New(ctx)` instead of `ui.New(ctx)`. Upstream's
`internal/ui/ui.go` is untouched, so `go build ./cmd/jjui` keeps working.

Journal mode needs one seam in upstream: `revisions.Model` today pulls rows from
`graph.GraphStreamer`. We add a small interface — a **row source** that yields
`[]parser.Row` (optionally with unselectable header rows, i.e. `Commit == nil`)
— and make the graph streamer its default implementation. Our
`internal/journal` provides the journal implementation (buffer, sort, insert
headers). This is the one deliberate upstream edit with logic in it; it is
small, and it is what lets every operation work on journal rows for free.

The P0 spike (§14) proves this seam and the side-by-side hosting before
anything else is built.

### 12.3 Rules for touching upstream files

1. New behaviour goes in new packages.
2. Upstream files may be edited only for: (a) the row-source seam; (b) wiring
   (registering a panel, exporting a symbol, a constant for the config dir);
   (c) a bug fix we also send upstream as a PR.
3. Every touched upstream file is listed in `UPSTREAM.md` with the reason.
4. No reformatting, renaming or moving of upstream code.

### 12.4 Sync procedure

`scripts/sync-upstream.sh` (also `make sync-upstream`):

```bash
jj git fetch --remote upstream
jj rebase -s 'roots(main@upstream..main)' -d main@upstream   # our stack onto upstream
go build ./... && go test ./... && go build ./cmd/jjui        # upstream still intact
jj bookmark set main -r 'heads(main@upstream..)'              # then: jj git push
```

Conflicts are resolved by hand (mergiraf is configured). Sync points are
upstream release tags; `UPSTREAM.md` records the last synced tag.

## 13. Testing

- **Unit** (`go test ./...`): `internal/journal` (sorting, day/week grouping
  incl. DST and year boundaries, header text, `Today`/`Yesterday`, heatmap
  bucketing, streak, daily-note block build and idempotent replace, placement
  above a ```` ```base ```` fence); `internal/backlog` (frontmatter variants,
  status order from config, task-id regex with a custom prefix); `shell`
  layout math (accordion, zoom, narrow fallback).
- **End-to-end:** upstream's PTY harness in `e2e/` gets one jjj scenario:
  launch in a fixture repo, assert the day header and a journal row render,
  focus each panel by number, zoom Stats, toggle graph mode.
- **Upstream intact:** `go build ./cmd/jjui` and upstream's own tests must
  pass on every commit (Makefile `check` target).
- **Manual:** run `jjj` in `~/Developer/Home/ticktock` (55 changes) and in a
  repo with a `backlog/` folder.

## 14. Delivery order

Each item becomes one or more tasks in the implementation plan.

| Phase | Deliverable | Done when |
|---|---|---|
| **P0 spike** | shell hosting Journal (graph mode) + Detail + Status side by side; row-source seam with a throwaway header row | `jjj` renders three panels in ticktock; `jjui` still builds; seam diff is small |
| **P1 journal** | journal mode, headers, bundled templates, jump keys, config dir, `make install` → `~/bin/jjj`, `jjournal` wrapper retired, dash + Tooling Index updated | AC: full history, dated one-line rows, detail pane, operations work on journal rows |
| **P2 panels** | Files, Bookmarks, Notes panels; accordion + zoom; narrow fallback | all left panels live; `+` zooms |
| **P3 tasks** | Tasks panel, linking, filter | shown when `backlog/` exists; `⋄TASK-nnn` markers |
| **P4 stats** | compact strip + zoomed heatmap | counts match `jj log` for the same revset |
| **P5 export** | daily-note export | idempotent block in a temp vault in tests; manual run into the real daily note |
| **P6 sync** | `sync-upstream.sh`, `UPSTREAM.md`, `check` target, e2e scenario | one clean sync against upstream `main` performed |

## 15. Bookkeeping (house rules)

- `backlog init "jjjournal" --agent-instructions claude --integration-mode cli --task-prefix TASK --zero-padded-ids 3`
  in the fork (it has no CLAUDE.md or AGENTS.md); statuses set to the standard four.
- Project doc `Humdrum/Claude/Projects/jjjournal.md` (created with this spec),
  with the issues Base once `backlog/` exists, plus the vault symlink
  `Humdrum/Claude/Issues/jjjournal → ~/Developer/Home/jjjournal/backlog/tasks`.
- Custom Tooling Index: `jjj` row (Homebrew-installed TUIs table or a new
  "own builds" row), dash entry `jjj` replacing `jjournal`, config-dir row.
- Every repo on this machine is jj-colocated; all VCS mutations in the fork go
  through `jj`.

## 16. Out of scope

- Cross-repo timeline (phase 2, its own spec).
- Editing Backlog.md tasks from inside jjj.
- Any change to stock jjui's behaviour, config, or binary.
- Homebrew tap / custard release pipeline (possible later; `make install` is enough now).
- Theming beyond upstream's theme files.

## 17. Resolved questions

- *Notes panel content* → full description body (user, 2026-09-04).
- *Author in rows* → omitted; several identities make it noise.
- *Timestamp* → committer time, local; matches jj's own log.
- *Panel set* → as in §3.1; Files is a full member of P2, not optional.
