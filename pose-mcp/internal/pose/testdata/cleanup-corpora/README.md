# Cleanup corpus reproductions

These captures reproduce the historical table in `.pose/reports/2026-11-02-pose-v7-legacy-cleanup-both-corpora.md`; they do not measure the present repository.

The original table labels pose-dist `fcd11da9` and Harne8 `ba83cbfb`, measured by engine `59358c28`. The independent review reran the same inventory code with engine `f931afe0`. Its pose-dist capture is therefore at `f931afe0`; its Harne8 capture remains at `ba83cbfb`. Every class/count matches the earlier table. The distinct revisions are retained in `provenance.json` rather than rewriting the original measurement.

`pose-dist.json` retains the complete public raw output. `harne8-summary.json` omits only the private `details` field; it preserves classes, counts, completeness, losses, risks and declared writes. The raw SHA-256 and export SHA-256 are distinct integrity references. A digest is not an authenticated receipt. The complete original captures are retained locally at `.pose/private-evidence/legacy-cleanup/{pose-dist,harne8}.json`, ignored by Git. Public tests do not claim access to the private raw capture.

The review's before/after digest recipe used sorted relative paths, literal backslash-zero, and contents. The earlier author's recipe used `sha256sum` output; the two digests must not be compared to each other. The public tests instead execute a synthetic migration dry-run and compare their own before/after paths, modes, metadata and file contents under one consistent recipe.
