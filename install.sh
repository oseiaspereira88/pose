#!/usr/bin/env bash
# POSE installer — installs the POSE machinery into a target repository.
#
# Usage:
#   bash install.sh <target-dir> [options]
#
# Options:
#   --project-name <name>   Name substituted into {{PROJECT_NAME}} (default: basename of target)
#   --project-id <id>       MCP project id (default: proj.<project-name>)
#   --mcp-binary <path>     Pre-built pose-mcp binary to vendor (skips go build)
#   --skip-mcp              Do not build/vendor the MCP server
#   --force                 Overwrite AGENTS.md/POSE.md even if already present
#   --locale <tag>          Docs/templates locale (default: en; available: pt-BR)
#   --allow-non-git         Allow installing into a directory that is not a git repo
#   -h | --help             Show this help
#
# Behavior:
#   - Machinery (scripts/workflows/rules/templates/hooks, skills, CLI) is
#     always updated in place.
#   - Instance content (.pose/specs, adr, knowledge, reports, roadmaps,
#     changelogs) is NEVER touched.
#   - AGENTS.md/POSE.md are installed once with placeholders substituted;
#     re-runs keep the user's edits unless --force.
#   - Ends by running `./pose init && ./pose check --strict` in the target.
set -euo pipefail

log()  { printf '[pose-install] %s\n' "$*"; }
err()  { printf '[pose-install] ERROR: %s\n' "$*" >&2; }
die()  { err "$*"; exit 1; }

usage() { sed -n '2,20p' "${BASH_SOURCE[0]}" | sed 's/^# \{0,1\}//'; }

SRC="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"

TARGET=""
PROJECT_NAME=""
PROJECT_ID=""
MCP_BINARY=""
SKIP_MCP=0
FORCE=0
ALLOW_NON_GIT=0
LOCALE="en"

while [[ $# -gt 0 ]]; do
  case "$1" in
    --project-name) PROJECT_NAME="${2:?--project-name requires a value}"; shift 2 ;;
    --project-id)   PROJECT_ID="${2:?--project-id requires a value}"; shift 2 ;;
    --mcp-binary)   MCP_BINARY="${2:?--mcp-binary requires a value}"; shift 2 ;;
    --skip-mcp)     SKIP_MCP=1; shift ;;
    --force)        FORCE=1; shift ;;
    --locale)       LOCALE="${2:?--locale requires a value}"; shift 2 ;;
    --allow-non-git) ALLOW_NON_GIT=1; shift ;;
    -h|--help)      usage; exit 0 ;;
    -*)             die "unknown option: $1" ;;
    *)              [[ -n "$TARGET" ]] && die "unexpected argument: $1"; TARGET="$1"; shift ;;
  esac
done

[[ -n "$TARGET" ]] || { usage; die "target directory is required"; }
[[ -d "$TARGET" ]] || die "target directory does not exist: $TARGET"
TARGET="$(cd "$TARGET" && pwd)"
[[ "$TARGET" != "$SRC" ]] || die "target must not be the distribution directory itself"

if [[ "$ALLOW_NON_GIT" -eq 0 ]] && ! git -C "$TARGET" rev-parse --git-dir >/dev/null 2>&1; then
  die "target is not a git repository: $TARGET (use --allow-non-git to override)"
fi

PROJECT_NAME="${PROJECT_NAME:-$(basename "$TARGET")}"
PROJECT_ID="${PROJECT_ID:-proj.${PROJECT_NAME}}"

log "source:       $SRC"
log "target:       $TARGET"
log "project name: $PROJECT_NAME"
log "project id:   $PROJECT_ID"

PRIOR_INSTALL=0
[[ -f "$TARGET/.pose/scripts/pose-lib.sh" ]] && PRIOR_INSTALL=1
[[ "$PRIOR_INSTALL" -eq 1 ]] && log "existing POSE install detected — machinery will be updated, instance content preserved"

# --- 1. Machinery -----------------------------------------------------------
# Engine dirs are replaced wholesale (users must not edit them; upgrades win).
# Extensible dirs are merged: distribution files are updated, files the user
# added (custom rules/workflows/templates/skills) are preserved.
ENGINE_DIRS=(
  ".pose/scripts"
  ".pose/hooks"
)
EXTENSIBLE_DIRS=(
  ".pose/workflows"
  ".pose/rules"
  ".pose/templates"
  ".agents/skills"
)

for dir in "${ENGINE_DIRS[@]}"; do
  [[ -d "$SRC/$dir" ]] || die "distribution is missing machinery dir: $dir"
  mkdir -p "$TARGET/$(dirname "$dir")"
  rm -rf "${TARGET:?}/$dir"
  cp -a "$SRC/$dir" "$TARGET/$dir"
  log "engine (replaced): $dir"
done

for dir in "${EXTENSIBLE_DIRS[@]}"; do
  [[ -d "$SRC/$dir" ]] || die "distribution is missing machinery dir: $dir"
  mkdir -p "$TARGET/$dir"
  cp -a "$SRC/$dir/." "$TARGET/$dir/"
  log "machinery (merged): $dir"
done

# .claude/skills are relative symlinks into .agents/skills — recreate them.
mkdir -p "$TARGET/.claude/skills"
if [[ -d "$SRC/.claude/skills" ]]; then
  find "$TARGET/.claude/skills" -maxdepth 1 -type l -delete
  while IFS= read -r -d '' link; do
    name="$(basename "$link")"
    dest="$(readlink "$link")"
    ln -sfn "$dest" "$TARGET/.claude/skills/$name"
  done < <(find "$SRC/.claude/skills" -maxdepth 1 -type l -print0)
  log "machinery: .claude/skills (symlinks)"
fi

# CLI dispatcher.
install -m 0755 "$SRC/pose" "$TARGET/pose"
log "machinery: pose (CLI)"

# --- 2. Config indexes: copied only when absent ------------------------------
mkdir -p "$TARGET/.pose/indexes"
for idx in "$SRC"/.pose/indexes/*.json; do
  name="$(basename "$idx")"
  if [[ ! -f "$TARGET/.pose/indexes/$name" ]]; then
    cp "$idx" "$TARGET/.pose/indexes/$name"
    log "index (seed): $name"
  fi
done

# --- 3. Legal texts: vendored under .pose/ ----------------------------------
cp "$SRC/LICENSE" "$TARGET/.pose/LICENSE"
cp "$SRC/NOTICE" "$TARGET/.pose/NOTICE"
log "vendored: .pose/LICENSE, .pose/NOTICE"

# --- 4. Root docs: installed once, placeholders substituted ------------------
substitute_placeholders() {
  # sed -i is not portable between GNU/BSD; write to a temp file instead.
  local file="$1" tmp
  tmp="$(mktemp)"
  sed -e "s/{{PROJECT_NAME}}/${PROJECT_NAME//\//\\/}/g" \
      -e "s/{{PROJECT_ID}}/${PROJECT_ID//\//\\/}/g" "$file" >"$tmp"
  mv "$tmp" "$file"
}

DOCS_SRC="$SRC"
if [[ "$LOCALE" != "en" ]]; then
  if [[ -d "$SRC/locales/$LOCALE" ]]; then
    DOCS_SRC="$SRC/locales/$LOCALE"
    log "locale: $LOCALE (docs/templates localized)"
    if [[ -d "$SRC/locales/$LOCALE/templates" ]]; then
      cp -a "$SRC/locales/$LOCALE/templates/." "$TARGET/.pose/templates/"
      log "machinery (locale override): .pose/templates"
    fi
    for locale_dir in .pose/workflows .pose/rules .agents/skills; do
      if [[ -d "$SRC/locales/$LOCALE/$locale_dir" ]]; then
        mkdir -p "$TARGET/$locale_dir"
        cp -a "$SRC/locales/$LOCALE/$locale_dir/." "$TARGET/$locale_dir/"
        log "machinery (locale override): $locale_dir"
      fi
    done
  else
    log "locale '$LOCALE' not available — falling back to en"
  fi
fi

for doc in AGENTS.md POSE.md; do
  if [[ -f "$TARGET/$doc" && "$FORCE" -eq 0 ]]; then
    log "kept existing: $doc (use --force to overwrite)"
  else
    cp "$DOCS_SRC/$doc" "$TARGET/$doc"
    substitute_placeholders "$TARGET/$doc"
    log "installed: $doc"
  fi
done

# --- 5. MCP server -----------------------------------------------------------
write_mcp_wrapper() {
  mkdir -p "$TARGET/.pose/bin"
  cat >"$TARGET/.pose/bin/pose-mcp-claude" <<WRAPPER
#!/usr/bin/env bash
# Generated by POSE install.sh — project root is derived, never hardcoded.
export POSE_PROJECT_ROOT="\$(cd "\$(dirname "\$0")/../.." && pwd)"
export POSE_DEFAULT_PROJECT_ID="${PROJECT_ID}"
exec "\$(dirname "\$0")/pose-mcp" --stdio "\$@"
WRAPPER
  chmod 0755 "$TARGET/.pose/bin/pose-mcp-claude"
}

if [[ "$SKIP_MCP" -eq 1 ]]; then
  log "MCP: skipped (--skip-mcp)"
elif [[ -n "$MCP_BINARY" ]]; then
  [[ -x "$MCP_BINARY" ]] || die "--mcp-binary is not an executable file: $MCP_BINARY"
  mkdir -p "$TARGET/.pose/bin"
  install -m 0755 "$MCP_BINARY" "$TARGET/.pose/bin/pose-mcp"
  write_mcp_wrapper
  log "MCP: vendored binary + wrapper at .pose/bin/"
elif command -v go >/dev/null 2>&1 && { [[ -f "$SRC/../pose-mcp/go.mod" ]] || [[ -f "$SRC/pose-mcp/go.mod" ]]; }; then
  # Dual-home: monorepo tem o fonte em ../pose-mcp; o repo standalone em ./pose-mcp.
  MCP_SRC="$SRC/../pose-mcp"
  [[ -f "$MCP_SRC/go.mod" ]] || MCP_SRC="$SRC/pose-mcp"
  log "MCP: building from source (go build)…"
  ( cd "$MCP_SRC" && go build -o "$TARGET/.pose/bin/pose-mcp" ./cmd/pose-mcp ) \
    || die "go build of pose-mcp failed"
  write_mcp_wrapper
  log "MCP: built binary + wrapper at .pose/bin/"
else
  log "MCP: not installed — no --mcp-binary given and no Go toolchain/source tree available."
  log "     The CLI and all gates work without it. To add MCP later, re-run with --mcp-binary <path>."
fi

# Seed .mcp.json only when the target has none.
if [[ ! -f "$TARGET/.mcp.json" && -x "$TARGET/.pose/bin/pose-mcp-claude" ]]; then
  cat >"$TARGET/.mcp.json" <<MCPJSON
{
  "mcpServers": {
    "pose": {
      "type": "stdio",
      "command": "./.pose/bin/pose-mcp-claude"
    }
  }
}
MCPJSON
  log "seeded: .mcp.json (server \"pose\")"
fi

# --- 6. Instance directories, schema stamp + final gate ----------------------
( cd "$TARGET" && ./pose init )
SCHEMA_VERSION="$(sed -n 's/^POSE_SCHEMA_VERSION=\([0-9]\+\)$/\1/p' "$SRC/.pose/scripts/pose-lib.sh" | head -1)"
if [[ -n "$SCHEMA_VERSION" ]]; then
  if [[ -f "$TARGET/.pose/schema-version" ]]; then
    log "schema-version present ($(cat "$TARGET/.pose/schema-version")) — run './pose upgrade' in the target if behind"
  else
    printf '%s\n' "$SCHEMA_VERSION" >"$TARGET/.pose/schema-version"
    log "schema-version stamped: v$SCHEMA_VERSION"
  fi
fi
( cd "$TARGET" && ./pose index >/dev/null )
log "indexes regenerated for the target"
log "running final gate: ./pose check --strict"
if ( cd "$TARGET" && ./pose check --strict ); then
  log "install complete — POSE is ready in $TARGET"
  log "next steps: ./pose hooks install; ./pose new-spec <slug>; ./pose suggest feature"
else
  die "post-install gate failed: ./pose check --strict (see output above)"
fi
