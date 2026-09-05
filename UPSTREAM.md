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

## Sync procedure (manual until P6 scripts it)

    jj git fetch --remote upstream
    jj rebase -s 'roots(main@upstream..main)' -d main@upstream
    make check
    jj bookmark set main -r 'heads(main@upstream..)'
    jj git push
