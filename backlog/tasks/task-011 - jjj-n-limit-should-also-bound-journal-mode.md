---
id: TASK-011
title: jjj -n/--limit should also bound journal mode
status: Backlog
assignee: []
created_date: '2026-09-05 03:05'
labels:
  - bug
dependencies: []
priority: low
ordinal: 11000
---

## Description

<!-- SECTION:DESCRIPTION:BEGIN -->
config.Current.Limit is ignored by the journal source; take min(Limit, journal.limit) when the flag is set.
<!-- SECTION:DESCRIPTION:END -->
