---
spec: pose-upgrade-path-audit-fixes
category: fixed
breaking: false
refs:
---

`pose update`/`pose install` no longer silently drop AGENTS.md/POSE.md customizations outside `instance-owned` sections without a warning, and `pose update` without `--force` now seeds `.pose/policy`, `.pose/review-profiles` and the discovered module-metadata/indexes it previously left un-seeded on an old instance — `pose doctor` also gained checks for both an incomplete instance and orphaned module-metadata entries. An explicit `--locale en`/`pt-BR` is now honored reliably by `pose update`/`pose install`/`pose extension install`, including across a locale switch on an already-installed instance (no more duplicated sections). Module/stack discovery now excludes `testdata`/`fixture`/`fixtures` directories (fixes a false-positive that could invalidate closed specs' review evidence on this repository's own reinstalls) and classifies Android/Kotlin Gradle modules as `"android"` instead of the generic `"java"`. `pose install`'s post-install gate now says plainly when a failure is caused by governance debt already present in the target before the run started, and a forced update/install rerun preserves the project's declared identity across a directory rename instead of silently replacing it with the current directory name.
