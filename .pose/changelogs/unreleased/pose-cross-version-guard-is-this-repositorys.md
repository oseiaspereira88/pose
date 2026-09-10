---
spec: pose-cross-version-guard-is-this-repositorys
category: fixed
breaking: false
refs:
---

The cross-version compatibility test fails on this repository's CI and skips
everywhere else. It used to fail on any CI at all.

The guard exists because a skip on the machine whose verdict gates a release is a
test quietly not running. But a repository that vendors this engine as a submodule
and runs its suite is also CI, and its submodule checkout has no tags: it cannot
build the previous release, it is not responsible for this engine's release
history, and there is nothing for it to configure. The guard told it to fix
something that was not its to fix, and broke the adopting repository's build the
same day v4.0.0 reached it — on a pull request whose only content was the
adoption.

The signal is now the repository the workflow runs for, compared against the
release repository the engine already names as a constant. The protection
survives: a shallow checkout in this repository's own CI still fails rather than
skips.

The alternative — having consumers fetch tags for the submodule — would spread
this engine's test preconditions to everyone who vendors it, which is the shape of
the defect rather than a fix for it.
