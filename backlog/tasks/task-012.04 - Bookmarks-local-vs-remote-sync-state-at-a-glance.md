---
id: TASK-012.04
title: 'Bookmarks: local vs remote sync state at a glance'
status: Backlog
assignee: []
created_date: '2026-10-14 04:31'
labels:
  - feature
dependencies: []
parent_task_id: TASK-012
priority: medium
ordinal: 16000
---

## Description

<!-- SECTION:DESCRIPTION:BEGIN -->
User request 2026-10-13: see remotes and which commit each is on vs local. Today the hosted bookmark pane shows remote child rows with commit ids only when expanded. Add, on each local bookmark row: per tracked remote a sync marker (= in sync, ↑n ahead, ↓m behind, ⇅ diverged) computed from jj log counts (remote..local / local..remote), plus the remote commit's short id and subject when they differ; optionally a Journal row decoration for remote bookmark positions (name@remote). Builds on P2a Bookmarks panel (TASK-012.01).
<!-- SECTION:DESCRIPTION:END -->

## Acceptance Criteria
<!-- AC:BEGIN -->
- [ ] #1 Each local bookmark row shows sync state vs every tracked remote without expanding
- [ ] #2 Diverged bookmarks are visually distinct
- [ ] #3 Counts refresh after fetch/push/advance
<!-- AC:END -->
