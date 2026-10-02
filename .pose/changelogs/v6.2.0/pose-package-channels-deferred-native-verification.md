---
spec: pose-package-channels-deferred-native-verification
category: changed
breaking: false
---

Native Homebrew and WinGet verification is deferred to an explicitly dispatched
GitHub Actions round after implementation. It no longer starts automatically
after releases or blocks other implementation work; actual channel support
claims still require successful native evidence.
