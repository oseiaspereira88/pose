---
spec: pose-signed-action-answers
category: added
breaking: false
refs:
---

Action answers can be proven without any external service. `pose identity add [<principal>] --key <file.pub> [--role <role>] --apply` registers a principal's SSH key (`ssh-ed25519` or the security-key form `sk-ssh-ed25519@openssh.com`) under `keys` in `.pose/policy/actions.json`, suggesting `human:<name>` from the git identity; `pose identity list` and `remove` manage them. `pose action resolve … --sign <key>` signs the answer with the user's own `ssh-keygen` (POSE never reads the key), `--signature <file>` attaches a signature made elsewhere over `pose action statement`, and POSE verifies the SSHSIG natively, records it on the answer and re-verifies it in `pose action show`. Under verified assurance an answer needs such a signature or a trusted issuer's claim; `require_presence` refuses a human answer whose security key was not touched, and `pose doctor` reports human role holders that cannot prove an answer.
