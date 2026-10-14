---
id: TASK-012.08
title: 'jjj panels: small UX gaps from P2a review'
status: Backlog
assignee: []
created_date: '2026-10-14 13:57'
updated_date: '2026-10-14 14:42'
labels:
  - feature
dependencies: []
parent_task_id: TASK-012
priority: low
ordinal: 20000
---

## Description

<!-- SECTION:DESCRIPTION:BEGIN -->
Deferred from the P2a final review: (1) undo/redo/oplog (u/o) are unreachable while Notes, Detail or Status is focused; spec 3.3 assumed they live in the ui scope, but upstream binds them in revisions. (2) In Bookmarks, A (advance and push) confirms before advancing, does not name the bookmarks or remote, and with nothing selected a bare jj git push may push more than was advanced. No undo hint is shown, and u means untrack inside the pane. (3) esc from a file diff in Detail returns to Journal, not Files. (4) In Bookmarks, J pages down (upstream shift+j) instead of toggling the journal, but the docs present J as global.
<!-- SECTION:DESCRIPTION:END -->

## Implementation Notes

<!-- SECTION:NOTES:BEGIN -->
(5) The scroll wheel over upstream's inline details (l on Journal) also scrolls the Files panel. Seen in the P2a final-fix round.

(6) If a resize removes the focused Files panel's box while a confirmation is open, focus falls back to Journal and the confirmation stays armed but hidden. A later reload can put another revision's file list under the prompt, while Yes still splits the old change. Fix: send confirmation.CloseMsg to the panel on that forced focus move. (7) FocusNextPane is now always taken by jjj.shell, which overrides a custom ui.focus_next_pane binding that would open upstream's split pane.
<!-- SECTION:NOTES:END -->
