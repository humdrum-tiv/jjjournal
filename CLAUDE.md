@AGENTS.md

# jjjournal (jjj)

Tracking fork of jjui turned into a commit-journal TUI. Read before working here:

- `docs/superpowers/specs/2026-09-04-jjjournal-design.md` — the approved design
- `docs/superpowers/plans/` — implementation plans, one per phase
- `UPSTREAM.md` — which upstream files may be touched and how to sync

Rules: new behaviour goes in new packages; upstream files get seams only, each
listed in `UPSTREAM.md`; `make check` must stay green (it also builds the stock
`jjui`). This machine is jj-only: every VCS mutation goes through `jj`.


<!-- BACKLOG.MD GUIDELINES START -->
<CRITICAL_INSTRUCTION>

## Backlog.md Workflow

This project uses Backlog.md for task and project management.

**For every user request in this project, run `backlog instructions overview` before answering or taking action.**

Use the overview to decide whether to search, read, create, or update Backlog tasks.

Use the detailed guides when needed:
- `backlog instructions task-creation` for creating or splitting tasks
- `backlog instructions task-execution` for planning and implementation workflow
- `backlog instructions task-finalization` for completion and handoff

Use `backlog <command> --help` before running unfamiliar commands. Help shows options, fields, and examples.

Do not edit Backlog task, draft, document, decision, or milestone markdown files directly. Use the `backlog` CLI so metadata, relationships, and history stay consistent.

</CRITICAL_INSTRUCTION>
<!-- BACKLOG.MD GUIDELINES END -->
