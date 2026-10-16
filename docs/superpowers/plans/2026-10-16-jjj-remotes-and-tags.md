# jjj — Remotes panel and Tags tab — Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Turn panel 3 into **Remotes** (local bookmarks vs every tracked remote, plus each remote's last fetch) with a second **Tags** tab (your milestone tags, newest first, date and subject; `enter` reveals the commit in Journal), reachable as `3` and `8`.

**Architecture:** Slice 1 (commit `784b2e8f`) already ships sync markers, trunk pinning and the journal view of upstream's bookmark pane. This plan adds per-remote fetch ages to that view, a new jjj package `internal/ui/tags` (data load + list model), and a tab mechanism in the shell so the panel 3 box hosts either the bookmark pane (Remotes tab) or the tags model (Tags tab). No new upstream seams: every change is in jjj packages, jjj-owned files inside upstream packages that `UPSTREAM.md` already lists, or generated files.

**Tech Stack:** Go (module `github.com/idursun/jjui`), Bubble Tea v2 / Lipgloss v2 (`charm.land/*`), `github.com/charmbracelet/x/ansi`, `stretchr/testify`, `jj` 0.45 on PATH.

**Spec:** Backlog `TASK-012.10` (`backlog task view TASK-012.10 --plain`: description and notes hold the decisions of 2026-10-16). Base spec: `docs/superpowers/specs/2026-10-13-jjj-functional-design.md` §4.3.

**Deviations from the task, decided while planning (tell the reviewer and the user):**

1. **Tag ownership filter.** The agreed rule was "tracked on origin, or local and unpushed; hide tags only on another remote". jj 0.45 does not record which remote a tag came from: in this repo the jjui release tags (`v0.10.x`) list as local tags with only an `@git` copy, never `@upstream`. Rule used instead: **hide a tag when its commit is reachable from the bookmarks of a foreign remote**, where a foreign remote is one with no tracked bookmark (here `upstream`). In a repo with only `origin`, or with mirrors whose `main` you track (`origin` + `gitlab`), nothing is hidden.
2. **Fetch age per remote comes from `FETCH_HEAD`.** jj records no operation for a fetch that changes nothing, so the op log cannot date fetches. `FETCH_HEAD` names the URL(s) of the last fetch; remotes whose URL appears there get its mtime, others show no age.
3. **Identifiers keep the old names.** Only the display title changes to "Remotes": `PanelBookmarks`, scope `jjj.bookmarks`, action `jjj.shell.focus_bookmarks` stay, so configs keep working. `[shell] hidden` accepts `"remotes"` as an alias of `"bookmarks"`, and `"tags"` hides only the Tags tab.

## Global Constraints

- Every VCS mutation goes through `jj` (a hook blocks mutating `git`). Commit with `jj commit -m "..."`. Never `git add` / `git commit`.
- No new edits to upstream files beyond those `UPSTREAM.md` already lists; `internal/jj/jjj_commands.go` and `internal/ui/bookmarkpane/jjj_view.go` are jjj-owned and may grow (update their `UPSTREAM.md` rows when their exported names change).
- `make check` (vet, build, `go build ./cmd/jjui`, `go test ./...`) passes after every task.
- Never hand-edit `internal/ui/actions/catalog_gen.go`, `internal/ui/actionmeta/builtins_gen.go`, `internal/config/default/types.lua`; run `go run ./cmd/genactions`.
- Models handle `intents.Intent`, not raw `tea.KeyMsg`; `HandleIntent` returns `handled = true` whenever the scope consumes the intent.
- Rows are clipped to their box width (use `bookmarkpane`'s `rowWriter` pattern: `ansi.Truncate(text, left, "…")`).
- Ids passed to `bookmarkpane.RevealRevisionMsg` use `commit_id.shortest()` (the form Journal rows carry; `revisions.selectRevisionExact` compares exactly).
- jj commands for jjj data always pass `--ignore-working-copy --color never`; `context.RunCommandImmediate` strips the trailing newline, so never count `\n`.
- Repo root `/Users/kortum/Developer/Home/jjjournal`; no new module dependencies.

## Review Focus

1. **A tag on a commit outside Journal's loaded rows** (older than `[journal] limit`, or outside a custom revset): `enter` must not move focus to Journal with nothing selected; keep focus in Tags and flash nothing. Pinned by `TestRevealUnloadedTagStaysInTags` (Task 5).
2. **A conflicted or deleted tag** (`normal_target` empty): listed with `conflict`, no date, not revealable — never a parse error that drops the whole list. Pinned by `TestParseTagsConflicted` (Task 3).
3. **A repo with no remotes, no tags, or never fetched:** Remotes shows rows without ages, Tags shows `No tags`, nothing errors. Pinned by `TestRemoteFetchTimesNoFetchHead` (Task 1) and `TestTagsEmpty` (Task 4).
4. **Pressing `8` while the bookmark pane has a filter or confirmation open:** focus/tab must not change (same rule as `setFocus` refusing while editing). Pinned by `TestTagsKeyRefusedWhileRemotesEditing` (Task 5).
5. **Tag names and remote URLs with odd shapes** (`ssh://git@host:22/x.git`, `https://user@host/x`, trailing `/`): URL matching normalises scheme, user, port-less `:` path and `.git`. Pinned by `TestNormalizeRemoteURL` (Task 1).

---

### Task 1: Per-remote fetch age

**Files:**
- Modify: `internal/statusinfo/statusinfo.go` (add functions; existing ones unchanged)
- Test: `internal/statusinfo/statusinfo_test.go`

**Interfaces:**
- Produces: `func RemoteFetchTimes(repoRoot string, remotes map[string]string) map[string]time.Time` — remote name → URL in, remote name → `FETCH_HEAD` mtime out, only for remotes whose normalised URL appears after `" of "` on a `FETCH_HEAD` line. Same two candidate paths as `FetchTime`. Missing file → empty map.
- Produces: `func ParseRemoteList(output []byte) map[string]string` — parses `jj git remote list` lines `name url`.
- Produces (unexported): `normalizeRemoteURL(u string) string` — lowercases host, strips `scheme://`, `user@`, a trailing `.git` and `/`, turns the first `host:path` `:` into `/`. `git@github.com:a/b.git`, `ssh://git@github.com/a/b`, `https://github.com/a/b/` and `github.com:a/b` all → `github.com/a/b`.

- [ ] **Step 1: Write failing tests** — `TestNormalizeRemoteURL` (the four forms above → `github.com/a/b`), `TestRemoteFetchTimes` (temp dir with `.git/FETCH_HEAD` containing `7d1e…\tnot-for-merge\tbranch 'main' of github.com:a/b\n`; remotes `{origin: git@github.com:a/b.git, upstream: git@github.com:c/d.git}` → only `origin`, equal to the file's mtime), `TestRemoteFetchTimesNoFetchHead` (empty dir → empty map), `TestParseRemoteList`.
- [ ] **Step 2: Run** `go test ./internal/statusinfo/ -run 'Remote|Normalize'` — expect FAIL (undefined).
- [ ] **Step 3: Implement** the three functions in `statusinfo.go`.
- [ ] **Step 4: Run** the same command — PASS.
- [ ] **Step 5: Commit** `jj commit -m "feat(jjj): per-remote fetch time from FETCH_HEAD (TASK-012.10)"`

### Task 2: Remotes view shows fetch ages; panel titled Remotes

**Files:**
- Modify: `internal/ui/bookmarkpane/jjj_view.go`, `internal/ui/bookmarkpane/bookmarkpane.go` (only inside the existing `m.jjj` branches of `ViewRect`)
- Modify: `internal/ui/shell/bookmarks.go` (`loadSync`, `syncLoadedMsg`, `title`), `internal/ui/shell/shell.go` (`panelNames`)
- Modify: `internal/jj/jjj_commands.go` (add `GitRemoteList`)
- Test: `internal/ui/bookmarkpane/jjj_view_test.go`, `internal/ui/shell/bookmarks_test.go`

**Interfaces:**
- Consumes: `statusinfo.RemoteFetchTimes`, `statusinfo.ParseRemoteList`, `statusinfo.FormatAge` (Task 1 / existing).
- Produces: `func GitRemoteList() CommandArgs` → `{"git", "remote", "list", "--ignore-working-copy", "--color", "never"}`.
- Produces: `func (m *Model) SetFetchTimes(at map[string]time.Time, now time.Time)` in `jjj_view.go`.
- Produces: `syncLoadedMsg` gains `fetched map[string]time.Time`; `loadSync` runs `GitRemoteList` once and `statusinfo.RemoteFetchTimes(ctx.Location, statusinfo.ParseRemoteList(out))` (`MainContext.Location` is the repo root, as in `statuspanel`).

Behaviour: in journal view the remote line is always drawn (it replaces the old "only while focused" rule) and reads `all  local  origin 22m ago  gitlab 3d ago  upstream` — each remote name followed by its dimmed age when known, the active filter in the existing selected style, names still clickable (`RemoteClickedMsg`). Panel title `Remotes`. `[shell] hidden = ["remotes"]` hides it like `"bookmarks"`.

- [ ] **Step 1: Write failing tests** — `TestJournalViewRemoteLineShowsFetchAge` (pane from `newJournalPane`, `SetFetchTimes({origin: now-22m}, now)`, rendered and ANSI-stripped output contains `origin 22m ago`, and `upstream`-style remotes without a time show the bare name); in `TestJournalViewRowsShowSyncState` keep `NotContains "Remotes:"` (the old prefix is gone) and add `Contains "all  local  origin"` (the line is now always drawn); `TestLoadSyncCountsOnlyDifferingTrackedPairs` expects `GitRemoteList()` before the counts and a non-nil `fetched` map.
- [ ] **Step 2: Run** `go test ./internal/ui/bookmarkpane/ ./internal/ui/shell/` — FAIL.
- [ ] **Step 3: Implement** `GitRemoteList`, `SetFetchTimes`, a `renderJournalRemotes(dl, box)` used by `ViewRect` when `m.jjj != nil`, the `loadSync`/`update` changes, title and alias.
- [ ] **Step 4: Run** the same command, then `make check` — PASS.
- [ ] **Step 5: Commit** `jj commit -m "feat(jjj): Remotes panel shows each remote's last fetch (TASK-012.10)"`

### Task 3: Tag data — own tags, newest first

**Files:**
- Create: `internal/ui/tags/data.go`, `internal/ui/tags/data_test.go`
- Modify: `internal/jj/jjj_commands.go` (add `TagListJournal`), `internal/jj/jjj_commands_test.go`

**Interfaces:**
- Produces: `func TagListJournal(foreign []string) CommandArgs` → `jj tag list --ignore-working-copy --color never -T <tagTemplate>` where the template emits one line per tag, `;`-separated: `name`, `present`, `commit_id.shortest()` (empty if no `normal_target`), committer timestamp `%Y-%m-%dT%H:%M:%S%z` (empty if none), `foreign` (`true`/`false` from `normal_target.contained_in('<revset>')`), subject `description.first_line()` last (it may contain `;`). `<revset>` is `::(remote_bookmarks(remote=exact:"a") | remote_bookmarks(remote=exact:"b"))` for the foreign remotes, `none()` when there are none. Local tags only (no `-a`): remote copies are not milestones.
- Produces: `type Tag struct { Name, CommitID, Subject string; When time.Time; Conflict bool }`
- Produces: `func ForeignRemotes(bookmarkListOutput string) []string` — remotes (excluding `git`) that appear in `jj.BookmarkListAll()` output with no tracked bookmark; sorted.
- Produces: `func ParseTags(output string) []Tag` — drops `foreign=true` and `present=false` tags; conflicted (`CommitID == ""`) kept with `Conflict: true`; sorted by `When` descending, then name; conflicted last.

- [ ] **Step 1: Write failing tests** — `TestTagListJournalRevset` (no foreign → template contains `contained_in('none()')`; `["upstream"]` → contains `remote_bookmarks(remote=exact:"upstream")`), `TestForeignRemotes` (origin has a tracked `main`, upstream only untracked → `["upstream"]`; `git` never listed), `TestParseTagsNewestFirstAndOwnOnly` (three tags, one `foreign=true` dropped, order by date desc), `TestParseTagsConflicted` (empty id/date → kept, `Conflict`, sorted last), `TestParseTagsSubjectWithSemicolon`.
- [ ] **Step 2: Run** `go test ./internal/jj/ ./internal/ui/tags/` — FAIL.
- [ ] **Step 3: Implement.** Verify the template by hand once: `jj tag list --ignore-working-copy --color never -T '<template>'` in this repo must mark every `v0.10.x` as `foreign=true` when passed `["upstream"]`.
- [ ] **Step 4: Run** — PASS.
- [ ] **Step 5: Commit** `jj commit -m "feat(jjj): tag data — own milestone tags, newest first (TASK-012.10)"`

### Task 4: Tags model — list, keys, reveal

**Files:**
- Create: `internal/ui/tags/tags.go`, `internal/ui/tags/tags_test.go`
- Modify: `internal/ui/intents/jjj_intents.go`, `internal/jjj/defaults.toml`; regenerate with `go run ./cmd/genactions`

**Interfaces:**
- Consumes: `jj.BookmarkListAll`, `jj.TagListJournal`, `ForeignRemotes`, `ParseTags` (Task 3); `bookmarkpane.RevealRevisionMsg`.
- Produces intents (scope `jjj.tags`): `TagsMove{Delta int}` (`//jjui:bind scope=jjj.tags action=move_down set=Delta:1`, `action=move_up set=Delta:-1`), `TagsReveal{}` (`action=reveal`), back via existing `ShellFocusPanel` (`//jjui:bind scope=jjj.tags action=back set=Panel:4`). Generated constant `actions.ScopeJjjTags`.
- Produces: `func New(ctx *appContext.MainContext) *Model`; `(*Model) Load() tea.Cmd` (seq-tagged `loadedMsg`, two jj calls: bookmark list then tag list); `Update(tea.Msg) tea.Cmd` (handles `loadedMsg`, ignores stale seq); `HandleIntent(intents.Intent) (tea.Cmd, bool)`; `ViewRect(dl *render.DisplayContext, box layout.Box)`; `SetFocused(bool)`; `Selected() (Tag, bool)`.
- Defaults in `defaults.toml`: `j`/`down` move_down, `k`/`up` move_up, `enter` and `s` reveal, `esc` back.

Row: `name  YYYY-MM-DD  subject` — name in the bookmark-name style, date dimmed, subject plain, clipped with `…`; conflicted rows `name  conflict` in the error style. Selected row highlighted with `common.DefaultPalette.Get("revisions", "", "", true)` only while focused. Empty list: `No tags` dimmed. `TagsReveal` on a non-conflicted tag returns a cmd producing `bookmarkpane.RevealRevisionMsg{CommitID: tag.CommitID}`; on a conflicted tag, handled with no cmd.

- [ ] **Step 1: Write failing tests** — `TestTagsLoadAndOrder` (runner expects `BookmarkListAll` then `TagListJournal(["upstream"])`; rendered rows in date order), `TestTagsMoveClamps`, `TestTagsRevealEmitsRevealMsg`, `TestTagsRevealConflictedIsInert`, `TestTagsEmpty` (renders `No tags`), `TestTagsStaleLoadIgnored`.
- [ ] **Step 2: Run** `go test ./internal/ui/tags/` — FAIL.
- [ ] **Step 3: Implement**; add intents; `go run ./cmd/genactions`; add bindings.
- [ ] **Step 4: Run** `go test ./internal/ui/tags/ ./cmd/genactions ./internal/jjj/` — PASS.
- [ ] **Step 5: Commit** `jj commit -m "feat(jjj): Tags model with reveal in Journal (TASK-012.10)"`

### Task 5: Panel 3 tabs — Remotes and Tags (`3` / `8`)

**Files:**
- Modify: `internal/ui/shell/layout.go` (add `PanelTags PanelID = 8`; not in `Order`, `leftFlex` or the layout: it is a tab of `PanelBookmarks`)
- Modify: `internal/ui/shell/bookmarks.go`, `internal/ui/shell/shell.go` (`ShellFocusPanel` handling, `drawPanel` title, `panelNames["tags"]`), `internal/ui/shell/guide.go`
- Modify: `internal/ui/intents/jjj_intents.go` (`//jjui:bind scope=jjj.shell action=focus_tags set=Panel:8`), `internal/jjj/defaults.toml` (`{ key = "8", action = "jjj.shell.focus_tags", scope = "jjj.shell", desc = "focus tags" }`); regenerate
- Test: `internal/ui/shell/bookmarks_test.go`, `internal/ui/shell/shell_test.go`

**Interfaces:**
- Consumes: `tags.Model` (Task 4).
- Produces: `bookmarksPanel` gains `tags *tags.Model` and `tab panelTab` (`tabRemotes`, `tabTags`); `title()` returns `"Remotes"` or `"Tags"`; a new optional interface in `panels.go`: `type tabbedPanel interface { tabTitles() []tabTitle }` with `tabTitle struct { id PanelID; title string; active bool }`, which `drawPanel` uses to draw `[3]─Remotes─[8]─Tags` (active bold in the focus colour, inactive dimmed) instead of the single `[n]─Title`.
- Behaviour: `ShellFocusPanel{Panel: 8}` → if `setFocus(PanelBookmarks)` succeeds (it refuses while the pane is editing), set `tab = tabTags`; `Panel: 3` → `tab = tabRemotes`. Clicking either tab label focuses that tab. While `tabTags`: `draw` delegates to `tags.ViewRect`, `scopes()` returns `[{jjj.tags, LeakGlobal, handler}]`, `handleIntent` routes `TagsMove`/`TagsReveal` to the model, `editing()` is false, `setFocused` forwards to the tags model. `init()` and `common.RefreshMsg` also call `tags.Load()`. `[shell] hidden = ["tags"]` disables the tab (`8` then does nothing) without hiding Remotes. Reveal: existing `RevealRevisionMsg` handling moves focus to Journal; if the commit is not among Journal's rows (`core.Revisions.GetCommitIds()` does not contain it), keep focus in Tags instead.
- Guide: rename the group to `Common tasks · Remotes (3)`, add `Common tasks · Tags (8)` with `TagsReveal` ("Show the tag's commit in Journal") and a note line: "Tags are your milestones: `jj tag set v0.3 -r <rev>`, then `jj git push --tag v0.3`."

- [ ] **Step 1: Write failing tests** — `TestFocusTagsShowsTagsTab` (`ShellFocusPanel{8}` → focus `PanelBookmarks`, scopes start with `ScopeJjjTags`), `TestFocusRemotesSwitchesBack` (`8` then `3` → bookmark pane scopes), `TestTagsKeyRefusedWhileRemotesEditing` (pane filter open → `8` leaves tab and focus unchanged), `TestRevealUnloadedTagStaysInTags`, `TestHiddenTagsDisablesTab`, `TestGuideHasTagsGroup`, and a render test asserting the frame tab reads `[3]─Remotes─[8]─Tags`.
- [ ] **Step 2: Run** `go test ./internal/ui/shell/` — FAIL.
- [ ] **Step 3: Implement**; `go run ./cmd/genactions`.
- [ ] **Step 4: Run** `make check` — PASS. Then headless check: build to the scratchpad, run under the pyte snapshot helper with a light background in this repo, press `8`: tab title shows Tags active and the list does not contain `v0.10`.
- [ ] **Step 5: Docs and bookkeeping** — `docs/jjj.md`: keys table rows for `8` and the Tags keys, a short "Remotes and Tags" paragraph (markers, fetch ages, the ownership rule from deviation 1); `UPSTREAM.md` rows for `jjj_commands.go` (new builders) and `jjj_view.go` (`SetFetchTimes`); `backlog task edit TASK-012.10 --check-ac 1 --check-ac 2 --check-ac 3 -s Done`.
- [ ] **Step 6: Commit** `jj commit -m "feat(jjj): Remotes/Tags tabs in panel 3, 8 focuses Tags (TASK-012.10)"`
