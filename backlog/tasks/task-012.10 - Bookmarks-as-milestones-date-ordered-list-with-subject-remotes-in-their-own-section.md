---
id: TASK-012.10
title: Remotes panel (local vs each remote) and a separate Tags panel for milestones
status: Backlog
assignee: []
created_date: '2026-10-16 19:41'
updated_date: '2026-10-16 20:36'
labels:
  - feature
dependencies: []
parent_task_id: TASK-012
priority: medium
ordinal: 22000
---

## Description

<!-- SECTION:DESCRIPTION:BEGIN -->
Decided 2026-10-16: milestones move from bookmarks to git tags (jj 0.45 has jj tag set/delete/list/track and git push/fetch --tag). Tags never advance (bookmark advance can drag a milestone bookmark forward), are immutable heads by default (trunk() | tags() | ...), and stay out of the remote branch list. So the panels split in two. (1) The Bookmarks panel becomes Remotes: trunk/working bookmarks (main first) with sync state vs every tracked remote (= ↑n ↓n ⇅a/b; slice 1 already ships the markers, pinning and clipping), plus per remote the last fetch time. (2) A new Tags panel lists your milestone tags newest first with date and commit subject; selecting one reveals it in Journal. Only your own tags: exclude tags fetched from other remotes (e.g. jjjournal carries upstream jjui tags v0.10.x); decide the filter in the plan (tags on commits not reachable from an upstream remote's bookmarks, or tags tracked on origin, or untracked local). Optional helper: convert a milestone bookmark to a tag (jj tag set N -r N; jj bookmark delete N; push --tag N --bookmark N).
<!-- SECTION:DESCRIPTION:END -->

## Acceptance Criteria
<!-- AC:BEGIN -->
- [ ] #1 Remotes panel shows each local bookmark's sync state vs every tracked remote and each remote's last fetch
- [ ] #2 Tags panel lists own milestone tags newest first with date and subject, excluding tags from other projects' remotes
- [ ] #3 Selecting a tag reveals its commit in Journal
<!-- AC:END -->

## Implementation Notes

<!-- SECTION:NOTES:BEGIN -->
2026-10-16: slice 1 shipped with TASK-012.04: sync markers, trunk pinning, heading dropped, remote filter only while focused, rows clipped to width. Remaining: date and subject per bookmark, newest-first milestone order, separate Remotes section, reveal-in-Journal on select, a visible divider between trunk and milestones.

2026-10-16 decisions: (a) Tags filter = tags tracked on origin plus local tags not pushed anywhere; hide tags that exist only on another remote (e.g. upstream). (b) Placement = both: Tags is a second tab inside the Remotes panel (switch tabs within the panel) and also has its own digit key that focuses the panel on the Tags tab. Pick the digit in the plan (5 is reserved for Tasks/P3).
<!-- SECTION:NOTES:END -->
