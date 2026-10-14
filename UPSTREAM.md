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
| `internal/ui/ui.go` | `Extension` hook (`Extension`, `Core`, `NewWithExtension`, `inNormalMode`, `WindowTitlePrefix`) and its three call sites in `Init`/`Update`/`View`; `Update`'s return value became a named result (`cmd tea.Cmd`) with a deferred `cmd = tea.Batch(extCmd, cmd)` so the extension's command is folded into every return path, not just one. P2a: an extension implementing `common.ScopeProvider` has its scopes inserted before the revisions scopes in `dispatchScopes` (revisions-primary branch only), and one implementing `common.SelectionProvider` is consulted in `selectionProviders` before revisions |
| `internal/ui/intents/jjj_intents.go` | new file: jjj intent types and `//jjui:bind` annotations (the generator scans only this directory); no upstream intent file edited |
| `internal/jj/jjj_commands.go` | new file: jjj command builders (`BookmarkAdvance`, `FileAnnotate`, `ResolveFiles`, `DiffEditFiles`) |
| `internal/ui/actions/catalog_gen.go`, `internal/ui/actionmeta/builtins_gen.go`, `internal/config/default/types.lua` | regenerated with jjj entries — after every upstream sync run `go run ./cmd/genactions`, never hand-merge |
| `internal/jj/commands.go` | `LogTemplate` extracted from `Log` (pure extraction) so other row sources emit parser-compatible output |
| `internal/ui/revisions/revisions.go` | `RowSource` interface, `RowSourceFactory`, `DefaultRowSource`, `SetRowSourceFactory`; `streamer` fields typed as the interface |

## Sync procedure (manual until P6 scripts it)

    jj git fetch --remote upstream
    jj rebase -s 'roots(main@upstream..main)' -d main@upstream
    go run ./cmd/genactions
    make check
    jj bookmark set main -r 'heads(main@upstream..)'
    jj git push

Checklist before pushing the rebase:

- [ ] diff `ui.Model`'s fields and `dispatchScopes()` against `inNormalMode()`
      — a new upstream modal must be added there or the shell will steal keys
      inside it.

## Key collisions and forced settings

Shell keys are intercepted raw (not through upstream's bindings) while the UI
is in normal mode: `0` `1` `4`, `tab`, `shift+tab`, `J`, `{`, `}`, and, with
Detail focused, `j` `k` `up` `down` `ctrl+d` `ctrl+u` `pgup` `pgdown`.
Upstream's `tab` (split focus) is therefore unreachable in shell mode; the
Detail panel is the preview. Tracked as a backlog task: make them rebindable
actions.

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
