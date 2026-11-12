### Added
- Optional per-agent `review_profile` selects fresh and resumed governed-review commands without widening read tasks, adjudication or smoke.
- This repository's Claude review profile executes tests in the mandatory shell sandbox, with private Go cache/temp scratch and an absolute write denial for sealed source. Missing bubblewrap, socat or jq refuses launch with installation advice; sandbox failure never falls back to an unrestricted command.
