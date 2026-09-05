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
| `internal/ui/ui.go` | `Extension` hook (`Extension`, `Core`, `NewWithExtension`, `inNormalMode`, `WindowTitlePrefix`) and its three call sites in `Init`/`Update`/`View` |
| `internal/jj/commands.go` | `LogTemplate` extracted from `Log` (pure extraction) so other row sources emit parser-compatible output |
| `internal/ui/revisions/revisions.go` | `RowSource` interface, `RowSourceFactory`, `DefaultRowSource`, `SetRowSourceFactory`; `streamer` fields typed as the interface |

## Sync procedure (manual until P6 scripts it)

    jj git fetch --remote upstream
    jj rebase -s 'roots(main@upstream..main)' -d main@upstream
    make check
    jj bookmark set main -r 'heads(main@upstream..)'
    jj git push

## Key collisions

Shell keys are intercepted raw (not through upstream's bindings) while the UI
is in normal mode: `0` `1` `4`, `tab`, `shift+tab`, `J`, `{`, `}`, and, with
Detail focused, `j` `k` `ctrl+d` `ctrl+u` `pgup` `pgdown`. Upstream's `tab`
(split focus) is therefore unreachable in shell mode; the Detail panel is the
preview. Tracked as a backlog task: make them rebindable actions.

## Known cosmetic quirks

- `p` (upstream preview toggle) opens a second preview inside the Journal
  panel; harmless, close it with `p` again. jjj ships `show_at_start = false`
  for this reason.
