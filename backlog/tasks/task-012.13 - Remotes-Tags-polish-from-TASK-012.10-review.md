---
id: TASK-012.13
title: 'Remotes/Tags: polish from TASK-012.10 review'
status: Backlog
assignee: []
created_date: '2026-10-16 21:58'
labels:
  - feature
dependencies: []
parent_task_id: TASK-012
priority: low
ordinal: 25000
---

## Description

<!-- SECTION:DESCRIPTION:BEGIN -->
Deferred minors from the 2026-10-16 final review of TASK-012.10: (1) jj git remote list may print a '(push: url)' suffix, so that remote never matches FETCH_HEAD and shows no age — verify and cut at the first space; (2) fetch ages freeze between refreshes (now fixed at load time); (3) tags load runs even when [shell] hidden = ["tags"], and BookmarkListAll runs twice per refresh (sync + tags) — share it; (4) Tags cursor is kept by index across reloads, keep it by name; (5) a failed jj tag list shows 'No tags' instead of an error; (6) user-facing strings still say bookmarks (help desc for key 3 'focus bookmarks', docs/jjj.md table rows, defaults comment); (7) a remote name with ' or a tag name with ; breaks the template/parse; (8) active and inactive tab labels look the same while panel 3 is unfocused; (9) no mouse selection on tag rows.
<!-- SECTION:DESCRIPTION:END -->
