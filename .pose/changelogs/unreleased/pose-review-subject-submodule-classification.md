---
spec: pose-review-subject-submodule-classification
category: fixed
breaking: false
refs:
---

A repository that vendors a dependency as a Git submodule can now seal a review
bundle. Previously any submodule in the attributed change set blocked the seal
with `unclassified review subject path`, because the classifier decides from the
path alone and a submodule has no shape that distinguishes it from a directory.
It is now recognised as a gitlink and reviewed as what it is: the commit it is
pinned to.
