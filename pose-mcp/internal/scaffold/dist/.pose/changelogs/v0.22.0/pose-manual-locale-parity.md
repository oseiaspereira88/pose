---
spec: pose-manual-locale-parity
category: fixed
breaking: false
refs:
---

The pt-BR manual is back in parity with the English one. `pose upgrade` ships
the manual matching each instance's locale, so thirteen feature commits that
never crossed into the translation had been distributing a POSE without the
release lifecycle, `pose state`, docs governance or the signed extension
ecosystem to every pt-BR instance. The drift ran both ways: the translation
carried rationale the English manual had lost, and documented three real MCP
tools that appeared in no English file. A symmetric check now fails on either
kind of drift, closing for the manuals the same gap that was already closed for
translated skills.
