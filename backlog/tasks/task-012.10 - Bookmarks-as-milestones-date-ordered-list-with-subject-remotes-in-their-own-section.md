---
id: TASK-012.10
title: >-
  Bookmarks as milestones: date-ordered list with subject, remotes in their own
  section
status: Backlog
assignee: []
created_date: '2026-10-16 19:41'
labels:
  - feature
dependencies: []
parent_task_id: TASK-012
priority: medium
ordinal: 22000
---

## Description

<!-- SECTION:DESCRIPTION:BEGIN -->
User 2026-10-16: the hosted Bookmarks pane is confusing (redundant 'Bookmarks' heading, always-on 'Remotes:' filter line, ' local ' tag, repeated remote names, bare commit id). Bookmarks double as app milestones: the user wants to look back at them, not only main. Remote state is a separate concern: where each copy of the repo (each remote) is relative to local. Chosen direction (option 'readable rows + sync'): drop the heading, show the remote filter only while focused, rows show name, date and target commit subject, newest first; a separate Remotes section shows, per remote, each tracked bookmark's sync state (= in sync, ↑n ahead, ↓n behind, ⇅ diverged) and when it was last fetched. Selecting a milestone reveals it in Journal. Needs a row-renderer/sort seam in upstream internal/ui/bookmarkpane (record in UPSTREAM.md) or a jjj-owned list; decide in the plan. Subsumes the display part of TASK-012.04.
<!-- SECTION:DESCRIPTION:END -->

## Acceptance Criteria
<!-- AC:BEGIN -->
- [ ] #1 Bookmarks section lists local bookmarks newest first with date and subject, no heading or always-on remote line
- [ ] #2 A separate Remotes section shows per-remote sync state for every tracked bookmark
- [ ] #3 Selecting a bookmark reveals its commit in Journal
<!-- AC:END -->
