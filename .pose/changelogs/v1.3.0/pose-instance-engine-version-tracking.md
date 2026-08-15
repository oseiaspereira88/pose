---
spec: pose-instance-engine-version-tracking
category: added
breaking: false
refs: ISSUE#20
---

`pose version` now accepts an optional target path (`pose version <dir>`) and, when the resolved instance's machinery was delivered by a recorded engine version, prints `instance last updated by: pose <version>` — so checking whether a project is in sync with a given `pose` release no longer requires diffing content by hand. `pose update`/`pose install` now record their delivering version in `.pose/state/machinery-manifest.json`; instances updated before this change simply omit the line until their next sync.
