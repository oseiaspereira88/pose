# POSE governed-delivery demo — recorded with charmbracelet/vhs.
#
# Regenerate:  vhs docs-site/docs/assets/demo.tape
# Requires:    pose on PATH, git, bash, a writable /tmp.
#
# The recording is scripted end to end. Hidden steps prepare a throwaway
# repository so every gate result shown is a real gate result. Timings are
# conservative; adjust Sleep values if the machine is slow.

Output docs-site/docs/assets/demo.gif

Set Shell bash
Set Width 960
Set Height 520
Set FontSize 17
Set Theme "Catppuccin Mocha"
Set Padding 24

# ------------------------------------------------------------------ off-camera
Hide

# Throwaway repository with an initial commit (pose install requires Git).
Type@0ms "rm -rf /tmp/pose-demo && mkdir -p /tmp/pose-demo && cd /tmp/pose-demo && git init -q && git config user.email demo@pose.local && git config user.name 'POSE Demo' && git commit -q --allow-empty -m 'init'"
Enter@0ms
Sleep 1s

Show
# --------------------------------------------------------------------- install

Sleep 500ms
Type "pose install ."
Sleep 800ms
Enter
Sleep 4s

# ------------------------------------------------------- a spec is not enough

Sleep 500ms
Type "pose new-spec customer-export"
Sleep 600ms
Enter
Sleep 2s

# The entry gate refuses an unready spec: no readiness, no execution.
Sleep 500ms
Type "pose lint-spec customer-export --ready-check"
Sleep 600ms
Enter
Sleep 3s

# ------------------------------------------------------ the structural gate

Sleep 500ms
Type "pose check --strict"
Sleep 600ms
Enter
Sleep 3s

Sleep 1500ms
