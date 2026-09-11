---
spec: pose-one-follow-up-format
category: fixed
breaking: false
refs:
---

The spec template and the closeout skill show the one follow-up format POSE
reads — `- [open] <text> (owner:@alias crit:low|medium|high review:YYYY-MM-DD)`,
the group last, in parentheses — and `pose lint-spec` warns, in any spec
status, when ownership is written anywhere else.

Written any other way, ownership is ignored: the item reads as unowned, with no
criticality and no review date, and never becomes overdue. The template showed
`- [open] ` with no metadata at the one place an author writes a follow-up, and
the lint checked ownership only on done specs, so a wrong format could go
unnoticed indefinitely — this repository had 27 open items that way.

The lint also reads a follow-up's continuation lines now, as `pose followups`
does. It read only the first line, so a wrapped ownership group was owned for
one and "unowned" for the other.
