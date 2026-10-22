# B38 integration requalification — 2026-10-22

PR198 previously completed native closeout on 2026-10-19 with actual R13 approval. Main 2d42b84 merged at e27f1f7, preserving both append-only state histories and regenerating current assessments/indexes. No release/tag publication, historical attestation rewrite, Harne8 adoption/gitlink or shared-checkout changes.

Separate Codex R1 report records one medium source finding extending historical low R11-L14 into positive next-cycle approval. Domain test proved corrupt loose ref was classified absent; real CLI two-release fixture proved open-next moved from refusal to success after corrupting only the newer tag ref. Logs `/tmp/pose-b38-broken-tag-before.log` (domain) and `/tmp/pose-b38-broken-tag-cli-before.log` (CLI). Initial CLI fixture serialization error was corrected before claiming the reproduction. All corruption and synthetic publication events confined to disposable test repos.

Fix: retain successful Git stderr as uncertain result, preserve unknown/corrupt versus proven absent, use one tag reader in status and CLI plan/check/backfill. Targeted regression suite passed after source fix; final global strict validation, race and separate sealed adjudication pending. Historical source review low rationale is superseded for this unsafe consumer variant, not silently edited.
