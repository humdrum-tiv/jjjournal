---
id: TASK-012.11
title: '? help: common tasks guide with explanations; light-theme selection readable'
status: Done
assignee: []
created_date: '2026-10-16 20:04'
updated_date: '2026-10-16 20:04'
labels:
  - feature
dependencies: []
parent_task_id: TASK-012
ordinal: 23000
---

## Description

<!-- SECTION:DESCRIPTION:BEGIN -->
User 2026-10-16: wanted a section in ? listing squash/absorb/describe/etc. with explanations, and could not read the selected Bookmarks row (grey highlight). Done: help.Intro + ScopeGroup.Wide seam (UPSTREAM.md); shell.Guide builds 'Common tasks' groups for Journal and Bookmarks with keys from live bindings; jjj binds ? to ui.open_help (scrollable, searchable overlay) and f1 to upstream's expanded status help. Light terminals: upstream's ':selected' bg is ANSI white, rendered mid-grey by many light palettes, so dimmed text vanished; jjj.ApplyTheme sets fixed 256-colour 253 for the selection keys unless the user set them in [ui.colors].
<!-- SECTION:DESCRIPTION:END -->

## Acceptance Criteria
<!-- AC:BEGIN -->
- [x] #1 ? opens help starting with common tasks and explanations
- [x] #2 Selected rows are readable on light terminals
<!-- AC:END -->
