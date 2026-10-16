---
id: TASK-012.04
title: 'Bookmarks: local vs remote sync state at a glance'
status: Done
assignee: []
created_date: '2026-10-14 04:31'
updated_date: '2026-10-16 20:24'
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
- [x] #1 Each local bookmark row shows sync state vs every tracked remote without expanding
- [x] #2 Diverged bookmarks are visually distinct
- [x] #3 Counts refresh after fetch/push/advance
<!-- AC:END -->

## Implementation Notes

<!-- SECTION:NOTES:BEGIN -->
2026-10-16: user wants remote state shown separately from the bookmark (milestone) list, as where each copy of the repo is. Display now planned in the Bookmarks-as-milestones task; this task keeps the sync computation (counts, refresh after fetch/push/advance).

2026-10-16: done as the first slice of TASK-012.10. Shell bookmarksPanel.loadSync runs jj bookmark list plus two jj log counts (jj.BookmarkOnlyIn) per tracked remote whose commit differs from local, on init and every common.RefreshMsg; seq-tagged. Pane journal view (bookmarkpane/jjj_view.go) shows per-remote markers = ↑n ↓n ⇅a/b on each local row and words on expanded rows; diverged in red. Pinned bookmarks ([bookmarks] pinned, default main/master/trunk) sort first tagged trunk.
<!-- SECTION:NOTES:END -->
