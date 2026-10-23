# B29 replacement qualification — 2026-10-23

Status: in-progress. No independent final approval or native closure is claimed.

Source checkpoint `7d9c07c` combines the operation-scoped update replacement with reviewed B30, B28 and B38. The original copier history and rejected delivery statements remain explicitly historical. Actual implementation and composition tests, failures, fixes and limitations are recorded in `2026-10-22-b29-update-preview-implementation.md`.

Fresh canonical native collection: 146/146 checks pass, zero errors, 208.0 seconds; command `pose validate --strict --report --json-out .pose/results/delivery-validation.json`, log `/tmp/pose-b29-composed-validation.log`. This is the registered matrix collection, not a substitution with direct Go tests. Direct full Go, vet, full affected-package race tests, six platform builds and compiled-CLI inotify mutation audit also pass. The audit observes filesystem events and does not claim syscall tracing or runtime testing on macOS/Windows.

Discovery refreshed pose-mcp at the composed source/matrix. Integration assessment retains heuristic unobserved-consumer findings and is not a clean contract certificate. Technical-debt assessment is INCOMPLETE because two legacy specs are invalid: `2026-08-15-pose-instance-engine-version-tracking.md` and `2026-08-15-pose-knowledge-durable-reference-type.md`; scanned content contains one covered TODO and zero uncovered markers, which does not prove unread sources are debt-free.

Actual separate source review was first attempted with Claude Opus, run `/home/go/.local/state/secondary-agent/20261023T040419-claude-3547062`; organization-disabled subscription access prevented execution. A separate same-vendor Codex source review was launched before the user's subsequent Claude-only instruction, run `/home/go/.local/state/secondary-agent/20261023T040719-codex-3642496`. No cross-vendor, human or cryptographic independence is inferred. Subsequent secondary reviews must use Claude Opus, per the user's current instruction.

The actual source review requested changes for three medium filesystem fidelity gaps. Regressions reproduced each before fixes; the implementation report records correction, full-suite/race/vet/build/audit evidence and conservative INCOMPLETE behavior where access, umask or hard-link effects cannot be safely evaluated. A further parent regression reproduced and corrected trailing-separator directory requirements. Source review remains a historical rejected attempt, not retroactively approved.

The user subsequently explicitly authorized resuming separate Codex secondary reviews because Claude configuration cannot be resolved now. This supersedes the interim Claude-only preference; preserve actual same-vendor execution attribution.

Pending: fresh native collection on corrected source, immutable artifact reconciliation, delivery surface gate, sealed final adjudication, native review verification and closure, dedicated PR and required CI. No release/tag publication, Harne8 adoption or shared-checkout/gitlink changes are authorized or performed by this qualification.
