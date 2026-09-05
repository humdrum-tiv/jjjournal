# jjjournal P0 + P1 Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Ship `jjj`: jjui with its own identity, a three-panel lazygit-style shell (Status, Journal, Detail), and journal mode (full history sorted by time with day and week headers), installed to `~/bin/jjj` and replacing the `jjournal` wrapper.

**Architecture:** A second entrypoint `cmd/jjj` boots upstream's root UI model with an `Extension` hook (new, ~40 lines in `internal/ui/ui.go`) that lets our `internal/ui/shell` package own layout, focus and panel keys while upstream keeps owning input dispatch, operations, Lua and modals. Journal mode plugs into a new `RowSource` seam in `internal/ui/revisions` so every upstream operation works on journal rows unchanged. All new behaviour lives in new packages; upstream files get only the seams and identity variables listed in `UPSTREAM.md`.

**Tech Stack:** Go 1.25 (toolchain 1.26 installed), Bubble Tea v2 / Lipgloss v2 (`charm.land/*`), BurntSushi TOML, `stretchr/testify` for tests, `jj` 0.43 on PATH, Python 3 for the PTY smoke script.

**Spec:** `docs/superpowers/specs/2026-09-04-jjjournal-design.md` — this plan implements §2, §3.1 (panels 0, 1, 4 only), §3.2 (focus, digits, tab; zoom is P2), §3.3, §4, §7 (Status panel), §11, §12, §14 rows P0 and P1, §15. Files, Bookmarks, Notes, Stats, Tasks, daily-note export and the sync script are later plans (P2–P6).

## Global Constraints

- Every VCS mutation goes through `jj` (this machine is jj-only; a hook blocks mutating `git`). Commit with `jj commit -m "..."` then `jj bookmark set main -r @-`. Never `git add`/`git commit`.
- Upstream files may be edited only for the seams named in this plan. Every touched upstream file is listed in `UPSTREAM.md` with the reason (spec §12.3). No reformatting or renaming of upstream code.
- `go build ./cmd/jjui` must keep succeeding after every task (spec §13, "upstream intact").
- Go module path stays `github.com/idursun/jjui` (spec §2).
- jjj reads `~/.config/jjj/config.toml`, `JJJ_CONFIG_DIR`, and `<repo>/.jjj/`; it never reads jjui's directories (spec §11).
- Default revset `all()`; row templates are compiled in; author is never part of a row (spec §4).
- The repo root is `/Users/kortum/Developer/Home/jjjournal`. Run every command from there unless a step says otherwise.
- Tests: `go test ./...` from the root; `stretchr/testify` is already a dependency. Do not add new module dependencies.
- The e2e suite under `e2e/` needs Docker and is not part of this plan; `scripts/pty-smoke.py` is the end-to-end check here.

---

## File structure

New files (ours):

| Path | Responsibility |
|---|---|
| `cmd/jjj/main.go` | entrypoint: sets identity vars, applies jjj defaults, loads jjj config, boots `ui.NewWithExtension` |
| `internal/jjj/defaults.toml`, `internal/jjj/config.go`, `internal/jjj/config_test.go` | compiled-in jjj defaults applied to upstream config; jjj-only `[journal]`/`[shell]`/`[tasks]`/`[stats]` config |
| `internal/journal/template.go`, `template_test.go` | the two bundled row templates |
| `internal/journal/timestamps.go`, `timestamps_test.go` | jj args + parser for commit times keyed by short commit id |
| `internal/journal/arrange.go`, `arrange_test.go` | pure logic: sort rows by time, insert day/week header lines |
| `internal/journal/source.go`, `source_test.go` | `revisions.RowSource` that runs `jj log --no-graph` and arranges |
| `internal/ui/shell/layout.go`, `layout_test.go` | pure layout math for the panel boxes |
| `internal/ui/shell/shell.go` | the `ui.Extension`: panels, focus, keys, render |
| `internal/statusinfo/statusinfo.go`, `statusinfo_test.go` | pure logic for the Status panel: jj args, parsers, formatting |
| `internal/ui/statuspanel/statuspanel.go` | Status panel model (3 lines) |
| `Makefile`, `UPSTREAM.md`, `scripts/pty-smoke.py`, `docs/jjj.md`, `CLAUDE.md`, `backlog/` | build, upstream record, smoke test, quickstart, house bookkeeping |

Upstream files modified (seams only):

| Path | Change |
|---|---|
| `internal/config/loader.go` | three identity variables replace the `"jjui"`, `"JJUI_CONFIG_DIR"`, `".jjui"` literals |
| `internal/ui/ui.go` | `Extension` interface, `Core`, `NewWithExtension`, `inNormalMode`, `WindowTitlePrefix`, hook calls in `Init`/`Update`/`View` |
| `internal/ui/revisions/revisions.go` | `RowSource` interface, `RowSourceFactory`, `DefaultRowSource`, `SetRowSourceFactory`; `streamer` fields typed as `RowSource` |
| `internal/jj/commands.go` | `LogTemplate(jjTemplate string) string` extracted from `Log` |
| `.gitignore` | add `/jjj` and `/bin/` |

---

### Task 1: Identity seam, `cmd/jjj` entrypoint, Makefile, smoke script

**Files:**
- Modify: `internal/config/loader.go:23-66` and `:227-238`
- Create: `internal/config/identity_test.go`
- Create: `cmd/jjj/main.go`
- Create: `Makefile`, `scripts/pty-smoke.py`, `UPSTREAM.md`
- Modify: `.gitignore`

**Interfaces:**
- Produces: `config.AppDirName`, `config.EnvConfigDirName`, `config.RepoConfigDirName` (package-level `string` vars, defaults `"jjui"`, `"JJUI_CONFIG_DIR"`, `".jjui"`).
- Produces: `cmd/jjj/main.go` `run() int` mirroring `cmd/jjui/main.go`; later tasks add lines to it at the markers `// jjj: defaults` and `// jjj: extension`.
- Produces: `scripts/pty-smoke.py BIN REPO --expect TEXT [--expect TEXT ...] [--wait SECS] [--key K --after SECS]` exiting 0 when every expected text appears on the final frame.

- [ ] **Step 1: Write the failing identity test**

Create `internal/config/identity_test.go`:

```go
package config

import (
	"os"
	"path/filepath"
	"testing"
)

func TestIdentityVarsSelectConfigPaths(t *testing.T) {
	origApp, origEnv, origRepo := AppDirName, EnvConfigDirName, RepoConfigDirName
	t.Cleanup(func() { AppDirName, EnvConfigDirName, RepoConfigDirName = origApp, origEnv, origRepo })
	AppDirName, EnvConfigDirName, RepoConfigDirName = "jjj", "JJJ_CONFIG_DIR", ".jjj"

	home := t.TempDir()
	t.Setenv("HOME", home)
	t.Setenv("XDG_CONFIG_HOME", "")
	t.Setenv("JJUI_CONFIG_DIR", t.TempDir()) // jjui's variable must be ignored
	t.Setenv("JJJ_CONFIG_DIR", "")

	want := filepath.Join(home, ".config", "jjj", "config.toml")
	if got := getConfigFilePath(); got != want {
		t.Fatalf("global config path: got %q want %q", got, want)
	}

	envDir := t.TempDir()
	t.Setenv("JJJ_CONFIG_DIR", envDir)
	if got, want := getConfigFilePath(), filepath.Join(envDir, "config.toml"); got != want {
		t.Fatalf("env config path: got %q want %q", got, want)
	}

	repo := t.TempDir()
	if err := os.MkdirAll(filepath.Join(repo, ".jjj"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(repo, ".jjj", "config.toml"), []byte("limit = 7\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	data, err := LoadRepoConfigFile(repo)
	if err != nil {
		t.Fatalf("LoadRepoConfigFile: %v", err)
	}
	if string(data) != "limit = 7\n" {
		t.Fatalf("repo config: got %q", data)
	}
}
```

- [ ] **Step 2: Run it to verify it fails**

Run: `go test ./internal/config/ -run TestIdentityVarsSelectConfigPaths`
Expected: FAIL to compile with `undefined: AppDirName`.

- [ ] **Step 3: Add the identity variables to the loader**

In `internal/config/loader.go`, insert after the `mergeOverlay` type (line 22):

```go
// Identity of the running application. cmd/jjj overrides these before any
// config is read so that jjj uses ~/.config/jjj, JJJ_CONFIG_DIR and <repo>/.jjj
// instead of jjui's directories. Upstream jjui leaves them at their defaults.
var (
	AppDirName        = "jjui"            // ~/.config/<AppDirName>/config.toml
	EnvConfigDirName  = "JJUI_CONFIG_DIR" // environment variable overriding the config dir
	RepoConfigDirName = ".jjui"           // <repo>/<RepoConfigDirName>/config.toml
)
```

Then replace, keeping everything else byte-for-byte:

- line 25: `os.Getenv("JJUI_CONFIG_DIR")` → `os.Getenv(EnvConfigDirName)`
- line 57: `filepath.Join(dir, "jjui", "config.toml")` → `filepath.Join(dir, AppDirName, "config.toml")`
- line 64: `filepath.Join(configDirs[0], "jjui", "config.toml")` → `filepath.Join(configDirs[0], AppDirName, "config.toml")`
- line 228: `filepath.Join(repoRoot, ".jjui", "config.toml")` → `filepath.Join(repoRoot, RepoConfigDirName, "config.toml")`
- line 237: `filepath.Join(repoRoot, ".jjui", "config.lua")` → `filepath.Join(repoRoot, RepoConfigDirName, "config.lua")`

Verify nothing else is left: `grep -n '"jjui"\|JJUI_CONFIG_DIR\|\.jjui' internal/config/loader.go` must print only the comment lines and the var block.

- [ ] **Step 4: Run the test to verify it passes**

Run: `go test ./internal/config/`
Expected: `ok` (the whole package, so upstream tests still pass).

- [ ] **Step 5: Create `cmd/jjj/main.go`**

Copy `cmd/jjui/main.go` to `cmd/jjj/main.go` (`cp cmd/jjui/main.go cmd/jjj/main.go`) and make exactly these edits:

1. In `init()`, the usage line: `fmt.Printf("Usage: jjui [flags] [location]\n")` → `fmt.Printf("Usage: jjj [flags] [location]\n")`.
2. In `run()`, replace the first two lines (`os.Setenv("JJUI", "1")` and the comment above it) with:

```go
	// Identity: jjj reads its own config directories. Set before anything
	// touches config. JJUI=1 stays set so `jj` conditionals written for jjui
	// keep working under jjj.
	config.AppDirName = "jjj"
	config.EnvConfigDirName = "JJJ_CONFIG_DIR"
	config.RepoConfigDirName = ".jjj"
	os.Setenv("JJUI", "1")
	os.Setenv("JJJ", "1")
```

3. The error hint `Please pass the location of a `+"`jj`"+` repo as an argument to `+"`jjui`"+`.` → say `jjj`.
4. `repoConfigDir := filepath.Join(rootLocation, ".jjui")` → `repoConfigDir := filepath.Join(rootLocation, config.RepoConfigDirName)`.
5. Directly after the block that loads the global config file (the `if output, err := config.LoadConfigFile(); err == nil { ... } else if ... { return 1 }` block), add a marker comment line on its own: `// jjj: defaults` — Task 2 inserts code above and below it.
6. Replace `p := tea.NewProgram(ui.New(appContext), tea.WithInput(os.Stdin))` with:

```go
	// jjj: extension
	p := tea.NewProgram(ui.New(appContext), tea.WithInput(os.Stdin))
```

Build: `go build ./cmd/jjj && ./jjj -version` → prints `unknown` (no build flags) or a VCS version. Then `rm -f jjj`.

- [ ] **Step 6: Add `.gitignore` entries, Makefile, UPSTREAM.md**

Append to `.gitignore`:

```
/jjj
/bin/
```

Create `Makefile`:

```make
BIN ?= $(HOME)/bin
REPO ?= $(HOME)/Developer/Home/ticktock

.PHONY: build install check smoke

build:
	mkdir -p bin
	go build -o bin/jjj ./cmd/jjj

install:
	mkdir -p $(BIN)
	go build -o $(BIN)/jjj ./cmd/jjj

# Everything that must stay green: our tests, upstream's tests, and the
# untouched upstream binary.
check:
	go vet ./...
	go build ./...
	go build -o /dev/null ./cmd/jjui
	go test ./...

smoke: build
	python3 scripts/pty-smoke.py bin/jjj $(REPO) --wait 12 --expect "revset:"
```

Create `UPSTREAM.md`:

```markdown
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
```

- [ ] **Step 7: Create `scripts/pty-smoke.py`**

```python
#!/usr/bin/env python3
"""Launch a TUI binary in a pseudo-terminal and assert text on its last frame.

    scripts/pty-smoke.py bin/jjj ~/Developer/Home/ticktock --expect "revset:" \
        --wait 12 --key j --after 3 --expect "Commit ID"

Keys are sent in order after --wait seconds, each followed by --after seconds.
Exit status 0 when every --expect string appears on the final frame; otherwise
the cleaned frame is printed and the status is 1.
"""
import argparse
import fcntl
import os
import pty
import re
import select
import signal
import struct
import sys
import termios
import time

ANSI = re.compile(r"\x1b\[[0-9;?>=]*[ -/]*[@-~]|\x1b\][^\x07]*\x07|\x1b[=>]")


def pump(fd, seconds, out):
    end = time.time() + seconds
    while time.time() < end:
        ready, _, _ = select.select([fd], [], [], 0.2)
        if ready:
            try:
                out.append(os.read(fd, 65536))
            except OSError:
                return


def main():
    ap = argparse.ArgumentParser()
    ap.add_argument("binary")
    ap.add_argument("repo")
    ap.add_argument("--expect", action="append", default=[])
    ap.add_argument("--wait", type=float, default=8.0)
    ap.add_argument("--key", action="append", default=[])
    ap.add_argument("--after", type=float, default=3.0)
    ap.add_argument("--cols", type=int, default=200)
    ap.add_argument("--rows", type=int, default=45)
    args = ap.parse_args()

    pid, fd = pty.fork()
    if pid == 0:
        os.environ["TERM"] = "xterm-256color"
        os.chdir(args.repo)
        os.execvp(args.binary, [args.binary])
    fcntl.ioctl(fd, termios.TIOCSWINSZ, struct.pack("HHHH", args.rows, args.cols, 0, 0))
    chunks = []
    pump(fd, args.wait, chunks)
    for key in args.key:
        os.write(fd, key.encode())
        pump(fd, args.after, chunks)
    try:
        os.write(fd, b"q")
        pump(fd, 0.7, chunks)
    except OSError:
        pass
    try:
        os.kill(pid, signal.SIGTERM)
    except ProcessLookupError:
        pass

    text = ANSI.sub("", b"".join(chunks).decode("utf-8", "replace")).replace("\r", "")
    lines = [l.rstrip() for l in text.split("\n") if l.strip()]
    frame = "\n".join(lines)
    missing = [e for e in args.expect if e not in frame]
    if missing:
        print(frame[-6000:])
        print("\nMISSING:", missing, file=sys.stderr)
        return 1
    print("ok:", ", ".join(args.expect))
    return 0


if __name__ == "__main__":
    sys.exit(main())
```

`chmod +x scripts/pty-smoke.py`.

- [ ] **Step 8: Prove the identity end to end**

```bash
make build
mkdir -p /tmp/jjjcfg && printf '[revisions]\nrevset = "all()"\n' > /tmp/jjjcfg/config.toml
JJJ_CONFIG_DIR=/tmp/jjjcfg python3 scripts/pty-smoke.py bin/jjj ~/Developer/Home/ticktock --wait 6 --expect "revset: all()"
JJUI_CONFIG_DIR=/tmp/jjjcfg python3 scripts/pty-smoke.py bin/jjj ~/Developer/Home/ticktock --wait 6 --expect "revset: present(@)"
make check
```

Expected: first smoke prints `ok: revset: all()`; second prints `ok: revset: present(@)` (jjui's variable is ignored by jjj); `make check` ends with all packages `ok`.

- [ ] **Step 9: Commit**

```bash
jj commit -m "feat(jjj): identity seam and cmd/jjj entrypoint

config gains AppDirName/EnvConfigDirName/RepoConfigDirName so a second
binary can own its config directories. cmd/jjj mirrors cmd/jjui with jjj
identity. Makefile, UPSTREAM.md and scripts/pty-smoke.py added."
jj bookmark set main -r @-
```

---

### Task 2: Compiled-in jjj defaults, jjj config, bundled row templates

**Files:**
- Create: `internal/journal/template.go`, `internal/journal/template_test.go`
- Create: `internal/jjj/defaults.toml`, `internal/jjj/config.go`, `internal/jjj/config_test.go`
- Modify: `cmd/jjj/main.go` at the `// jjj: defaults` marker

**Interfaces:**
- Produces: `journal.RowTemplateGraph`, `journal.RowTemplateJournal` (`const string`, jj template expressions ending in `++ "\n"`).
- Produces: `jjj.Config{Journal JournalConfig; Shell ShellConfig; Tasks TasksConfig; Stats StatsConfig}`, `jjj.Current *Config`, `func (c *Config) Load(data string) error`, `func ApplyDefaults(upstream *config.Config) error`.
- `JournalConfig{Enabled bool; Template string; Limit int; DailyNote string; Callout string; CreateNote bool}`; `ShellConfig{Enabled bool; NarrowColumns int; LeftWidth float64; NotesRows int; Hidden []string}`; `TasksConfig{Enabled bool; ShowDone bool}`; `StatsConfig{Weeks int}`.

- [ ] **Step 1: Write the template test (runs the real jj)**

Create `internal/journal/template_test.go`:

```go
package journal

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

// newJJRepo creates a throwaway colocated-free jj repo with one described commit.
func newJJRepo(t *testing.T) string {
	t.Helper()
	if _, err := exec.LookPath("jj"); err != nil {
		t.Skip("jj not on PATH")
	}
	dir := t.TempDir()
	run := func(args ...string) string {
		cmd := exec.Command("jj", args...)
		cmd.Dir = dir
		cmd.Env = append(os.Environ(), "JJ_USER=t", "JJ_EMAIL=t@example.com", "JJ_CONFIG="+filepath.Join(dir, "no-config.toml"))
		out, err := cmd.CombinedOutput()
		if err != nil {
			t.Fatalf("jj %v: %v\n%s", args, err, out)
		}
		return string(out)
	}
	run("git", "init")
	if err := os.WriteFile(filepath.Join(dir, "a.txt"), []byte("a\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	run("commit", "-m", "feat: first")
	return dir
}

func TestRowTemplatesRenderOneLine(t *testing.T) {
	dir := newJJRepo(t)
	for name, tmpl := range map[string]string{"graph": RowTemplateGraph, "journal": RowTemplateJournal} {
		cmd := exec.Command("jj", "log", "-r", "@-", "--no-graph", "--color", "never", "-T", tmpl)
		cmd.Dir = dir
		cmd.Env = append(os.Environ(), "JJ_CONFIG="+filepath.Join(dir, "no-config.toml"))
		out, err := cmd.CombinedOutput()
		if err != nil {
			t.Fatalf("%s template failed: %v\n%s", name, err, out)
		}
		line := strings.TrimRight(string(out), "\n")
		if strings.Contains(line, "\n") {
			t.Fatalf("%s template rendered more than one line: %q", name, line)
		}
		if !strings.Contains(line, "feat: first") {
			t.Fatalf("%s template lost the subject: %q", name, line)
		}
	}
}

func TestJournalTemplateShowsWorkingCopyMarker(t *testing.T) {
	dir := newJJRepo(t)
	cmd := exec.Command("jj", "log", "-r", "@", "--no-graph", "--color", "never", "-T", RowTemplateJournal)
	cmd.Dir = dir
	cmd.Env = append(os.Environ(), "JJ_CONFIG="+filepath.Join(dir, "no-config.toml"))
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("%v\n%s", err, out)
	}
	if !strings.Contains(string(out), "(working copy)") {
		t.Fatalf("empty undescribed @ should read (working copy): %q", out)
	}
}
```

- [ ] **Step 2: Run to verify it fails**

Run: `go test ./internal/journal/`
Expected: compile error `undefined: RowTemplateGraph`.

- [ ] **Step 3: Write the templates**

Create `internal/journal/template.go`:

```go
// Package journal turns jj log output into a time-sorted, day-grouped list.
package journal

// RowTemplateGraph is the compiled-in row template for graph mode: one line
// per change with the full local date. `format_short_change_id_with_change_offset`
// and `description_placeholder` are jj built-in template aliases (jj >= 0.37).
const RowTemplateGraph = `separate(" ", format_short_change_id_with_change_offset(self), committer.timestamp().local().format("%a %Y-%m-%d %H:%M"), bookmarks, tags, working_copies, if(conflict, label("conflict", "conflict")), if(empty, label("empty", "(empty)")), if(description, description.first_line(), label(if(empty, "empty"), if(current_working_copy, "(working copy)", description_placeholder)))) ++ "\n"`

// RowTemplateJournal is the journal-mode row: time only, because the date is
// carried by the day header above the first row of each day.
const RowTemplateJournal = `separate(" ", format_short_change_id_with_change_offset(self), committer.timestamp().local().format("%H:%M"), bookmarks, tags, working_copies, if(conflict, label("conflict", "conflict")), if(empty, label("empty", "(empty)")), if(description, description.first_line(), label(if(empty, "empty"), if(current_working_copy, "(working copy)", description_placeholder)))) ++ "\n"`
```

- [ ] **Step 4: Run the template tests**

Run: `go test ./internal/journal/ -v`
Expected: both tests PASS (they shell out to the installed jj).

- [ ] **Step 5: Write the failing jjj config test**

Create `internal/jjj/config_test.go`:

```go
package jjj

import (
	"testing"

	"github.com/idursun/jjui/internal/config"
	"github.com/idursun/jjui/internal/journal"
	"github.com/stretchr/testify/require"
)

func TestApplyDefaultsSetsJournalDefaults(t *testing.T) {
	upstream := &config.Config{}
	Current = &Config{}
	require.NoError(t, ApplyDefaults(upstream))

	require.Equal(t, "all()", upstream.Revisions.Revset)
	require.Equal(t, journal.RowTemplateGraph, upstream.Revisions.Template)
	require.False(t, upstream.Preview.ShowAtStart)
	require.Equal(t, []string{"show", "--color", "always", "-r", "$change_id", "--stat", "--git"}, upstream.Preview.RevisionCommand)
	require.Equal(t, []string{"diff", "--color", "always", "-r", "$change_id", "--git", "$file"}, upstream.Preview.FileCommand)

	require.True(t, Current.Journal.Enabled)
	require.Equal(t, 5000, Current.Journal.Limit)
	require.Equal(t, "~/Humdrum/Daily/{{date}}.md", Current.Journal.DailyNote)
	require.Equal(t, "journal", Current.Journal.Callout)
	require.True(t, Current.Journal.CreateNote)
	require.True(t, Current.Shell.Enabled)
	require.Equal(t, 120, Current.Shell.NarrowColumns)
	require.InDelta(t, 0.34, Current.Shell.LeftWidth, 0.001)
	require.Equal(t, 8, Current.Shell.NotesRows)
	require.Empty(t, Current.Shell.Hidden)
	require.True(t, Current.Tasks.Enabled)
	require.False(t, Current.Tasks.ShowDone)
	require.Equal(t, 26, Current.Stats.Weeks)
}

func TestLoadOverridesOnlyGivenKeys(t *testing.T) {
	Current = &Config{}
	require.NoError(t, ApplyDefaults(&config.Config{}))
	require.NoError(t, Current.Load("[journal]\nlimit = 10\n[shell]\nhidden = [\"files\"]\n"))
	require.Equal(t, 10, Current.Journal.Limit)
	require.Equal(t, []string{"files"}, Current.Shell.Hidden)
	require.True(t, Current.Journal.Enabled, "untouched keys keep their defaults")
	require.Equal(t, 26, Current.Stats.Weeks)
}

func TestLoadIgnoresUpstreamTables(t *testing.T) {
	Current = &Config{}
	require.NoError(t, Current.Load("[revisions]\nrevset = \"mine()\"\n[ui]\ntheme = \"x\"\n"))
}
```

- [ ] **Step 6: Run to verify it fails**

Run: `go test ./internal/jjj/`
Expected: compile error (package does not exist).

- [ ] **Step 7: Write defaults.toml and config.go**

Create `internal/jjj/defaults.toml`:

```toml
# jjj compiled-in defaults. Applied on top of upstream's embedded defaults and
# before ~/.config/jjj/config.toml and <repo>/.jjj/config.toml, so every key
# here can be overridden there. The row templates live in Go
# (internal/journal/template.go) and are applied by ApplyDefaults.

[revisions]
revset = "all()"

[preview]
show_at_start = false
revision_command = ["show", "--color", "always", "-r", "$change_id", "--stat", "--git"]
file_command = ["diff", "--color", "always", "-r", "$change_id", "--git", "$file"]

[journal]
enabled = true
limit = 5000
daily_note = "~/Humdrum/Daily/{{date}}.md"
callout = "journal"
create_note = true

[shell]
enabled = true
narrow_columns = 120
left_width = 0.34
notes_rows = 8
hidden = []

[tasks]
enabled = true
show_done = false

[stats]
weeks = 26
```

Create `internal/jjj/config.go`:

```go
// Package jjj holds what makes jjj different from jjui at configuration
// level: compiled-in defaults and the jjj-only config tables.
package jjj

import (
	_ "embed"

	"github.com/BurntSushi/toml"
	"github.com/idursun/jjui/internal/config"
	"github.com/idursun/jjui/internal/journal"
)

//go:embed defaults.toml
var defaultsTOML string

type JournalConfig struct {
	Enabled    bool   `toml:"enabled"`
	Template   string `toml:"template"` // empty → journal.RowTemplateJournal
	Limit      int    `toml:"limit"`
	DailyNote  string `toml:"daily_note"`
	Callout    string `toml:"callout"`
	CreateNote bool   `toml:"create_note"`
}

type ShellConfig struct {
	Enabled       bool     `toml:"enabled"`
	NarrowColumns int      `toml:"narrow_columns"`
	LeftWidth     float64  `toml:"left_width"`
	NotesRows     int      `toml:"notes_rows"`
	Hidden        []string `toml:"hidden"`
}

type TasksConfig struct {
	Enabled  bool `toml:"enabled"`
	ShowDone bool `toml:"show_done"`
}

type StatsConfig struct {
	Weeks int `toml:"weeks"`
}

// Config is the jjj-only part of config.toml. Upstream tables in the same
// file are decoded by config.Config; unknown tables are ignored on both sides.
type Config struct {
	Journal JournalConfig `toml:"journal"`
	Shell   ShellConfig   `toml:"shell"`
	Tasks   TasksConfig   `toml:"tasks"`
	Stats   StatsConfig   `toml:"stats"`
}

// Current is the live jjj config, filled by ApplyDefaults and then by each
// Load of the user's global and repo config text.
var Current = &Config{}

// Load merges the jjj tables from a config.toml text into c.
func (c *Config) Load(data string) error {
	_, err := toml.Decode(data, c)
	return err
}

// ApplyDefaults loads defaults.toml into the upstream config (revset, preview
// commands) and into Current, and installs the bundled graph-mode row template.
func ApplyDefaults(upstream *config.Config) error {
	if err := upstream.Load(defaultsTOML, ""); err != nil {
		return err
	}
	upstream.Revisions.Template = journal.RowTemplateGraph
	return Current.Load(defaultsTOML)
}
```

- [ ] **Step 8: Run the jjj config tests**

Run: `go test ./internal/jjj/ ./internal/config/`
Expected: PASS.

- [ ] **Step 9: Wire defaults and jjj config into `cmd/jjj/main.go`**

Replace the `// jjj: defaults` marker line and the global-config block just above it so the sequence reads:

```go
	if err := jjj.ApplyDefaults(config.Current); err != nil {
		fmt.Fprintf(os.Stderr, "Error applying jjj defaults: %v\n", err)
		return 1
	}
	if output, err := config.LoadConfigFile(); err == nil {
		if err := config.Current.Load(string(output), config.GetConfigDir()); err != nil {
			fmt.Fprintf(os.Stderr, "Error loading configuration: %v\n", err)
			return 1
		}
		if err := jjj.Current.Load(string(output)); err != nil {
			fmt.Fprintf(os.Stderr, "Error loading jjj configuration: %v\n", err)
			return 1
		}
		for _, warning := range config.DeprecatedConfigWarnings(string(output)) {
			fmt.Fprintf(os.Stderr, "Warning: %s\n", warning)
		}
	} else if !errors.Is(err, fs.ErrNotExist) {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		return 1
	}
	// jjj: defaults
```

In the repo-config block that follows (`if config.EnvConfigDir() == "" { if output, err := config.LoadRepoConfigFile(rootLocation); err == nil { ...`), add directly after the `config.Current.Load(string(output), repoConfigDir)` error check:

```go
			if err := jjj.Current.Load(string(output)); err != nil {
				fmt.Fprintf(os.Stderr, "Error loading repository jjj configuration: %v\n", err)
				return 1
			}
```

Add the import `"github.com/idursun/jjui/internal/jjj"`.

- [ ] **Step 10: Verify with the smoke script**

```bash
make build
rm -rf /tmp/jjjcfg; mkdir -p /tmp/jjjcfg   # empty config dir: only compiled-in defaults apply
JJJ_CONFIG_DIR=/tmp/jjjcfg python3 scripts/pty-smoke.py bin/jjj ~/Developer/Home/ticktock --wait 8 --expect "revset: all()" --expect "Mon 2026-07-13"
make check
```

Expected: `ok: revset: all(), Mon 2026-07-13` (one-line dated rows from the bundled graph template; ticktock's newest commit is dated 2026-07-13) and a green `make check`.

- [ ] **Step 11: Commit**

```bash
jj commit -m "feat(jjj): compiled-in defaults, jjj config tables, bundled row templates

all() revset, stat+git detail commands, [journal]/[shell]/[tasks]/[stats]
tables, and the graph/journal row templates (author deliberately omitted)."
jj bookmark set main -r @-
```

---

### Task 3: Extension hook in `ui.Model` and the three-panel shell (P0 spike, part 1)

**Files:**
- Modify: `internal/ui/ui.go` (`Model` struct :47-71, `Init` :86-88, `Update` :147-160, `View` :433-442, `wrapper.View` :814, plus new declarations)
- Create: `internal/ui/shell/layout.go`, `internal/ui/shell/layout_test.go`, `internal/ui/shell/shell.go`
- Modify: `cmd/jjj/main.go` at the `// jjj: extension` marker
- Modify: `UPSTREAM.md` (add the `ui.go` row)

**Interfaces:**
- Produces (upstream seam, package `ui`):

```go
type Extension interface {
	Attach(core Core)
	Init() tea.Cmd
	Update(msg tea.Msg) (cmd tea.Cmd, handled bool)
	HandleKey(msg tea.KeyMsg) (cmd tea.Cmd, handled bool)
	Render(dl *render.DisplayContext, box layout.Box, core Core) bool
}
type Core struct {
	Revisions  *revisions.Model
	Revset     *revset.Model
	Status     *status.Model
	RenderMain func(box layout.Box)
}
var WindowTitlePrefix = "jjui"
func NewWithExtension(c *context.MainContext, ext Extension) tea.Model
```

- Produces (package `shell`): `type PanelID int` with constants `PanelDetail=0, PanelStatus=1, PanelFiles=2, PanelBookmarks=3, PanelJournal=4, PanelTasks=5, PanelNotes=6, PanelStats=7`; `type Layout struct{ Status, Journal, Detail layout.Box }`; `const StatusHeight = 5`; `func Compute(box layout.Box, leftWidth float64) Layout`; `type Shell struct`; `func New(ctx *context.MainContext, cfg *jjj.Config) *Shell` implementing `ui.Extension`.
- Consumes: `jjj.Config.Shell.{Enabled, LeftWidth, NarrowColumns}` from Task 2.

- [ ] **Step 1: Write the failing layout tests**

Create `internal/ui/shell/layout_test.go`:

```go
package shell

import (
	"testing"

	"github.com/idursun/jjui/internal/ui/layout"
	"github.com/stretchr/testify/require"
)

func TestComputeSplitsColumnsAndStatus(t *testing.T) {
	box := layout.NewBox(layout.Rect(0, 0, 200, 50))
	lay := Compute(box, 0.34)

	require.Equal(t, 68, lay.Status.R.Dx(), "left column is 34% of 200")
	require.Equal(t, 68, lay.Journal.R.Dx())
	require.Equal(t, 132, lay.Detail.R.Dx())
	require.Equal(t, StatusHeight, lay.Status.R.Dy())
	require.Equal(t, 50-StatusHeight, lay.Journal.R.Dy())
	require.Equal(t, 50, lay.Detail.R.Dy())
	require.Equal(t, 0, lay.Status.R.Min.Y)
	require.Equal(t, StatusHeight, lay.Journal.R.Min.Y)
	require.Equal(t, 68, lay.Detail.R.Min.X)
}

func TestComputeClampsLeftWidth(t *testing.T) {
	box := layout.NewBox(layout.Rect(0, 0, 200, 50))
	require.Equal(t, 40, Compute(box, 0.05).Journal.R.Dx(), "floor 20%")
	require.Equal(t, 120, Compute(box, 0.9).Journal.R.Dx(), "ceiling 60%")
}

func TestPanelCycleOrder(t *testing.T) {
	require.Equal(t, PanelJournal, PanelStatus.next())
	require.Equal(t, PanelDetail, PanelJournal.next())
	require.Equal(t, PanelStatus, PanelDetail.next())
	require.Equal(t, PanelDetail, PanelStatus.prev())
	require.Equal(t, PanelJournal, PanelFiles.next(), "panels not in this plan fall back to Journal")
}
```

- [ ] **Step 2: Run to verify it fails**

Run: `go test ./internal/ui/shell/`
Expected: compile error (package does not exist).

- [ ] **Step 3: Write `layout.go`**

```go
// Package shell is jjj's lazygit-style layout: several panels visible at once
// around upstream's revisions list. It plugs into ui.Model through the
// ui.Extension hook and only ever sees keys while upstream is in normal mode.
package shell

import "github.com/idursun/jjui/internal/ui/layout"

// PanelID numbers panels the lazygit way (spec §3.1); the digit is the focus key.
type PanelID int

const (
	PanelDetail    PanelID = 0
	PanelStatus    PanelID = 1
	PanelFiles     PanelID = 2 // P2
	PanelBookmarks PanelID = 3 // P2
	PanelJournal   PanelID = 4
	PanelTasks     PanelID = 5 // P3
	PanelNotes     PanelID = 6 // P2
	PanelStats     PanelID = 7 // P4
)

// cycle is the tab order among the panels this plan ships.
var cycle = []PanelID{PanelStatus, PanelJournal, PanelDetail}

func (p PanelID) next() PanelID { return stepPanel(p, 1) }
func (p PanelID) prev() PanelID { return stepPanel(p, -1) }

func stepPanel(p PanelID, d int) PanelID {
	for i, id := range cycle {
		if id == p {
			return cycle[(i+d+len(cycle))%len(cycle)]
		}
	}
	return PanelJournal
}

// Layout holds the outer box (border included) of each panel for one frame.
type Layout struct {
	Status, Journal, Detail layout.Box
}

// StatusHeight is the Status panel's outer height: three content lines plus border.
const StatusHeight = 5

// Compute splits box into a left column (Status above Journal) and a right
// column (Detail). leftWidth is the left column's share, clamped to [0.2, 0.6].
func Compute(box layout.Box, leftWidth float64) Layout {
	if leftWidth < 0.2 {
		leftWidth = 0.2
	}
	if leftWidth > 0.6 {
		leftWidth = 0.6
	}
	cols := box.H(layout.Percent(int(leftWidth*100+0.5)), layout.Fill(1))
	if len(cols) < 2 {
		return Layout{Journal: box}
	}
	rows := cols[0].V(layout.Fixed(StatusHeight), layout.Fill(1))
	if len(rows) < 2 {
		return Layout{Journal: cols[0], Detail: cols[1]}
	}
	return Layout{Status: rows[0], Journal: rows[1], Detail: cols[1]}
}
```

- [ ] **Step 4: Run the layout tests**

Run: `go test ./internal/ui/shell/`
Expected: PASS. (If `TestComputeSplitsColumnsAndStatus` fails on the 68/132 split, read `internal/ui/layout/box.go:57-68` — `Percent.calc` — and adjust the expectation only if `Percent` rounds differently; the intent is "34 % of the width".)

- [ ] **Step 5: Add the Extension hook to `internal/ui/ui.go`**

a. Add `ext Extension` as the last field of `Model` (after `mode2031Supported bool`).

b. After the `scriptFrame` type (line 76), add:

```go
// Extension lets an embedding binary (jjj) add panels and keys around the
// stock model without replacing its input handling. See internal/ui/shell.
type Extension interface {
	// Attach hands the Extension the stock models right after construction,
	// before Init runs, so it can install seams (e.g. a row source factory).
	Attach(core Core)
	Init() tea.Cmd
	// Update sees every message before the default handling. handled=true
	// stops further processing; a non-nil cmd is batched either way.
	Update(msg tea.Msg) (cmd tea.Cmd, handled bool)
	// HandleKey is offered raw key presses while the UI is in normal mode
	// (see inNormalMode). handled=true consumes the key.
	HandleKey(msg tea.KeyMsg) (cmd tea.Cmd, handled bool)
	// Render draws the main layout. Return false to use the default layout.
	Render(dl *render.DisplayContext, box layout.Box, core Core) bool
}

// Core exposes the stock models an Extension composes into its layout.
type Core struct {
	Revisions *revisions.Model
	Revset    *revset.Model
	Status    *status.Model
	// RenderMain draws the revisions list with its split content (preview,
	// bookmark pane) into box, exactly as the default layout does.
	RenderMain func(box layout.Box)
}

// WindowTitlePrefix is the leading part of the terminal window title.
var WindowTitlePrefix = "jjui"

// NewWithExtension is New with an Extension attached.
func NewWithExtension(c *context.MainContext, ext Extension) tea.Model {
	ui := NewUI(c)
	ui.ext = ext
	ext.Attach(ui.core())
	return &wrapper{ui: ui}
}

func (m *Model) core() Core {
	return Core{
		Revisions:  m.revisions,
		Revset:     m.revsetModel,
		Status:     m.status,
		RenderMain: func(box layout.Box) { m.renderSplit(m.revisions, box) },
	}
}

// inNormalMode reports whether a raw key can safely be offered to the
// Extension: nothing modal is open, no key sequence is pending, and no text
// input or revision operation is active.
func (m *Model) inNormalMode() bool {
	return m.stacked == nil && m.diff == nil && m.oplog == nil && m.password == nil &&
		len(m.sequenceHelp) == 0 &&
		!m.revsetModel.IsEditing() && !m.status.IsFocused() &&
		!m.splitContainer.ContentFocused() &&
		m.revisions.InNormalMode() && !m.revisions.HasQuickSearch()
}
```

c. Replace `Init`:

```go
func (m *Model) Init() tea.Cmd {
	cmds := []tea.Cmd{m.revisions.Init(), m.scheduleAutoRefresh()}
	if m.ext != nil {
		cmds = append(cmds, m.ext.Init())
	}
	return tea.Batch(cmds...)
}
```

d. In `Update`, directly after `var cmds []tea.Cmd` (line 154) and before the `handleSplitMsg` call, insert:

```go
	if m.ext != nil {
		if cmd, handled := m.ext.Update(msg); handled {
			return m.withSelectionSync(cmd)
		} else if cmd != nil {
			cmds = append(cmds, cmd)
		}
		if key, ok := msg.(tea.KeyMsg); ok && m.inNormalMode() {
			if cmd, handled := m.ext.HandleKey(key); handled {
				return m.withSelectionSync(cmd)
			}
		}
	}
```

e. In `View`, replace the inner `if m.oplog != nil { ... } else { m.renderRevisionsLayout(box) }` with:

```go
		if m.oplog != nil {
			m.renderOpLogLayout(box)
		} else if m.ext == nil || !m.ext.Render(m.displayContext, box, m.core()) {
			m.renderRevisionsLayout(box)
		}
```

f. In `wrapper.View`, replace `v.WindowTitle = fmt.Sprintf("jjui - %s", title)` with `v.WindowTitle = fmt.Sprintf("%s - %s", WindowTitlePrefix, title)`.

Build: `go build ./... && go test ./internal/ui/...` → green; `jjui` is unaffected because `ext` is nil there.

- [ ] **Step 6: Write `shell.go`**

```go
package shell

import (
	"fmt"
	"path/filepath"

	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"

	"github.com/idursun/jjui/internal/jjj"
	"github.com/idursun/jjui/internal/ui"
	"github.com/idursun/jjui/internal/ui/common"
	"github.com/idursun/jjui/internal/ui/context"
	"github.com/idursun/jjui/internal/ui/layout"
	"github.com/idursun/jjui/internal/ui/preview"
	"github.com/idursun/jjui/internal/ui/render"
	"github.com/idursun/jjui/internal/ui/revisions"
)

// focusPanelMsg is produced by clicking a panel's rectangle.
type focusPanelMsg struct{ id PanelID }

var (
	borderFocused   = lipgloss.NewStyle().Foreground(lipgloss.Color("2"))
	borderUnfocused = lipgloss.NewStyle().Foreground(lipgloss.Color("8"))
	focusKeys       = map[string]PanelID{"0": PanelDetail, "1": PanelStatus, "4": PanelJournal}
)

// Shell is the ui.Extension that lays out panels 1 (Status), 4 (Journal)
// and 0 (Detail) and routes keys by focused panel.
type Shell struct {
	ctx     *context.MainContext
	cfg     *jjj.Config
	detail  *preview.Model
	focus   PanelID
	core    ui.Core
	hasCore bool
}

func New(ctx *context.MainContext, cfg *jjj.Config) *Shell {
	return &Shell{ctx: ctx, cfg: cfg, detail: preview.New(ctx), focus: PanelJournal}
}

// Attach receives the stock models before Init; Task 5 installs the journal
// row-source factory here.
func (s *Shell) Attach(core ui.Core) { s.core, s.hasCore = core, true }

func (s *Shell) Init() tea.Cmd { return s.detail.Init() }

// Update forwards every non-input message to our panels (the Detail preview
// listens for SelectionChangedMsg/RefreshMsg and its own content message) and
// handles panel clicks.
func (s *Shell) Update(msg tea.Msg) (tea.Cmd, bool) {
	switch msg := msg.(type) {
	case focusPanelMsg:
		s.setFocus(msg.id)
		return nil, true
	case revisions.PaneClickedMsg, revisions.ItemClickedMsg:
		s.setFocus(PanelJournal) // let revisions handle the click too
		return nil, false
	}
	if common.IsInputMessage(msg) {
		return nil, false
	}
	return s.detail.Update(msg), false
}

// HandleKey: digits and tab move focus; when Detail is focused the movement
// keys scroll it. Everything else falls through to upstream.
func (s *Shell) HandleKey(msg tea.KeyMsg) (tea.Cmd, bool) {
	press, ok := msg.(tea.KeyPressMsg)
	if !ok {
		return nil, false
	}
	key := press.String()
	if id, ok := focusKeys[key]; ok {
		s.setFocus(id)
		return nil, true
	}
	switch key {
	case "tab":
		s.setFocus(s.focus.next())
		return nil, true
	case "shift+tab":
		s.setFocus(s.focus.prev())
		return nil, true
	}
	if s.focus == PanelDetail {
		switch key {
		case "j", "down":
			return s.detail.Scroll(1), true
		case "k", "up":
			return s.detail.Scroll(-1), true
		case "ctrl+d":
			return s.detail.HalfPageDown(), true
		case "ctrl+u":
			return s.detail.HalfPageUp(), true
		case "pgdown":
			return s.detail.PageDown(), true
		case "pgup":
			return s.detail.PageUp(), true
		}
	}
	return nil, false
}

func (s *Shell) setFocus(id PanelID) {
	s.focus = id
	if s.hasCore {
		s.core.Revisions.SetFocused(id == PanelJournal)
	}
}

// Render draws the panels. It returns false (stock layout) when the shell is
// disabled or the terminal is narrower than [shell] narrow_columns.
func (s *Shell) Render(dl *render.DisplayContext, box layout.Box, core ui.Core) bool {
	s.core, s.hasCore = core, true
	if !s.cfg.Shell.Enabled || box.R.Dx() < s.cfg.Shell.NarrowColumns {
		return false
	}
	content, statusLine := box.CutBottom(1)
	lay := Compute(content, s.cfg.Shell.LeftWidth)

	s.panel(dl, lay.Status, PanelStatus, "Status", func(inner layout.Box) {
		s.drawStatusPlaceholder(dl, inner)
	})
	s.panel(dl, lay.Journal, PanelJournal, "Journal", func(inner layout.Box) {
		rows := inner.V(layout.Fixed(1), layout.Fill(1))
		if len(rows) < 2 {
			return
		}
		core.Revset.ViewRect(dl, rows[0])
		core.RenderMain(rows[1])
	})
	s.panel(dl, lay.Detail, PanelDetail, "Detail", func(inner layout.Box) {
		s.detail.ViewRect(dl, inner)
	})
	core.Status.ViewRect(dl, statusLine)
	return true
}

// panel draws a rounded border with a "[n]─Title" tab, registers a click
// target that focuses the panel, then draws the content inside the border.
func (s *Shell) panel(dl *render.DisplayContext, box layout.Box, id PanelID, title string, draw func(inner layout.Box)) {
	w, h := box.R.Dx(), box.R.Dy()
	if w < 4 || h < 3 {
		return
	}
	style := borderUnfocused
	if s.focus == id {
		style = borderFocused
	}
	frame := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(style.GetForeground()).
		Width(w - 2).
		Height(h - 2).
		Render("")
	dl.AddDraw(box.R, frame, render.ZBase)
	dl.AddInteraction(box.R, focusPanelMsg{id: id}, render.InteractionClick, -1)
	dl.Text(box.R.Min.X+2, box.R.Min.Y, render.ZBase).
		Styled(fmt.Sprintf("[%d]─%s", int(id), title), style.Bold(s.focus == id)).
		Done()
	draw(box.Inset(1))
}

// drawStatusPlaceholder is replaced by the statuspanel model in Task 6.
func (s *Shell) drawStatusPlaceholder(dl *render.DisplayContext, inner layout.Box) {
	dl.Text(inner.R.Min.X, inner.R.Min.Y, render.ZBase).
		Write(filepath.Base(s.ctx.Location)).
		NewLine().
		Write(s.ctx.CurrentRevset).
		Done()
}
```

Notes for the implementer:
- `render.InteractionClick` and the `-1` z-index mirror how `revisions.ViewRect` registers its own pane click (`revisions.go:1225`), so row clicks inside Journal still reach the list.
- `preview.Model.Scroll/PageDown/...` return `tea.Cmd` (`preview.go:104-140`); returning them is what makes the wrapper re-render.
- `lipgloss.Style.GetForeground()` returns a `color.Color`, which is what `BorderForeground` takes in Lipgloss v2.

- [ ] **Step 7: Boot the shell from `cmd/jjj/main.go`**

Replace the two lines at the `// jjj: extension` marker with:

```go
	// jjj: extension
	ui.WindowTitlePrefix = "jjj"
	var model tea.Model
	if jjj.Current.Shell.Enabled {
		model = ui.NewWithExtension(appContext, shell.New(appContext, jjj.Current))
	} else {
		model = ui.New(appContext)
	}
	p := tea.NewProgram(model, tea.WithInput(os.Stdin))
```

Add the import `"github.com/idursun/jjui/internal/ui/shell"`.

- [ ] **Step 8: Smoke test the three panels and the narrow fallback**

```bash
make build
rm -rf /tmp/jjjcfg && mkdir -p /tmp/jjjcfg
JJJ_CONFIG_DIR=/tmp/jjjcfg python3 scripts/pty-smoke.py bin/jjj ~/Developer/Home/ticktock --wait 12 \
  --expect "[1]─Status" --expect "[4]─Journal" --expect "[0]─Detail" --expect "revset: all()" --expect "Commit ID"
JJJ_CONFIG_DIR=/tmp/jjjcfg python3 scripts/pty-smoke.py bin/jjj ~/Developer/Home/ticktock --wait 8 --cols 100 --expect "revset: all()"
make check
```

Expected: first run `ok: ...` (Detail shows the `jj show --stat --git` header for the selected working copy); second run `ok` (narrow terminal falls back to the stock layout without crashing); `make check` green including `go build -o /dev/null ./cmd/jjui`.

Then run it interactively once: `bin/jjj ~/Developer/Home/ticktock`. Press `0`, `j`, `k` (Detail scrolls; its border turns green), `4` (Journal focused again), `j`/`k` (rows move, Detail follows), `d` (describe opens as in jjui), `esc`, `q`. If a digit or `tab` is swallowed while a modal is open, `inNormalMode` is missing a case — fix there, not in the shell.

- [ ] **Step 9: Record the seam and commit**

Add to the `UPSTREAM.md` table:

```
| `internal/ui/ui.go` | `Extension` hook (`Extension`, `Core`, `NewWithExtension`, `inNormalMode`, `WindowTitlePrefix`) and its three call sites in `Init`/`Update`/`View` |
```

```bash
jj commit -m "feat(jjj): Extension hook in ui.Model and the three-panel shell

ui.Model gains an optional Extension that sees messages first, gets raw
keys in normal mode, and may draw the main layout. internal/ui/shell uses
it for Status / Journal / Detail panels with digit and tab focus."
jj bookmark set main -r @-
```

---

### Task 4: `RowSource` seam, `jj.LogTemplate`, and the pure journal logic (timestamps, arrange)

**Files:**
- Modify: `internal/jj/commands.go:44-63` (`Log`)
- Modify: `internal/ui/revisions/revisions.go` (`Model` :56-77, `streamingReadyMsg` :120-125, `loadStreaming` :1297-1321, new declarations)
- Create: `internal/ui/revisions/rowsource_test.go`
- Create: `internal/journal/timestamps.go`, `internal/journal/timestamps_test.go`, `internal/journal/arrange.go`, `internal/journal/arrange_test.go`
- Modify: `UPSTREAM.md`

**Interfaces:**
- Produces (package `jj`): `func LogTemplate(rowTemplate string) string` — the marker prefix `stringify('_PREFIX:' ++ separate('_PREFIX:', change_id.shortest() ++ if(divergent, "/" ++ change_offset), commit_id.shortest())) ++ ' ' ++ rowTemplate`.
- Produces (package `revisions`):

```go
type RowSource interface {
	RequestMore() parser.RowBatch
	Close()
}
type RowSourceFactory func(ctx context.Context, runner appContext.CommandRunner, revset string, jjTemplate string) (RowSource, error)
func DefaultRowSource(ctx context.Context, runner appContext.CommandRunner, revset string, jjTemplate string) (RowSource, error)
func (m *Model) SetRowSourceFactory(f RowSourceFactory)
```

- Produces (package `journal`):

```go
type Stamp struct{ Time time.Time; WorkingCopy bool }
const TimestampTemplate string
func TimestampArgs(revset string, limit int) []string
func ParseTimestamps(output []byte) (stamps map[string]Stamp, skipped int)
type Options struct{ Now time.Time; DayStyle, WeekStyle lipgloss.Style }
type Arranged struct{ Rows []parser.Row; HeaderIndices []int }
func Arrange(rows []parser.Row, stamps map[string]Stamp, opts Options) Arranged
func JumpTarget(headers []int, cursor int, forward bool) int
```

- [ ] **Step 1: Extract `jj.LogTemplate`**

In `internal/jj/commands.go` replace the body of `Log` (lines 44-63) with:

```go
// LogTemplate returns the template jjui parses rows from: the change/commit
// id marker prefix followed by rowTemplate. Exported so other row sources
// (jjj's journal mode) produce output the same parser understands.
func LogTemplate(rowTemplate string) string {
	prefix := fmt.Sprintf(
		"stringify('%s' ++ separate('%s', change_id.shortest() ++ if(divergent, \"/\" ++ change_offset), commit_id.shortest()))",
		JJUIPrefix, JJUIPrefix)
	return fmt.Sprintf("%s ++ ' ' ++ %s", prefix, rowTemplate)
}

func Log(revset string, limit int, jjTemplate string) CommandArgs {
	args := []string{"log", "--color", "always", "--quiet"}
	if revset != "" {
		args = append(args, "-r", revset)
	}
	if limit > 0 {
		args = append(args, "--limit", strconv.Itoa(limit))
	}
	template := config.Current.Revisions.Template
	// If jjui's template is empty, fall back to jj's templates.log
	if template == "" {
		template = jjTemplate
	}
	args = append(args, "-T", LogTemplate(template))
	return args
}
```

Run: `go test ./internal/jj/ ./internal/parser/` → PASS (pure extraction; upstream tests cover `Log`).

- [ ] **Step 2: Write the failing row-source tests**

Create `internal/ui/revisions/rowsource_test.go`:

```go
package revisions

import (
	"context"
	"errors"
	"testing"

	"github.com/idursun/jjui/internal/config"
	"github.com/idursun/jjui/internal/parser"
	appContext "github.com/idursun/jjui/internal/ui/context"
	"github.com/stretchr/testify/require"
)

type fakeSource struct{ closed bool }

func (f *fakeSource) RequestMore() parser.RowBatch { return parser.RowBatch{} }
func (f *fakeSource) Close()                       { f.closed = true }

// failingRunner only implements the method DefaultRowSource reaches.
type failingRunner struct{ appContext.CommandRunner }

func (failingRunner) RunCommandStreaming(context.Context, []string) (*appContext.StreamingCommand, error) {
	return nil, errors.New("boom")
}

func TestLoadStreamingUsesConfiguredRowSourceFactory(t *testing.T) {
	m := New(&appContext.MainContext{JJConfig: &config.JJConfig{}})
	src := &fakeSource{}
	var gotRevset, gotTemplate string
	m.SetRowSourceFactory(func(_ context.Context, _ appContext.CommandRunner, revset, jjTemplate string) (RowSource, error) {
		gotRevset, gotTemplate = revset, jjTemplate
		return src, nil
	})
	tag := m.tag.Add(1)

	msg := m.loadStreaming("all()", tag)()

	ready, ok := msg.(streamingReadyMsg)
	require.True(t, ok, "expected streamingReadyMsg, got %T", msg)
	require.Same(t, src, ready.streamer)
	require.NoError(t, ready.err)
	require.Equal(t, "all()", gotRevset)
	require.Equal(t, "", gotTemplate, "jj's templates.log is empty in this context")
}

func TestLoadStreamingReportsFactoryFailure(t *testing.T) {
	m := New(&appContext.MainContext{JJConfig: &config.JJConfig{}})
	m.SetRowSourceFactory(func(context.Context, appContext.CommandRunner, string, string) (RowSource, error) {
		return nil, errors.New("no rows for you")
	})
	msg := m.loadStreaming("all()", m.tag.Add(1))()
	ready := msg.(streamingReadyMsg)
	require.Nil(t, ready.streamer)
	require.EqualError(t, ready.err, "no rows for you")
	require.Equal(t, "no rows for you", ready.output)
}

func TestDefaultRowSourceReturnsUntypedNilOnFailure(t *testing.T) {
	src, err := DefaultRowSource(context.Background(), failingRunner{}, "all()", "")
	require.Error(t, err)
	require.True(t, src == nil, "a failed source must be an untyped nil, not a nil *GraphStreamer in an interface")
}
```

- [ ] **Step 3: Run to verify they fail**

Run: `go test ./internal/ui/revisions/ -run 'RowSource|LoadStreaming'`
Expected: compile error `undefined: RowSource`.

- [ ] **Step 4: Add the seam to `revisions.go`**

a. Change the two field types and add the factory field:

- In `Model`: `streamer *graph.GraphStreamer` → `streamer RowSource`; add `rowSourceFactory RowSourceFactory` directly below it.
- In `streamingReadyMsg`: `streamer *graph.GraphStreamer` → `streamer RowSource`.

b. After the `appendRowsBatchMsg` type (line 131), add:

```go
// RowSource feeds parsed `jj log` rows to the list in batches.
// *graph.GraphStreamer is the stock implementation; jjj's journal mode
// supplies another through SetRowSourceFactory.
type RowSource interface {
	RequestMore() parser.RowBatch
	Close()
}

// RowSourceFactory builds the RowSource for one (re)load of revset.
// jjTemplate is jj's own templates.log, used when no template is configured.
type RowSourceFactory func(ctx context.Context, runner appContext.CommandRunner, revset string, jjTemplate string) (RowSource, error)

// DefaultRowSource is the stock factory: a streaming `jj log` graph.
func DefaultRowSource(ctx context.Context, runner appContext.CommandRunner, revset string, jjTemplate string) (RowSource, error) {
	streamer, err := graph.NewGraphStreamer(ctx, runner, revset, jjTemplate)
	if streamer == nil {
		// Return an untyped nil: a nil *GraphStreamer stored in the interface
		// would not compare equal to nil and the failure path would crash.
		return nil, err
	}
	return streamer, err
}

// SetRowSourceFactory replaces the stock factory; nil restores the default.
func (m *Model) SetRowSourceFactory(f RowSourceFactory) {
	m.rowSourceFactory = f
}
```

c. In `loadStreaming`, replace

```go
		streamer, err := graph.NewGraphStreamer(context.Background(), m.context, revset, m.context.JJConfig.Templates.Log)
```

with

```go
		factory := m.rowSourceFactory
		if factory == nil {
			factory = DefaultRowSource
		}
		streamer, err := factory(context.Background(), m.context, revset, m.context.JJConfig.Templates.Log)
```

Nothing else changes: `msg.streamer.Close()`, `m.streamer.Close()` and `streamer.RequestMore()` already go through the interface.

- [ ] **Step 5: Run the seam tests and the whole package**

Run: `go test ./internal/ui/revisions/ && go build ./cmd/jjui`
Expected: PASS, and jjui still builds.

- [ ] **Step 6: Write the failing timestamp tests**

Create `internal/journal/timestamps_test.go`:

```go
package journal

import (
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func TestParseTimestamps(t *testing.T) {
	out := []byte("0123abcd\t2026-09-04T16:30:00-0700\t@\n" +
		"deadbeef\t2026-07-13T08:55:03-0700\t-\n" +
		"garbage line without a tab\n" +
		"cafe\tnot-a-time\t-\n")
	stamps, skipped := ParseTimestamps(out)

	require.Equal(t, 2, skipped)
	require.Len(t, stamps, 2)
	require.True(t, stamps["0123abcd"].WorkingCopy)
	require.False(t, stamps["deadbeef"].WorkingCopy)
	want := time.Date(2026, 7, 13, 8, 55, 3, 0, time.FixedZone("", -7*3600))
	require.True(t, stamps["deadbeef"].Time.Equal(want), "got %v", stamps["deadbeef"].Time)
}

func TestTimestampArgs(t *testing.T) {
	args := TimestampArgs("all()", 5000)
	require.Equal(t, []string{"log", "--no-graph", "--color", "never", "--quiet", "--ignore-working-copy",
		"-r", "all()", "-T", TimestampTemplate, "--limit", "5000"}, args)
	require.NotContains(t, TimestampArgs("all()", 0), "--limit")
}
```

- [ ] **Step 7: Write `timestamps.go`**

```go
package journal

import (
	"strconv"
	"strings"
	"time"
)

// Stamp is what the journal needs to know about a commit beyond its row:
// when it was committed (local time) and whether it is the working copy.
type Stamp struct {
	Time        time.Time
	WorkingCopy bool
}

// TimestampTemplate keys each commit by the same shortest commit id the log
// marker carries (jj.LogTemplate), so rows and stamps join on Commit.CommitId.
// Fields: id, local committer time, "@" for the working copy.
const TimestampTemplate = `commit_id.shortest() ++ "\t" ++ committer.timestamp().local().format("%Y-%m-%dT%H:%M:%S%z") ++ "\t" ++ if(current_working_copy, "@", "-") ++ "\n"`

const timestampLayout = "2006-01-02T15:04:05-0700"

// TimestampArgs is the `jj log` invocation producing TimestampTemplate output.
// It runs after the row query, which already snapshotted the working copy,
// so --ignore-working-copy is safe and faster.
func TimestampArgs(revset string, limit int) []string {
	args := []string{"log", "--no-graph", "--color", "never", "--quiet", "--ignore-working-copy",
		"-r", revset, "-T", TimestampTemplate}
	if limit > 0 {
		args = append(args, "--limit", strconv.Itoa(limit))
	}
	return args
}

// ParseTimestamps parses TimestampTemplate output. Malformed lines are
// skipped and counted so a partial failure never hides rows.
func ParseTimestamps(output []byte) (map[string]Stamp, int) {
	stamps := make(map[string]Stamp)
	skipped := 0
	for _, line := range strings.Split(strings.TrimSpace(string(output)), "\n") {
		if line == "" {
			continue
		}
		fields := strings.Split(line, "\t")
		if len(fields) < 2 {
			skipped++
			continue
		}
		when, err := time.Parse(timestampLayout, strings.TrimSpace(fields[1]))
		if err != nil {
			skipped++
			continue
		}
		stamp := Stamp{Time: when}
		if len(fields) >= 3 && strings.TrimSpace(fields[2]) == "@" {
			stamp.WorkingCopy = true
		}
		stamps[strings.TrimSpace(fields[0])] = stamp
	}
	return stamps, skipped
}
```

Run: `go test ./internal/journal/ -run Timestamp` → PASS.

- [ ] **Step 8: Write the failing arrange tests**

Create `internal/journal/arrange_test.go`:

```go
package journal

import (
	"testing"
	"time"

	"github.com/idursun/jjui/internal/parser"
	"github.com/idursun/jjui/internal/screen"
	"github.com/stretchr/testify/require"
)

// row builds a parsed row the way the parser does for `--no-graph` output:
// one Revision|Highlightable line and an empty gutter.
func row(commitID, subject string) parser.Row {
	r := parser.NewGraphRow()
	r.Commit.CommitId = commitID
	r.Commit.ChangeId = "c" + commitID
	line := parser.NewGraphRowLine([]*screen.Segment{{Text: subject}})
	r.AddLine(&line)
	return r
}

func lineText(l *parser.GraphRowLine) string {
	s := ""
	for _, seg := range l.Segments {
		s += seg.Text
	}
	return s
}

var pdt = time.FixedZone("PDT", -7*3600)

func at(y, mo, d, h, mi int) time.Time { return time.Date(y, mo, d, h, mi, 0, 0, pdt) }

func TestArrangeSortsNewestFirstAndInsertsHeaders(t *testing.T) {
	now := at(2026, 9, 4, 17, 0) // Friday, ISO week 36
	rows := []parser.Row{
		row("a1", "older, same day"),
		row("b2", "newest"),
		row("c3", "yesterday"),
		row("d4", "last week"),
	}
	stamps := map[string]Stamp{
		"a1": {Time: at(2026, 9, 4, 10, 11)},
		"b2": {Time: at(2026, 9, 4, 16, 30)},
		"c3": {Time: at(2026, 9, 3, 9, 0)},
		"d4": {Time: at(2026, 8, 28, 12, 0)}, // Friday of ISO week 35
	}

	got := Arrange(rows, stamps, Options{Now: now})

	ids := func() []string {
		var out []string
		for _, r := range got.Rows {
			out = append(out, r.Commit.CommitId)
		}
		return out
	}()
	require.Equal(t, []string{"b2", "a1", "c3", "d4"}, ids)
	require.Equal(t, []int{0, 2, 3}, got.HeaderIndices)

	// first row: week header, day header, then the revision line
	first := got.Rows[0].Lines
	require.Len(t, first, 3)
	require.Equal(t, "══ Week 36 · Aug 31 – Sep 6", lineText(first[0]))
	require.Equal(t, "── Today · Fri 2026-09-04 · 2 changes", lineText(first[1]))
	require.Equal(t, "newest", lineText(first[2]))
	require.Equal(t, parser.RowLineFlags(0), first[0].Flags)
	require.Equal(t, parser.RowLineFlags(0), first[1].Flags)
	require.Equal(t, parser.Revision|parser.Highlightable, first[2].Flags)

	require.Len(t, got.Rows[1].Lines, 1, "same day: no header")
	require.Equal(t, "── Yesterday · Thu 2026-09-03 · 1 change", lineText(got.Rows[2].Lines[0]))

	lastWeek := got.Rows[3].Lines
	require.Len(t, lastWeek, 3)
	require.Equal(t, "══ Week 35 · Aug 24 – Aug 30", lineText(lastWeek[0]))
	require.Equal(t, "── Fri 2026-08-28 · 1 change", lineText(lastWeek[1]))

	for _, r := range got.Rows {
		require.Nil(t, r.Previous, "journal rows carry no graph continuity")
	}
}

func TestArrangePinsWorkingCopyFirstAndMarksIt(t *testing.T) {
	rows := []parser.Row{row("new1", "a fresh commit"), row("wc00", "(working copy)")}
	stamps := map[string]Stamp{
		"new1": {Time: at(2026, 9, 4, 12, 0)},
		"wc00": {Time: at(2026, 9, 1, 8, 0), WorkingCopy: true}, // older, still first
	}
	got := Arrange(rows, stamps, Options{Now: at(2026, 9, 4, 17, 0)})
	require.Equal(t, "wc00", got.Rows[0].Commit.CommitId)
	require.True(t, got.Rows[0].Commit.IsWorkingCopy)
	require.False(t, got.Rows[1].Commit.IsWorkingCopy)
}

func TestArrangeKeepsUnstampedRowsLastInOriginalOrder(t *testing.T) {
	rows := []parser.Row{row("x", "no stamp 1"), row("y", "stamped"), row("z", "no stamp 2")}
	stamps := map[string]Stamp{"y": {Time: at(2026, 9, 4, 12, 0)}}
	got := Arrange(rows, stamps, Options{Now: at(2026, 9, 4, 17, 0)})
	require.Equal(t, "y", got.Rows[0].Commit.CommitId)
	require.Equal(t, "x", got.Rows[1].Commit.CommitId)
	require.Equal(t, "z", got.Rows[2].Commit.CommitId)
	require.Len(t, got.Rows[1].Lines, 1, "unstamped rows get no header")
}

func TestJumpTarget(t *testing.T) {
	headers := []int{0, 4, 9}
	require.Equal(t, 4, JumpTarget(headers, 0, true))
	require.Equal(t, 9, JumpTarget(headers, 4, true))
	require.Equal(t, -1, JumpTarget(headers, 9, true))
	require.Equal(t, 4, JumpTarget(headers, 6, false))
	require.Equal(t, 0, JumpTarget(headers, 4, false))
	require.Equal(t, -1, JumpTarget(headers, 0, false))
	require.Equal(t, -1, JumpTarget(nil, 3, true))
}
```

- [ ] **Step 9: Run to verify they fail**

Run: `go test ./internal/journal/ -run 'Arrange|JumpTarget'`
Expected: compile error `undefined: Arrange`.

- [ ] **Step 10: Write `arrange.go`**

```go
package journal

import (
	"fmt"
	"sort"
	"time"

	"charm.land/lipgloss/v2"

	"github.com/idursun/jjui/internal/parser"
	"github.com/idursun/jjui/internal/screen"
)

// Options controls Arrange. Now anchors "Today"/"Yesterday" and the local
// zone; the styles are applied to header lines (zero values in tests).
type Options struct {
	Now       time.Time
	DayStyle  lipgloss.Style
	WeekStyle lipgloss.Style
}

// Arranged is Arrange's result: rows in display order plus the indices of the
// rows that carry a day header (the jump targets for `{` / `}`).
type Arranged struct {
	Rows          []parser.Row
	HeaderIndices []int
}

const dayKey = "2006-01-02"

// Arrange orders rows newest-first by committer time, working copy first,
// and prepends header lines: a day header on the first row of each calendar
// day and, above it, a week header on the first row of each ISO week. Rows
// with no stamp keep their relative order after the stamped rows and get no
// header. Header lines have no flags, so they are never highlighted or
// treated as the revision line.
func Arrange(rows []parser.Row, stamps map[string]Stamp, opts Options) Arranged {
	if opts.Now.IsZero() {
		opts.Now = time.Now()
	}
	loc := opts.Now.Location()

	sorted := make([]parser.Row, len(rows))
	copy(sorted, rows)
	for i := range sorted {
		sorted[i].Previous = nil
		if s, ok := stamps[sorted[i].Commit.CommitId]; ok && s.WorkingCopy {
			sorted[i].Commit.IsWorkingCopy = true
		}
	}
	stampOf := func(r parser.Row) (time.Time, bool) {
		s, ok := stamps[r.Commit.CommitId]
		return s.Time.In(loc), ok
	}
	sort.SliceStable(sorted, func(i, j int) bool {
		a, b := sorted[i], sorted[j]
		if a.Commit.IsWorkingCopy != b.Commit.IsWorkingCopy {
			return a.Commit.IsWorkingCopy
		}
		at, aok := stampOf(a)
		bt, bok := stampOf(b)
		if aok != bok {
			return aok
		}
		if !aok {
			return false
		}
		return at.After(bt)
	})

	counts := map[string]int{}
	for _, r := range sorted {
		if t, ok := stampOf(r); ok {
			counts[t.Format(dayKey)]++
		}
	}

	out := Arranged{Rows: sorted}
	prevDay := ""
	prevYear, prevWeek := 0, 0
	for i := range sorted {
		t, ok := stampOf(sorted[i])
		if !ok {
			continue
		}
		day := t.Format(dayKey)
		if day == prevDay {
			continue
		}
		prependLine(&sorted[i], dayHeader(t, opts.Now, counts[day]), opts.DayStyle)
		year, week := t.ISOWeek()
		if year != prevYear || week != prevWeek {
			prependLine(&sorted[i], weekHeader(t, week), opts.WeekStyle)
		}
		out.HeaderIndices = append(out.HeaderIndices, i)
		prevDay, prevYear, prevWeek = day, year, week
	}
	return out
}

func dayHeader(t, now time.Time, count int) string {
	label := t.Format("Mon 2006-01-02")
	switch t.Format(dayKey) {
	case now.Format(dayKey):
		label = "Today · " + label
	case now.AddDate(0, 0, -1).Format(dayKey):
		label = "Yesterday · " + label
	}
	noun := "changes"
	if count == 1 {
		noun = "change"
	}
	return fmt.Sprintf("── %s · %d %s", label, count, noun)
}

func weekHeader(t time.Time, week int) string {
	monday := t.AddDate(0, 0, -((int(t.Weekday()) + 6) % 7))
	sunday := monday.AddDate(0, 0, 6)
	return fmt.Sprintf("══ Week %d · %s – %s", week, monday.Format("Jan 2"), sunday.Format("Jan 2"))
}

func prependLine(row *parser.Row, text string, style lipgloss.Style) {
	line := parser.NewGraphRowLine([]*screen.Segment{{Text: text, Style: style}})
	line.Flags = 0
	row.Lines = append([]*parser.GraphRowLine{&line}, row.Lines...)
}

// JumpTarget returns the header index after (forward) or before cursor, or -1.
func JumpTarget(headers []int, cursor int, forward bool) int {
	if forward {
		for _, h := range headers {
			if h > cursor {
				return h
			}
		}
		return -1
	}
	for i := len(headers) - 1; i >= 0; i-- {
		if headers[i] < cursor {
			return headers[i]
		}
	}
	return -1
}
```

- [ ] **Step 11: Run all journal tests**

Run: `go test ./internal/journal/ -v`
Expected: every test PASS. If the week test fails on "Aug 31 – Sep 6", check `weekHeader`'s Monday arithmetic (Go's `Weekday()` has Sunday = 0; the `+6 %7` maps Monday to 0).

- [ ] **Step 12: Record seams and commit**

Add to the `UPSTREAM.md` table:

```
| `internal/jj/commands.go` | `LogTemplate` extracted from `Log` (pure extraction) so other row sources emit parser-compatible output |
| `internal/ui/revisions/revisions.go` | `RowSource` interface, `RowSourceFactory`, `DefaultRowSource`, `SetRowSourceFactory`; `streamer` fields typed as the interface |
```

```bash
make check
jj commit -m "feat(jjj): RowSource seam in revisions and the pure journal logic

revisions.Model loads rows through a pluggable RowSourceFactory (stock:
the graph streamer). internal/journal parses committer stamps, sorts rows
newest-first with the working copy pinned, and prepends day/week header
lines. jj.LogTemplate is extracted so the journal source shares the marker."
jj bookmark set main -r @-
```

---

### Task 5: Journal `Source`, journal mode by default, `J` toggle, day-jump keys (P0 spike, part 2 + P1 core)

**Files:**
- Create: `internal/journal/source.go`, `internal/journal/source_test.go`
- Modify: `internal/ui/shell/shell.go` (fields, `New`, `Attach`, `HandleKey`, new `rowSource` method)

**Interfaces:**
- Produces (package `journal`):

```go
func LogArgs(revset string, limit int, rowTemplate string) []string
type Source struct{ /* unexported */ }
func NewSource(runner appContext.CommandRunner, revset, rowTemplate string, limit int, opts Options) (*Source, error) // io.EOF when the revset is empty
func (s *Source) RequestMore() parser.RowBatch   // whole revset once, HasMore=false, then empty
func (s *Source) Close()
func (s *Source) HeaderIndices() []int
```

- Consumes: `revisions.RowSource`, `revisions.DefaultRowSource`, `(*revisions.Model).SetRowSourceFactory`, `Cursor()`, `SetCursor(int)` (Task 4 / upstream); `jjj.Config.Journal.{Enabled, Template, Limit}` (Task 2); `common.RefreshAndKeepSelections` (upstream `msgs.go:159`).

- [ ] **Step 1: Write the failing source tests**

Create `internal/journal/source_test.go`:

```go
package journal

import (
	"context"
	"errors"
	"io"
	"strings"
	"testing"
	"time"

	"github.com/idursun/jjui/internal/jj"
	appContext "github.com/idursun/jjui/internal/ui/context"
	"github.com/stretchr/testify/require"
)

// cannedRunner answers the two queries NewSource makes: the row query (its
// template carries the jjui marker) and the timestamp query.
type cannedRunner struct {
	appContext.CommandRunner
	rows, stamps string
	err          error
	calls        [][]string
}

func (c *cannedRunner) RunCommandImmediate(args []string) ([]byte, error) {
	c.calls = append(c.calls, args)
	if c.err != nil {
		return nil, c.err
	}
	for i, a := range args {
		if a == "-T" && strings.Contains(args[i+1], jj.JJUIPrefix) {
			return []byte(strings.TrimRight(c.rows, "\n")), nil // RunCommandImmediate trims newlines
		}
	}
	return []byte(c.stamps), nil
}

func marker(changeID, commitID string) string {
	return jj.JJUIPrefix + changeID + jj.JJUIPrefix + commitID + " "
}

func TestNewSourceArrangesRowsAndServesOnce(t *testing.T) {
	runner := &cannedRunner{
		rows: marker("wc", "0001") + "17:00 (working copy)\n" +
			marker("aa", "0002") + "10:11 older same day\n" +
			marker("bb", "0003") + "16:30 newest\n",
		stamps: "0001\t2026-09-04T17:00:00-0700\t@\n" +
			"0002\t2026-09-04T10:11:00-0700\t-\n" +
			"0003\t2026-09-04T16:30:00-0700\t-\n",
	}
	src, err := NewSource(runner, "all()", RowTemplateJournal, 5000, Options{Now: time.Date(2026, 9, 4, 17, 30, 0, 0, time.FixedZone("PDT", -7*3600))})
	require.NoError(t, err)

	batch := src.RequestMore()
	require.False(t, batch.HasMore)
	require.Len(t, batch.Rows, 3)
	require.Equal(t, "0001", batch.Rows[0].Commit.CommitId, "working copy first")
	require.True(t, batch.Rows[0].Commit.IsWorkingCopy)
	require.Equal(t, "0003", batch.Rows[1].Commit.CommitId)
	require.Equal(t, "0002", batch.Rows[2].Commit.CommitId)
	require.Equal(t, []int{0}, src.HeaderIndices())
	require.Contains(t, lineText(batch.Rows[0].Lines[1]), "Today · Fri 2026-09-04 · 3 changes")

	require.Empty(t, src.RequestMore().Rows, "second request is empty")
	src.Close()

	require.Len(t, runner.calls, 2)
	require.Contains(t, runner.calls[0], "--no-graph")
	require.NotContains(t, runner.calls[0], "--ignore-working-copy", "row query must snapshot the working copy")
	require.Contains(t, runner.calls[1], "--ignore-working-copy")
}

func TestNewSourceEmptyRevsetIsEOF(t *testing.T) {
	src, err := NewSource(&cannedRunner{rows: "", stamps: ""}, "none()", RowTemplateJournal, 0, Options{})
	require.Nil(t, src)
	require.ErrorIs(t, err, io.EOF)
}

func TestNewSourcePropagatesJJErrors(t *testing.T) {
	src, err := NewSource(&cannedRunner{err: errors.New("Error: Revision `nope` doesn't exist")}, "nope", RowTemplateJournal, 0, Options{})
	require.Nil(t, src)
	require.EqualError(t, err, "Error: Revision `nope` doesn't exist")
}

func TestLogArgs(t *testing.T) {
	args := LogArgs("all()", 5000, RowTemplateJournal)
	require.Equal(t, []string{"log", "--no-graph", "--color", "always", "--quiet", "-r", "all()", "-T", jj.LogTemplate(RowTemplateJournal), "--limit", "5000"}, args)
	_ = context.Background
}
```

- [ ] **Step 2: Run to verify it fails**

Run: `go test ./internal/journal/ -run 'Source|LogArgs'`
Expected: compile error `undefined: NewSource`.

- [ ] **Step 3: Write `source.go`**

```go
package journal

import (
	"bytes"
	"io"
	"strconv"

	"github.com/idursun/jjui/internal/jj"
	"github.com/idursun/jjui/internal/parser"
	appContext "github.com/idursun/jjui/internal/ui/context"
)

// LogArgs is `jj log --no-graph` for revset with jjui's row marker and
// rowTemplate. Unlike TimestampArgs it snapshots the working copy, exactly
// as the graph loader does.
func LogArgs(revset string, limit int, rowTemplate string) []string {
	args := []string{"log", "--no-graph", "--color", "always", "--quiet", "-r", revset, "-T", jj.LogTemplate(rowTemplate)}
	if limit > 0 {
		args = append(args, "--limit", strconv.Itoa(limit))
	}
	return args
}

// Source is a revisions.RowSource that serves the whole revset in one batch,
// sorted newest-first with day and week header lines (see Arrange).
type Source struct {
	batch         parser.RowBatch
	headerIndices []int
	served        bool
}

// NewSource runs the row query and the stamp query and arranges the rows.
// It returns io.EOF for an empty revset, matching graph.NewGraphStreamer,
// so the caller's "No revisions found" message keeps working.
func NewSource(runner appContext.CommandRunner, revset, rowTemplate string, limit int, opts Options) (*Source, error) {
	out, err := runner.RunCommandImmediate(LogArgs(revset, limit, rowTemplate))
	if err != nil {
		return nil, err
	}
	rows := parser.ParseRows(bytes.NewReader(out))
	if len(rows) == 0 {
		return nil, io.EOF
	}
	stampOut, err := runner.RunCommandImmediate(TimestampArgs(revset, limit))
	if err != nil {
		return nil, err
	}
	stamps, _ := ParseTimestamps(stampOut)
	arranged := Arrange(rows, stamps, opts)
	return &Source{
		batch:         parser.RowBatch{Rows: arranged.Rows, HasMore: false},
		headerIndices: arranged.HeaderIndices,
	}, nil
}

// RequestMore hands out the arranged rows once; later calls return an empty
// batch, which the list treats as "no more".
func (s *Source) RequestMore() parser.RowBatch {
	if s.served {
		return parser.RowBatch{}
	}
	s.served = true
	return s.batch
}

func (s *Source) Close() {}

// HeaderIndices are the row indices carrying a day header, newest first.
func (s *Source) HeaderIndices() []int { return s.headerIndices }
```

- [ ] **Step 4: Run the source tests**

Run: `go test ./internal/journal/`
Expected: PASS. If `TestNewSourceArrangesRowsAndServesOnce` sees zero rows, the canned marker text does not match what `parser.ParseRowPrefixes` expects (`row_line.go:27-52`): the segment must contain `_PREFIX:<change>_PREFIX:<commit>` followed by a space and more text on the same line.

- [ ] **Step 5: Wire journal mode into the shell**

In `internal/ui/shell/shell.go`:

a. Add imports `gocontext "context"`, `"sync/atomic"`, `"time"`, `"github.com/idursun/jjui/internal/journal"`.

b. Add to `Shell`:

```go
	journalMode atomic.Bool                    // read from the load goroutine
	source      atomic.Pointer[journal.Source] // last journal source, for jump keys
```

c. In `New`, after constructing the struct: `s.journalMode.Store(cfg.Journal.Enabled)` (and return `s`).

d. Replace `Attach` with:

```go
// Attach receives the stock models before Init and installs the journal
// row-source factory so the very first load already uses journal mode.
func (s *Shell) Attach(core ui.Core) {
	s.core, s.hasCore = core, true
	core.Revisions.SetRowSourceFactory(s.rowSource)
}
```

e. Add the factory method:

```go
// rowSource is the revisions.RowSourceFactory: the journal source in journal
// mode, upstream's graph streamer otherwise. It runs inside the list's load
// command (a goroutine), hence the atomics.
func (s *Shell) rowSource(ctx gocontext.Context, runner appContext.CommandRunner, revset, jjTemplate string) (revisions.RowSource, error) {
	if !s.journalMode.Load() {
		s.source.Store(nil)
		return revisions.DefaultRowSource(ctx, runner, revset, jjTemplate)
	}
	tmpl := s.cfg.Journal.Template
	if tmpl == "" {
		tmpl = journal.RowTemplateJournal
	}
	dim := common.DefaultPalette.Get("revisions", "", "dimmed", false)
	src, err := journal.NewSource(runner, revset, tmpl, s.cfg.Journal.Limit, journal.Options{
		Now:       time.Now(),
		DayStyle:  dim.Bold(true),
		WeekStyle: dim,
	})
	if err != nil {
		return nil, err // untyped nil, see revisions.DefaultRowSource
	}
	s.source.Store(src)
	return src, nil
}
```

(`appContext` is the import alias `appContext "github.com/idursun/jjui/internal/ui/context"`; the file already imports that package as `context` for `MainContext` — rename that import to `appContext` throughout the file.)

f. In `HandleKey`, before the `if s.focus == PanelDetail` block, add:

```go
	switch key {
	case "J":
		s.journalMode.Store(!s.journalMode.Load())
		return common.RefreshAndKeepSelections, true
	case "{", "}":
		src := s.source.Load()
		if src == nil || !s.hasCore || s.focus != PanelJournal {
			return nil, false
		}
		if idx := journal.JumpTarget(src.HeaderIndices(), s.core.Revisions.Cursor(), key == "}"); idx >= 0 {
			s.core.Revisions.SetCursor(idx)
		}
		return nil, true
	}
```

`common.RefreshAndKeepSelections` is a `func() tea.Msg`, which is a `tea.Cmd`; the list reloads through the factory and picks the mode from the atomic.

- [ ] **Step 6: Smoke test journal mode and the toggle**

```bash
make build
rm -rf /tmp/jjjcfg && mkdir -p /tmp/jjjcfg
JJJ_CONFIG_DIR=/tmp/jjjcfg python3 scripts/pty-smoke.py bin/jjj ~/Developer/Home/ticktock --wait 12 \
  --expect "══ Week" --expect "── Mon 2026-07-13" --expect "08:55 main chore(tockr)" --expect "[0]─Detail"
JJJ_CONFIG_DIR=/tmp/jjjcfg python3 scripts/pty-smoke.py bin/jjj ~/Developer/Home/ticktock --wait 12 --key J --after 6 \
  --expect "Mon 2026-07-13 08:55"
make check
```

Expected: run 1 `ok` — a week header, a day header, and a time-only row (`08:55 main chore(tockr): …`); run 2 `ok` — after `J` the rows carry the full date again (graph mode); `make check` green.

Interactive check in `bin/jjj ~/Developer/Home/ticktock`: `}` and `{` move between day headers; `j`/`k` skip nothing (headers belong to the row below them); `d` on a journal row opens describe for that row; `J` twice returns to journal mode with the same row selected. If the highlight covers a header line, the renderer is treating a non-`Highlightable` line as selectable — check `displaycontext_renderer.go:364-420` before touching `Arrange`.

- [ ] **Step 7: Commit**

```bash
jj commit -m "feat(jjj): journal mode — time-sorted rows with day/week headers, J toggle, {/} jumps

journal.Source runs jj log --no-graph plus a stamp query and serves the
arranged rows through the RowSource seam; the shell installs it on Attach
and flips to the stock graph streamer on J."
jj bookmark set main -r @-
```

---

### Task 6: Status panel (repo, working copy, bookmark ahead/behind, fetch age, mode + revset)

**Files:**
- Create: `internal/statusinfo/statusinfo.go`, `internal/statusinfo/statusinfo_test.go`
- Create: `internal/ui/statuspanel/statuspanel.go`
- Modify: `internal/ui/shell/shell.go` (`Shell` fields, `New`, `Init`, `Update`, `Render`; delete `drawStatusPlaceholder`)

**Interfaces:**
- Produces (package `statusinfo`):

```go
type Info struct {
	Repo, ChangeID, Bookmark, Remote string
	Ahead, Behind int
	HasRemote bool
	FetchedAt time.Time
	Revset, Mode string
}
func WorkingCopyArgs() []string
func ClosestBookmarkArgs() []string
func CountArgs(revset string) []string
func FirstLine(output []byte) string
func FirstBookmark(output []byte) string
func Count(output []byte) int
func FetchTime(repoRoot string) time.Time
func FormatAge(now, at time.Time) string
func (i Info) Lines(now time.Time) [3]string
func Gather(runner appContext.CommandRunner, repoRoot, remote, revset, mode string, now time.Time) Info
```

- Produces (package `statuspanel`): `func New(ctx *appContext.MainContext, mode func() string) *Model` with `Init() tea.Cmd`, `Update(tea.Msg) tea.Cmd`, `ViewRect(*render.DisplayContext, layout.Box)`.
- Consumes: `config.GetGitDefaultRemote(config.Current)` (upstream `config.go:365`), `common.RefreshMsg`, `common.UpdateRevisionsSuccessMsg` (upstream `msgs.go`).

- [ ] **Step 1: Write the failing statusinfo tests**

Create `internal/statusinfo/statusinfo_test.go`:

```go
package statusinfo

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	appContext "github.com/idursun/jjui/internal/ui/context"
	"github.com/stretchr/testify/require"
)

func TestParsers(t *testing.T) {
	require.Equal(t, "xmqwnktn", FirstLine([]byte("xmqwnktn\n")))
	require.Equal(t, "", FirstLine(nil))
	require.Equal(t, "main", FirstBookmark([]byte("main,release\n")))
	require.Equal(t, "", FirstBookmark([]byte("\n")))
	require.Equal(t, 3, Count([]byte("xxx")))
	require.Equal(t, 0, Count(nil))
}

func TestFormatAge(t *testing.T) {
	now := time.Date(2026, 9, 4, 17, 0, 0, 0, time.UTC)
	require.Equal(t, "never", FormatAge(now, time.Time{}))
	require.Equal(t, "just now", FormatAge(now, now.Add(-20*time.Second)))
	require.Equal(t, "5m ago", FormatAge(now, now.Add(-5*time.Minute)))
	require.Equal(t, "3h ago", FormatAge(now, now.Add(-3*time.Hour)))
	require.Equal(t, "2d ago", FormatAge(now, now.Add(-49*time.Hour)))
}

func TestLines(t *testing.T) {
	now := time.Date(2026, 9, 4, 17, 0, 0, 0, time.UTC)
	info := Info{Repo: "jjjournal", ChangeID: "rkvnppyv", Bookmark: "main", Remote: "origin",
		Ahead: 2, Behind: 0, HasRemote: true, FetchedAt: now.Add(-3 * time.Hour), Revset: "all()", Mode: "journal"}
	lines := info.Lines(now)
	require.Equal(t, "jjjournal  @ rkvnppyv", lines[0])
	require.Equal(t, "main ↑2 ↓0 origin · fetched 3h ago", lines[1])
	require.Equal(t, "journal · all()", lines[2])

	info.HasRemote = false
	require.Equal(t, "main · not on origin · fetched 3h ago", info.Lines(now)[1])
	info.Bookmark = ""
	require.Equal(t, "(no bookmark) · fetched 3h ago", info.Lines(now)[1])
}

func TestFetchTime(t *testing.T) {
	root := t.TempDir()
	require.True(t, FetchTime(root).IsZero())
	require.NoError(t, os.MkdirAll(filepath.Join(root, ".git"), 0o755))
	require.NoError(t, os.WriteFile(filepath.Join(root, ".git", "FETCH_HEAD"), []byte("x"), 0o644))
	require.False(t, FetchTime(root).IsZero())
}

// cannedRunner answers by the revset it is asked about.
type cannedRunner struct {
	appContext.CommandRunner
	byRevset map[string]string
}

func (c cannedRunner) RunCommandImmediate(args []string) ([]byte, error) {
	for i, a := range args {
		if a == "-r" {
			if out, ok := c.byRevset[args[i+1]]; ok {
				return []byte(out), nil
			}
			return nil, os.ErrNotExist
		}
	}
	return nil, os.ErrNotExist
}

func TestGather(t *testing.T) {
	runner := cannedRunner{byRevset: map[string]string{
		"@":                          "rkvnppyv",
		"heads(::@ & bookmarks())":   "main",
		"main@origin..main":          "xx",
		"main..main@origin":          "",
	}}
	info := Gather(runner, t.TempDir(), "origin", "all()", "journal", time.Now())
	require.Equal(t, "rkvnppyv", info.ChangeID)
	require.Equal(t, "main", info.Bookmark)
	require.True(t, info.HasRemote)
	require.Equal(t, 2, info.Ahead)
	require.Equal(t, 0, info.Behind)
	require.Equal(t, "journal", info.Mode)
	require.Equal(t, "all()", info.Revset)
	require.True(t, strings.HasPrefix(info.Repo, ""), "repo is the directory basename")

	noRemote := Gather(cannedRunner{byRevset: map[string]string{"@": "abc", "heads(::@ & bookmarks())": "feature"}}, t.TempDir(), "origin", "all()", "graph", time.Now())
	require.Equal(t, "feature", noRemote.Bookmark)
	require.False(t, noRemote.HasRemote, "count query failed → ahead/behind unknown")
}
```

- [ ] **Step 2: Run to verify it fails**

Run: `go test ./internal/statusinfo/`
Expected: compile error (package does not exist).

- [ ] **Step 3: Write `statusinfo.go`**

```go
// Package statusinfo gathers and formats what the Status panel shows. Pure
// functions except Gather, which runs read-only jj queries.
package statusinfo

import (
	"bytes"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	appContext "github.com/idursun/jjui/internal/ui/context"
)

type Info struct {
	Repo      string
	ChangeID  string
	Bookmark  string // nearest bookmark at or below @; "" when none
	Remote    string
	Ahead     int
	Behind    int
	HasRemote bool // Ahead/Behind are meaningful (the remote bookmark exists)
	FetchedAt time.Time
	Revset    string
	Mode      string // "journal" or "graph"
}

var readOnly = []string{"--no-graph", "--color", "never", "--quiet", "--ignore-working-copy"}

func WorkingCopyArgs() []string {
	return append([]string{"log", "-r", "@", "-T", `change_id.shortest(8) ++ "\n"`}, readOnly...)
}

// ClosestBookmarkArgs lists the local bookmarks on the nearest bookmarked
// ancestor of @ (including @ itself).
func ClosestBookmarkArgs() []string {
	return append([]string{"log", "-r", "heads(::@ & bookmarks())", "-T", `local_bookmarks.map(|b| b.name()).join(",") ++ "\n"`}, readOnly...)
}

// CountArgs prints one "x" per commit in revset; Count sums them.
func CountArgs(revset string) []string {
	return append([]string{"log", "-r", revset, "-T", `"x"`}, readOnly...)
}

func FirstLine(output []byte) string {
	line, _, _ := strings.Cut(string(output), "\n")
	return strings.TrimSpace(line)
}

func FirstBookmark(output []byte) string {
	name, _, _ := strings.Cut(FirstLine(output), ",")
	return strings.TrimSpace(name)
}

func Count(output []byte) int { return bytes.Count(output, []byte("x")) }

// FetchTime is the modification time of git's FETCH_HEAD (colocated repos
// first, then jj's internal git store); zero when the repo was never fetched.
func FetchTime(repoRoot string) time.Time {
	for _, p := range []string{
		filepath.Join(repoRoot, ".git", "FETCH_HEAD"),
		filepath.Join(repoRoot, ".jj", "repo", "store", "git", "FETCH_HEAD"),
	} {
		if st, err := os.Stat(p); err == nil {
			return st.ModTime()
		}
	}
	return time.Time{}
}

func FormatAge(now, at time.Time) string {
	if at.IsZero() {
		return "never"
	}
	d := now.Sub(at)
	switch {
	case d < time.Minute:
		return "just now"
	case d < time.Hour:
		return fmt.Sprintf("%dm ago", int(d.Minutes()))
	case d < 24*time.Hour:
		return fmt.Sprintf("%dh ago", int(d.Hours()))
	default:
		return fmt.Sprintf("%dd ago", int(d.Hours()/24))
	}
}

// Lines renders the three Status panel lines.
func (i Info) Lines(now time.Time) [3]string {
	fetched := "fetched " + FormatAge(now, i.FetchedAt)
	var second string
	switch {
	case i.Bookmark == "":
		second = "(no bookmark) · " + fetched
	case i.HasRemote:
		second = fmt.Sprintf("%s ↑%d ↓%d %s · %s", i.Bookmark, i.Ahead, i.Behind, i.Remote, fetched)
	default:
		second = fmt.Sprintf("%s · not on %s · %s", i.Bookmark, i.Remote, fetched)
	}
	return [3]string{
		fmt.Sprintf("%s  @ %s", i.Repo, i.ChangeID),
		second,
		fmt.Sprintf("%s · %s", i.Mode, i.Revset),
	}
}

// Gather runs the read-only queries. Any failure leaves that field at its
// zero value; the panel must never block or error the UI.
func Gather(runner appContext.CommandRunner, repoRoot, remote, revset, mode string, now time.Time) Info {
	info := Info{Repo: filepath.Base(repoRoot), Remote: remote, Revset: revset, Mode: mode, FetchedAt: FetchTime(repoRoot)}
	if out, err := runner.RunCommandImmediate(WorkingCopyArgs()); err == nil {
		info.ChangeID = FirstLine(out)
	}
	if out, err := runner.RunCommandImmediate(ClosestBookmarkArgs()); err == nil {
		info.Bookmark = FirstBookmark(out)
	}
	if info.Bookmark != "" && remote != "" {
		local, remoteRef := info.Bookmark, info.Bookmark+"@"+remote
		aheadOut, aheadErr := runner.RunCommandImmediate(CountArgs(remoteRef + ".." + local))
		behindOut, behindErr := runner.RunCommandImmediate(CountArgs(local + ".." + remoteRef))
		if aheadErr == nil && behindErr == nil {
			info.HasRemote = true
			info.Ahead, info.Behind = Count(aheadOut), Count(behindOut)
		}
	}
	return info
}
```

- [ ] **Step 4: Run the statusinfo tests**

Run: `go test ./internal/statusinfo/`
Expected: PASS.

- [ ] **Step 5: Write the panel model**

Create `internal/ui/statuspanel/statuspanel.go`:

```go
// Package statuspanel is jjj's Status panel (spec §7): three lines refreshed
// after every load, computed off the UI loop by statusinfo.Gather.
package statuspanel

import (
	"path/filepath"
	"time"

	tea "charm.land/bubbletea/v2"
	"github.com/charmbracelet/x/ansi"

	"github.com/idursun/jjui/internal/config"
	"github.com/idursun/jjui/internal/statusinfo"
	"github.com/idursun/jjui/internal/ui/common"
	appContext "github.com/idursun/jjui/internal/ui/context"
	"github.com/idursun/jjui/internal/ui/layout"
	"github.com/idursun/jjui/internal/ui/render"
)

type infoMsg struct{ info statusinfo.Info }

type Model struct {
	ctx    *appContext.MainContext
	mode   func() string
	info   statusinfo.Info
	loaded bool
}

// New builds the panel; mode reports "journal" or "graph" at gather time.
func New(ctx *appContext.MainContext, mode func() string) *Model {
	return &Model{ctx: ctx, mode: mode}
}

func (m *Model) Init() tea.Cmd { return m.reload() }

func (m *Model) Update(msg tea.Msg) tea.Cmd {
	switch msg := msg.(type) {
	case infoMsg:
		m.info, m.loaded = msg.info, true
	case common.RefreshMsg, common.UpdateRevisionsSuccessMsg:
		return m.reload()
	}
	return nil
}

func (m *Model) reload() tea.Cmd {
	ctx, mode := m.ctx, m.mode()
	remote := config.GetGitDefaultRemote(config.Current)
	return func() tea.Msg {
		return infoMsg{info: statusinfo.Gather(ctx, ctx.Location, remote, ctx.CurrentRevset, mode, time.Now())}
	}
}

func (m *Model) ViewRect(dl *render.DisplayContext, box layout.Box) {
	lines := m.info.Lines(time.Now())
	if !m.loaded {
		lines = [3]string{filepath.Base(m.ctx.Location), "…", m.ctx.CurrentRevset}
	}
	tb := dl.Text(box.R.Min.X, box.R.Min.Y, render.ZBase)
	for i, line := range lines {
		if i > 0 {
			tb.NewLine()
		}
		tb.Write(ansi.Truncate(line, box.R.Dx(), "…"))
	}
	tb.Done()
}
```

- [ ] **Step 6: Mount it in the shell**

In `internal/ui/shell/shell.go`:

- add the field `status *statuspanel.Model` and the import `"github.com/idursun/jjui/internal/ui/statuspanel"`;
- in `New`, after `s.journalMode.Store(...)`: `s.status = statuspanel.New(ctx, func() string { if s.journalMode.Load() { return "journal" }; return "graph" })`;
- `Init`: `return tea.Batch(s.detail.Init(), s.status.Init())`;
- `Update`: the final line becomes `return tea.Batch(s.detail.Update(msg), s.status.Update(msg)), false`;
- `Render`: the Status panel body becomes `s.status.ViewRect(dl, inner)`; delete `drawStatusPlaceholder` and the now-unused `path/filepath` import.

- [ ] **Step 7: Smoke and interactive check**

```bash
make build
JJJ_CONFIG_DIR=/tmp/jjjcfg python3 scripts/pty-smoke.py bin/jjj ~/Developer/Home/ticktock --wait 12 \
  --expect "ticktock  @" --expect "main ↑0 ↓0 origin · fetched" --expect "journal · all()"
make check
```

Expected: `ok` (ticktock's `main` matches `main@origin`, so ↑0 ↓0), green `make check`. Then in `bin/jjj ~/Developer/Home/jjjournal` the second line must read `main ↑N ↓0 origin · fetched …` with N = number of jjj commits not yet pushed.

- [ ] **Step 8: Commit**

```bash
jj commit -m "feat(jjj): Status panel — working copy, nearest bookmark ahead/behind, fetch age, mode"
jj bookmark set main -r @-
```

---

### Task 7: Install `jjj`, retire the `jjournal` wrapper, house bookkeeping

No Go code. Everything here is file edits outside the fork plus `backlog init` inside it. Run the steps in order; each is idempotent.

**Files:**
- Create: `~/bin/jjj` (built), `CLAUDE.md`, `backlog/` (via `backlog init`)
- Delete: `~/Developer/Home/Scripts/jjournal`, `~/.config/jjournal/`
- Modify: `~/.zshrc` (one dash entry), `/Users/kortum/Humdrum/Claude/Custom Tooling Index.md`, `/Users/kortum/Humdrum/Claude/Projects/jjjournal.md`, `/Users/kortum/.claude/projects/-Users-kortum-Developer-Home-slog/memory/user-commits-as-journal.md`
- Create: symlink `/Users/kortum/Humdrum/Claude/Issues/jjjournal`

- [ ] **Step 1: Install and verify**

```bash
make install
ls -la ~/bin/jjj && ~/bin/jjj -version
cd ~/Developer/Home/ticktock && jjj    # look, then q
```

Expected: the binary exists; the TUI opens in journal mode with three panels.

- [ ] **Step 2: Retire the wrapper and its profile**

```bash
rm ~/Developer/Home/Scripts/jjournal
rm -r ~/.config/jjournal
python3 - <<'PY'
p='/Users/kortum/.zshrc'; s=open(p).read()
old='    "jjournal|🥋 jjournal      — browse this repo\'s full commit history (journal rows + detail pane)"'
new='    "jjj|🥋 jjj           — commit journal TUI for this repo (jjjournal: dated rows, detail pane)"'
assert s.count(old)==1; open(p,'w').write(s.replace(old,new)); print("dash entry -> jjj")
PY
zsh -n ~/.zshrc && echo "zsh syntax OK"
```

The `jj journal` alias in `~/.config/jj/config.toml` stays (it is the plain-CLI view, spec §2).

- [ ] **Step 3: Update the Custom Tooling Index**

```bash
python3 - <<'PY'
import datetime
p='/Users/kortum/Humdrum/Claude/Custom Tooling Index.md'
lines=open(p).read().split('\n')
def find(prefix, before=None):
    idx=[i for i,l in enumerate(lines) if l.startswith(prefix)]
    if before:
        lim=[i for i,l in enumerate(lines) if l.startswith(before)][0]
        idx=[i for i in idx if i<lim]
    assert len(idx)==1,(prefix,idx); return idx[0]
# 1. drop the jjournal wrapper row from the Scripts table
del lines[find('| `jjournal` | **Pinned in `dash`**')]
# 2. dash-entries row
lines[find('| `jjournal` | 🥋 jjournal —', before='## Aliases (`~/.zshrc`)')] = \
 '| `jjj` | 🥋 jjj — commit journal TUI for this repo (jjjournal: dated rows, detail pane) | opens `jjj` in the current directory\'s repo. Simple fall-through entry (no case block). |'
# 3. jjui row: it is stock; the journal is jjj
lines[find('| `jjui` | Jujutsu TUI')] = \
 '| `jjui` | Jujutsu TUI — Go | Terminal UI for jj (v0.10.8): revset-driven log, preview pane, rebase/squash/bookmarks/op log. Runs **stock** — no `~/.config/jjui/config.toml`. The commit-journal fork is `jjj` (see `~/bin/` table). Gotcha: `jjui -config` creates an empty `config.toml` before opening `$EDITOR`. `brew install jjui`. Project: https://github.com/idursun/jjui |'
# 4. ~/bin/ table: jjj row after custard
i=find('| `custard` |')
lines.insert(i+1,
 '| `jjj` | **Pinned in `dash`.** Commit-journal TUI for jj repos: a tracking fork of jjui (repo `~/Developer/Home/jjjournal`, github.com/humdrum-tiv/jjjournal). Opens on the whole visible history (`all()`) sorted newest-first with day/week headers, lazygit-style panels — [1] Status (working copy, nearest bookmark ↑/↓ vs origin, fetch age, mode + revset), [4] Journal, [0] Detail (`jj show --stat --git`) — digits/tab focus, `J` toggles the stock graph, `{`/`}` jump days; every jjui operation works in the Journal panel. Config: `~/.config/jjj/config.toml`, `JJJ_CONFIG_DIR`, `<repo>/.jjj/`; jjui\'s config is never read. Build/install: `make install` in the repo (`~/bin/jjj`); `make check` keeps upstream `jjui` building. Spec/plan under `docs/superpowers/`. |')
# 5. jj config row: jjj compiles its own templates
i=find('| `~/.config/jj/config.toml` |')
old='`jjournal` (jjui under `JJUI_CONFIG_DIR=~/.config/jjournal`) reads `log_journal` for its rows.'
assert old in lines[i]; lines[i]=lines[i].replace(old,'`jjj` compiles its own copy of the row templates; the alias stays for the plain CLI.')
# 6. bump updated
i=find('updated: '); lines[i]='updated: '+datetime.date.today().isoformat()
open(p,'w').write('\n'.join(lines)); print("index updated")
PY
```

- [ ] **Step 4: Backlog.md in the fork, with a thin CLAUDE.md**

The fork has upstream's `AGENTS.md` and no `CLAUDE.md`, so the guide goes into a new `CLAUDE.md` (ours; never touches `AGENTS.md`):

```bash
cd /Users/kortum/Developer/Home/jjjournal
backlog init "jjjournal" --agent-instructions claude --integration-mode cli --task-prefix TASK --zero-padded-ids 3
python3 - <<'PY'
import re
p='backlog/config.yml'; s=open(p).read()
s=re.sub(r'(?m)^default_status:.*$', 'default_status: "Backlog"', s)
s=re.sub(r'(?ms)^statuses:.*?(?=^\w|\Z)', 'statuses: ["Backlog", "In progress", "Paused", "Done"]\n', s)
open(p,'w').write(s); print(open(p).read())
PY
python3 - <<'PY'
p='CLAUDE.md'; s=open(p).read()
head='''@AGENTS.md

# jjjournal (jjj)

Tracking fork of jjui turned into a commit-journal TUI. Read before working here:

- `docs/superpowers/specs/2026-09-04-jjjournal-design.md` — the approved design
- `docs/superpowers/plans/` — implementation plans, one per phase
- `UPSTREAM.md` — which upstream files may be touched and how to sync

Rules: new behaviour goes in new packages; upstream files get seams only, each
listed in `UPSTREAM.md`; `make check` must stay green (it also builds the stock
`jjui`). This machine is jj-only: every VCS mutation goes through `jj`.

'''
open(p,'w').write(head+s); print("CLAUDE.md prefixed")
PY
```

Then capture the follow-up phases as tasks (each is its own future plan):

```bash
backlog task create "P2: Files, Bookmarks and Notes panels; accordion and zoom" -l feature --priority high \
  -d "Spec §3.2 (accordion, + / _ zoom), §5 Notes (full description body, enter opens describe), §6 Files (standalone model adapted from operations/details; file diff in Detail), §7 Bookmarks (upstream bookmarkpane hosted persistently). Panels 2, 3, 6." --plain
backlog task create "P3: Tasks panel from Backlog.md frontmatter with commit linking" -l feature --priority medium \
  -d "Spec §8: parse backlog/tasks/*.md and backlog/config.yml directly; group by status; ⋄TASK-nnn markers on journal rows; f filters the journal to a task; read-only." --plain
backlog task create "P4: Stats panel — compact strip and zoomed heatmap" -l feature --priority medium \
  -d "Spec §9: one jj log stamp query bucketed by day; today/week/month/all, streak, weekday histogram; enter on a heatmap day sets the revset." --plain
backlog task create "P5: Push a day's changes to the Obsidian daily note" -l feature --priority medium \
  -d "Spec §10: E on a day header; callout wrapped in <!-- jjj:repo:date --> markers, idempotent replace, inserted above the first base fence; [journal] daily_note/callout/create_note." --plain
backlog task create "P6: scripts/sync-upstream.sh, UPSTREAM.md automation, e2e scenario" -l chore --priority medium \
  -d "Spec §12.4 and §13: make sync-upstream (fetch upstream, rebase our stack, make check, move main), record last synced tag, one jjj scenario on the e2e PTY harness (needs Docker)." --plain
backlog task create "Register shell keys as rebindable actions" -l tech-debt --priority low \
  -d "Digits, tab, J, {, } are intercepted raw in shell.HandleKey (Task 3/5 of the P0-P1 plan). Move them into upstream's actions/bindings system so users can rebind them and they show in help." --plain
backlog task create "Phase 2: cross-repo timeline" -l feature --priority low \
  -d "Separate feature per the user (2026-09-04): one chronological journal across every repo under ~/Developer. Needs its own spec; not part of the jjj P0-P6 phases." --plain
backlog task list --plain
```

Commit the bookkeeping inside the fork:

```bash
jj commit -m "chore: Backlog.md tracking, CLAUDE.md pointer, follow-up phase tasks"
jj bookmark set main -r @-
```

- [ ] **Step 5: Vault symlink and the issues Base in the project doc**

```bash
mkdir -p /Users/kortum/Humdrum/Claude/Issues
ln -sfn /Users/kortum/Developer/Home/jjjournal/backlog/tasks /Users/kortum/Humdrum/Claude/Issues/jjjournal
```

In `/Users/kortum/Humdrum/Claude/Projects/jjjournal.md`:

1. Frontmatter: set `updated` to today; set `last_update` to `"P0-P1 shipped: jjj installed to ~/bin, journal mode + three-panel shell; jjournal wrapper retired."`; set `next` to `"P2 plan: Files, Bookmarks, Notes panels, accordion and zoom (backlog TASK-001 in the fork)."`.
2. Under `## Activity log`, insert as the first bullet:
   `- **<today>** — P0-P1 done. jjj builds from the fork with identity vars, compiled-in defaults, the ui.Extension hook, the RowSource seam and journal mode; Status/Journal/Detail panels; installed to ~/bin/jjj; jjournal wrapper and ~/.config/jjournal removed; dash entry and Custom Tooling Index updated; Backlog.md initialised in the fork with P2-P6 and two follow-up tasks; issues Base added here.`
3. Under `## Issues / blockers`, append this block verbatim:

````markdown
```base
filters:
  and:
    - file.inFolder("Claude/Issues/jjjournal")
properties:
  note.status:
    displayName: Status
  note.priority:
    displayName: Priority
  note.labels:
    displayName: Labels
  file.name:
    displayName: Issue
views:
  - type: table
    name: Open
    filters:
      and:
        - note.status != "Done"
    order:
      - file.name
      - status
      - priority
      - labels
    sort:
      - property: status
        direction: ASC
  - type: table
    name: Done
    filters:
      and:
        - note.status == "Done"
    order:
      - file.name
      - status
      - priority
      - labels
```
````

4. Under `## Key files & commands`, add `- Binary: ~/bin/jjj (make install); plan: docs/superpowers/plans/2026-09-04-jjjournal-p0-p1.md`.

- [ ] **Step 6: Memory pointer**

In `/Users/kortum/.claude/projects/-Users-kortum-Developer-Home-slog/memory/user-commits-as-journal.md`, replace the sentence beginning `Today's surfaces:` with: `Today's surfaces: jjj (the jjjournal fork, ~/bin/jjj: journal mode, day/week headers, Status/Journal/Detail panels) and jj journal in the plain CLI; the jjournal wrapper is gone.` Leave the rest as is.

---

### Task 8: Quickstart doc, upstream record, final checks, first push

**Files:**
- Create: `docs/jjj.md`
- Modify: `UPSTREAM.md`

- [ ] **Step 1: Write `docs/jjj.md`**

```markdown
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

## Development

Spec: `docs/superpowers/specs/2026-09-04-jjjournal-design.md`.
Plans: `docs/superpowers/plans/`. Upstream rules and sync: `UPSTREAM.md`.
Smoke test without Docker: `make smoke REPO=/path/to/a/jj/repo`.
```

- [ ] **Step 2: Finish `UPSTREAM.md`**

Append:

```markdown
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
```

- [ ] **Step 3: Final verification**

```bash
make check
make build
rm -rf /tmp/jjjcfg && mkdir -p /tmp/jjjcfg
JJJ_CONFIG_DIR=/tmp/jjjcfg python3 scripts/pty-smoke.py bin/jjj ~/Developer/Home/ticktock --wait 12 \
  --expect "[1]─Status" --expect "[4]─Journal" --expect "[0]─Detail" --expect "══ Week" --expect "── Mon 2026-07-13" --expect "Commit ID"
JJJ_CONFIG_DIR=/tmp/jjjcfg python3 scripts/pty-smoke.py bin/jjj ~/Developer/Home/ticktock --wait 12 --key J --after 6 --expect "Mon 2026-07-13 08:55"
JJJ_CONFIG_DIR=/tmp/jjjcfg python3 scripts/pty-smoke.py bin/jjj ~/Developer/Home/ticktock --wait 8 --cols 100 --expect "revset: all()"
git -C /Users/kortum/Developer/Home/jjjournal diff --stat main@upstream main -- internal cmd/jjui   # read-only: upstream files touched
```

Expected: everything `ok`; the diff-stat lists only `internal/config/loader.go`, `internal/ui/ui.go`, `internal/ui/revisions/revisions.go`, `internal/jj/commands.go` among pre-existing upstream files (everything else is new). If another upstream file shows up, either revert it or add it to `UPSTREAM.md` with a reason.

- [ ] **Step 4: Commit and push the fork**

```bash
jj commit -m "docs(jjj): quickstart, upstream record, key collisions"
jj bookmark set main -r @-
jj log -r 'main@upstream..main' --no-graph -T 'change_id.short() ++ " " ++ description.first_line() ++ "\n"'
jj git push -b main
```

This is the first push of jjj work to the public fork (github.com/humdrum-tiv/jjjournal); upstream's GitHub workflows will run there. Confirm the log shows only the jjj commits on top of upstream before pushing.

---

## Self-review

**Spec coverage (this plan = P0 + P1):**

| Spec section | Task |
|---|---|
| §2 decisions (tracking fork, `jjj`, module path, stock jjui untouched) | 1, 3 (`go build ./cmd/jjui` in `make check`) |
| §3.1 panels 0, 1, 4 | 3, 6 |
| §3.2 digits, tab, click focus, Detail scrolling | 3; zoom and accordion → P2 task |
| §3.3 narrow fallback | 3 |
| §4 journal mode: ordering, headers, both templates, `all()`, limit, operations on rows, working-copy marker, `{`/`}` | 2, 4, 5 |
| §5 Detail commands | 2 (defaults), 3 (panel); Notes → P2 |
| §7 Status panel | 6 |
| §11 config dirs, defaults, `[journal]`/`[shell]` | 1, 2 |
| §12.1 packages, §12.2 seams, §12.3 rules, §12.4 manual sync | 1, 3, 4, 8 |
| §13 unit tests; PTY smoke instead of Docker e2e | every task; e2e → P6 task |
| §14 P0 spike, P1 journal | 3–5 (spike), 5–7 (P1) |
| §15 bookkeeping | 7 |

Deferred with a backlog task in Task 7: Files/Bookmarks/Notes/zoom (P2), Tasks (P3), Stats (P4), export (P5), sync script + e2e (P6), rebindable shell keys, cross-repo (phase 2).

**Placeholder scan:** no TBD/TODO; every code step carries the code; every command carries its expected output.

**Type consistency:** `RowSource`/`RowSourceFactory`/`DefaultRowSource`/`SetRowSourceFactory` (Task 4) are what Task 5's `rowSource` and `Attach` use; `journal.Options{Now, DayStyle, WeekStyle}`, `Stamp`, `Arranged.HeaderIndices`, `Source.HeaderIndices()` and `JumpTarget` match between Tasks 4 and 5; `ui.Core{Revisions, Revset, Status, RenderMain}` and `Extension{Attach, Init, Update, HandleKey, Render}` match between Task 3's declaration, Task 5's `Attach`, and `cmd/jjj`; `jjj.Config.Journal.{Enabled, Template, Limit}` and `Shell.{Enabled, LeftWidth, NarrowColumns}` (Task 2) are the fields Tasks 3 and 5 read; `statusinfo.Gather(runner, repoRoot, remote, revset, mode, now)` (Task 6) is called with that signature by `statuspanel.reload`.
