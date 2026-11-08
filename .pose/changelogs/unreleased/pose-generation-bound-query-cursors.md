---
spec: pose-generation-bound-query-cursors
category: changed
breaking: false
---

Paginated MCP list tools continue a query over the collection it was issued
for. A cursor is now signed and bound to its tool, project, filters and the
ordered collection (with its index generation); a collection that changed
between pages, another query, an expired token or a server restart is refused
with a `cursor-*` code and the caller restarts from the first page, instead of
silently skipping or repeating items. Version 1 cursors are refused, a page is
at most 500 items, and each page discloses its generation and collection. Calls
without cursor and limit are unchanged.
