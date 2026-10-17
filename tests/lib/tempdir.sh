# shellcheck shell=bash
# Sourced by the test scripts (spec pose-test-scripts-remove-their-temp-files).
#
# pose_test_tmp_init puts every temporary file of the script, and of the
# commands it runs, under one directory that is removed when the script exits:
# on success, on a failed assertion (`exit 1`), on an error under `set -e` and
# on INT or TERM. It must run before the script's first `mktemp`.
#
# A script that needs a trap of its own composes it with `pose_test_tmp_on_exit`
# so the directory is still removed.

pose_test_tmp_init() {
  POSE_TEST_TMP="$(mktemp -d)"
  export POSE_TEST_TMP
  # Everything below, including `go build` and `go test`, creates its
  # temporary files here.
  export TMPDIR="$POSE_TEST_TMP"
  trap 'pose_test_tmp_remove' EXIT
  trap 'exit 130' INT
  trap 'exit 143' TERM
}

pose_test_tmp_remove() {
  [ -n "${POSE_TEST_TMP:-}" ] || return 0
  # Directories a Go test left read-only would defeat rm -rf.
  chmod -R u+w "$POSE_TEST_TMP" 2>/dev/null || true
  rm -rf "$POSE_TEST_TMP"
}
