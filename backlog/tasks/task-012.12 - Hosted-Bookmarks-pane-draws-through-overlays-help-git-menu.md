---
id: TASK-012.12
title: 'Hosted Bookmarks pane draws through overlays (help, git menu)'
status: Backlog
assignee: []
created_date: '2026-10-16 20:04'
labels:
  - bug
dependencies: []
parent_task_id: TASK-012
priority: low
ordinal: 24000
---

## Description

<!-- SECTION:DESCRIPTION:BEGIN -->
The hosted upstream bookmark pane draws at render.ZMenuContent (it was built as an overlay), above the help overlay's backdrop/border (ZMenuBorder) and level with other menus, so its fill and text bleed through the left part of the ? help overlay (seen 2026-10-16 headless capture, rows over the Bookmarks panel). Fix options: draw the pane into a child DisplayContext and merge at ZBase, or a z-offset seam in bookmarkpane.
<!-- SECTION:DESCRIPTION:END -->

## Acceptance Criteria
<!-- AC:BEGIN -->
- [ ] #1 No Bookmarks panel content shows through ? help or the git menu
<!-- AC:END -->
