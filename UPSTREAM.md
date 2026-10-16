# Upstream tracking

jjj is a tracking fork of [idursun/jjui](https://github.com/idursun/jjui) (MIT).

- Forked at upstream `main` `b4378e9b` (2026-09-04, after v0.10.9).
- Last synced upstream tag: none yet (fork point).
- Remote `upstream` = idursun/jjui, `origin` = humdrum-tiv/jjjournal.

## Rules (spec §12.3)

1. New behaviour goes in new packages.
2. Upstream files are edited only for the seams listed below, for wiring, or
   for a bug fix that is also sent upstream.
3. Every touched upstream file appears in the table below with the reason.
4. No reformatting, renaming or moving of upstream code.

## Touched upstream files

| File | Reason |
|---|---|
| `internal/config/loader.go` | identity variables (`AppDirName`, `EnvConfigDirName`, `RepoConfigDirName`) replace the jjui literals |
| `.gitignore` | ignore `/jjj` and `/bin/` |
| `internal/ui/ui.go` | `Extension` hook (`Extension`, `Core`, `NewWithExtension`, `inNormalMode`, `WindowTitlePrefix`) and its three call sites in `Init`/`Update`/`View`; `Update`'s return value became a named result (`cmd tea.Cmd`) with a deferred `cmd = tea.Batch(extCmd, cmd)` so the extension's command is folded into every return path, not just one. P2a: an extension implementing `common.ScopeProvider` has its scopes inserted before the revisions scopes in `dispatchScopes` (revisions-primary branch only), and one implementing `common.SelectionProvider` is consulted in `selectionProviders` before revisions. P2a final review: `inNormalMode` split into `len(sequenceHelp) == 0 && noModalOpen()` (pure extraction); `noModalOpen` gates the extension scopes in `dispatchScopes` (no jjj key inside the diff overlay, split-pane focus, stacked views, password, status input or a revision operation) and is exposed as the new `Core.InNormalMode` field, which the shell uses to leave clicks in upstream's inline details / split pane alone. The pending-sequence clause is left out so the scope set does not change mid-sequence |
| `internal/ui/ui_extension_test.go` | new test file: extension scope order, selection, and absence of extension scopes while a stacked view, the diff overlay, the oplog or split-pane focus is active |
| `internal/jj/jjj_commands_test.go` | new test file for `jjj_commands.go` |
| `internal/ui/bookmarkpane/advance_test.go` | new test file for `advance.go` |
| `internal/ui/intents/jjj_intents.go` | new file: jjj intent types and `//jjui:bind` annotations (the generator scans only this directory); no upstream intent file edited |
| `internal/jj/jjj_commands.go` | new file: jjj command builders (`BookmarkAdvance`, `FileAnnotate`, `ResolveFiles`, `DiffEditFiles`, `BookmarkOnlyIn`, `GitRemoteListReadOnly`, `TagListJournal`) |
| `internal/ui/actions/catalog_gen.go`, `internal/ui/actionmeta/builtins_gen.go`, `internal/config/default/types.lua` | regenerated with jjj entries — after every upstream sync run `go run ./cmd/genactions`, never hand-merge |
| `internal/jj/commands.go` | `LogTemplate` extracted from `Log` (pure extraction) so other row sources emit parser-compatible output |
| `internal/ui/revisions/revisions.go` | `RowSource` interface, `RowSourceFactory`, `DefaultRowSource`, `SetRowSourceFactory`; `streamer` fields typed as the interface |
| `internal/ui/help/help.go`, `internal/ui/help/builder.go` | `help.Intro` (groups drawn before the keybinding groups) and `ScopeGroup.Wide` (one entry per line, description wrapped, kept through the search filter); jjj fills `Intro` from `shell.Guide` in `cmd/jjj/main.go` |
| `internal/ui/bookmarkpane/bookmarkpane.go`, `internal/ui/bookmarkpane/view.go` | jjj journal view hooks: `Model.jjj` field (nil upstream); `pinTree()` after each tree sort/load; `ViewRect` drops the heading and the spacer row and draws the remote line through `renderJournalRemotes` (always shown, with fetch ages) when `jjj` is set; `renderListRow` hands rows to `renderJournalTopRow`/`renderJournalRemoteRow` when `jjj` is set |
| `internal/ui/bookmarkpane/jjj_view.go` (+ test) | new file: journal view — pinned (trunk) bookmarks first, per-remote sync markers (`=` `↑n` `↓n` `⇅a/b`) from `SetSync`, rows clipped to width; `SetFetchTimes` draws each remote's last-fetch age on the always-shown remote line |
| `internal/ui/bookmarkpane/advance.go` | new file: `Model.Advance` (`jj bookmark advance`, optional push) for the jjj Bookmarks panel; uses the pane's private selection helpers; upstream PR candidate |

## Sync procedure (manual until P6 scripts it)

    jj git fetch --remote upstream
    jj rebase -s 'roots(main@upstream..main)' -d main@upstream
    go run ./cmd/genactions
    make check
    jj bookmark set main -r 'heads(main@upstream..)'
    jj git push

Checklist before pushing the rebase:

- [ ] diff `ui.Model`'s fields and `dispatchScopes()` against `noModalOpen()`
      (which `inNormalMode()` and `Core.InNormalMode` wrap) — a new upstream
      modal must be added there or the shell will steal keys and clicks
      inside it.

## Key collisions and forced settings

jjj keys are actions in jjj scopes (`internal/jjj/defaults.toml`). The focused
panel's scope comes first, then `jjj.shell` (LeakAll, Global), then upstream's
scopes. Collisions resolved by order: `tab`/`shift+tab` (upstream `ui.focus_next_pane`,
and `revisions.details` `tab`) and digits are claimed by jjj while no upstream
operation is open; inside Files, `tab` is bound in `jjj.files` because the
details scope sits between it and `jjj.shell`; inside Bookmarks, upstream's
`bookmark_pane` binds `tab` to `ui.focus_next_pane`, and `jjj.shell` handles
that intent (cycle to the next panel) before `ui`'s no-op handler.

`shell.Attach` forces `config.Current.Revisions.LogBatching = true`: journal
mode is delivered entirely through the streaming row source, and
`revisions.Model.refresh` only consults that source on the streaming
(`log_batching = true`) path — the non-streaming `load()` calls `jj.Log`
directly and would silently bypass journal mode. A user's carried-over
`log_batching = false` (e.g. from a jjui config) is overridden.

## Known cosmetic quirks

- `p` (upstream preview toggle) opens a second preview inside the Journal
  panel; harmless, close it with `p` again. jjj ships `show_at_start = false`
  for this reason.
