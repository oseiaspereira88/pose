---
spec: pose-debt-marker-lexical-precision
category: fixed
breaking: false
refs:
---

Technical-debt comment markers are now explicit uppercase declarations, so
ordinary lowercase prose such as Portuguese `todo` and explanatory `stub`
comments no longer creates fictitious debt. Executable stub and panic
constructs remain detected.
