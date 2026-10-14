---
id: TASK-012.01
title: 'P2a: scoped-key seam, Files/Bookmarks/Notes panels, accordion and zoom'
status: Done
assignee: []
created_date: '2026-10-14 03:17'
updated_date: '2026-10-14 04:39'
labels:
  - feature
dependencies: []
parent_task_id: TASK-012
priority: high
ordinal: 13000
---

## Description

<!-- SECTION:DESCRIPTION:BEGIN -->
Spec docs/superpowers/specs/2026-10-13-jjj-functional-design.md §3-§4. Extension.Scopes() seam + jjj_*.go intents (closes TASK-006); Files panel from DetailsList; Bookmarks = hosted bookmarkpane + advance (a / A); Notes; accordion, zoom, narrow fallback. Supersedes TASK-001.
<!-- SECTION:DESCRIPTION:END -->

## Acceptance Criteria
<!-- AC:BEGIN -->
- [ ] #1 All left panels live and focusable by digit
- [ ] #2 Bookmarks a runs jj bookmark advance --to the selected revision
- [ ] #3 Files follows the Journal cursor; Detail shows the file diff
- [ ] #4 With Files focused, revision keys do not leak
- [ ] #5 make check green
<!-- AC:END -->

## Implementation Notes

<!-- SECTION:NOTES:BEGIN -->
P2a shipped per docs/superpowers/plans/2026-10-13-jjj-p2a-panels.md.
<!-- SECTION:NOTES:END -->
