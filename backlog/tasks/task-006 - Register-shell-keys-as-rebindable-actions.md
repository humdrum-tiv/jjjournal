---
id: TASK-006
title: Register shell keys as rebindable actions
status: Backlog
assignee: []
created_date: '2026-09-05 02:29'
updated_date: '2026-10-14 03:17'
labels:
  - tech-debt
dependencies: []
priority: low
ordinal: 6000
---

## Description

<!-- SECTION:DESCRIPTION:BEGIN -->
Digits, tab, J, {, } are intercepted raw in shell.HandleKey (Task 3/5 of the P0-P1 plan). Move them into upstream's actions/bindings system so users can rebind them and they show in help.
<!-- SECTION:DESCRIPTION:END -->

## Implementation Notes

<!-- SECTION:NOTES:BEGIN -->
Folded into TASK-012 P2a (spec §3.2): shell keys become actions in a shell scope.
<!-- SECTION:NOTES:END -->
