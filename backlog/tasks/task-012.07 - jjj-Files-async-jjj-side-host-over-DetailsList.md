---
id: TASK-012.07
title: 'jjj Files: async jjj-side host over DetailsList'
status: Backlog
assignee: []
created_date: '2026-10-14 13:57'
updated_date: '2026-10-14 13:58'
labels:
  - feature
dependencies: []
parent_task_id: TASK-012
priority: medium
ordinal: 19000
---

## Description

<!-- SECTION:DESCRIPTION:BEGIN -->
P2a hosts upstream's modal details.Operation persistently in the Files panel (plan deviation). It loads synchronously on the UI loop and carries modal assumptions (common.Close on split, broadcast confirmations) that the shell has to work around. Replace with a jjj-side host over DetailsList that loads off the UI loop (spec 4.2). From the P2a final review.
<!-- SECTION:DESCRIPTION:END -->
