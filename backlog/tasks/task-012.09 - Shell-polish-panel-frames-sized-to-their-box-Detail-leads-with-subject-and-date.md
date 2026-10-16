---
id: TASK-012.09
title: >-
  Shell polish: panel frames sized to their box; Detail leads with subject and
  date
status: Done
assignee: []
created_date: '2026-10-16 19:41'
updated_date: '2026-10-16 19:41'
labels:
  - bug
dependencies: []
parent_task_id: TASK-012
ordinal: 21000
---

## Description

<!-- SECTION:DESCRIPTION:BEGIN -->
User 2026-10-16: boxes around sections not aligned; commit message and date should be elevated in Detail over commit id and committer. Cause of misalignment: lipgloss v2 Width/Height include the border, so drawPanel's Width(w-2)/Height(h-2) drew each frame 2 cols and 2 rows short of its box and content spilled over the right/bottom edges. Detail: [preview] revision_command in internal/jjj/defaults.toml now passes a jj show -T template (bold subject, body, author date + relative age, then dimmed change id, commit id, author), followed by --stat --git.
<!-- SECTION:DESCRIPTION:END -->

## Acceptance Criteria
<!-- AC:BEGIN -->
- [x] #1 Every panel border sits on its layout box
- [x] #2 Detail shows subject first, then date, ids and author dimmed
<!-- AC:END -->
