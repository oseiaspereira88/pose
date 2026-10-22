"""Production mutations of B07 (spec pose-git-and-dispatch-snapshot-integrity).

Each mutation reverts one guarantee in a COPY of the repository (the working tree is
never edited), runs the named tests there and must make them fail on an assertion,
not on a build error. Run from the repository root:

    python3 -I .pose/results/b07-mutations.py --output <file.json> [--only <text>]
"""
import argparse, json, pathlib, shutil, subprocess, tempfile

parser = argparse.ArgumentParser(description=__doc__)
parser.add_argument('--output', required=True)
parser.add_argument('--only', default='', help='run only the mutations whose name contains this text')
args = parser.parse_args()
root = pathlib.Path('.').resolve()
GS = "pose-mcp/internal/pose/git_snapshot_integrity.go"
CP = "pose-mcp/internal/pose/closeout_plan.go"
RD = "pose-mcp/internal/pose/review_dispatch.go"
SC = "pose-mcp/internal/pose/git_spawn_cache.go"
RB = "pose-mcp/internal/pose/review_bundle.go"
OP = "pose-mcp/internal/pose/obligation_projection.go"
P = "./internal/pose"
C = "./internal/cli"

M = [('unreadable closeout status is clean', [('pose-mcp/internal/pose/closeout_plan.go', 'paths, err := gitChangedPaths(s.Root)\n\tif err != nil {\n\t\treturn nil, err\n\t}', 'paths, err := gitChangedPaths(s.Root)\n\tif err != nil {\n\t\treturn nil, nil\n\t}')], './internal/pose', 'TestGitStateThatCannotBeReadIsUnknownNotClean'), ('damaged repository is unavailable', [('pose-mcp/internal/pose/git_snapshot_integrity.go', 'if !hasGitMetadata(root) {', 'if true {')], './internal/pose', 'TestGitStateThatCannotBeReadIsUnknownNotClean'), ('hidden bytes are ignored', [('pose-mcp/internal/pose/git_snapshot_integrity.go', 'if strings.TrimSpace(string(hash)) != fields[2] {', 'if strings.TrimSpace(string(hash)) != fields[2] && false {')], './internal/pose', 'TestAssumeUnchangedAndSkipWorktreeDoNotHideAnEdit'), ('executable mode is ignored', [('pose-mcp/internal/pose/git_snapshot_integrity.go', 'if runtime.GOOS != "windows" &&', 'if false && runtime.GOOS != "windows" &&')], './internal/pose', 'TestHiddenExecutableModeIsDivergence'), ('rename source is lost', [('pose-mcp/internal/pose/git_snapshot_integrity.go', 'entry.OrigPath = string(out[:i])', '_ = string(out[:i])')], './internal/pose', 'TestWorktreeDivergenceReadsPathsAndTypesExactly'), ('validation matrix is an allowed producer output', [('pose-mcp/internal/pose/git_snapshot_integrity.go', 'func IsDerivedRegeneration(path string, tracked bool) bool {', 'func IsDerivedRegeneration(path string, tracked bool) bool {\n if path == ".pose/indexes/validation-matrix.json" { return true }')], './internal/pose', 'TestDerivedRegenerationIsAListNotAPrefix'), ('parent links are accepted', [('pose-mcp/internal/pose/review_dispatch.go', 'if info.Mode()&os.ModeSymlink != 0 {', 'if info.Mode()&os.ModeSymlink != 0 && false {')], './internal/pose', 'TestGitSnapshotParentLinkIsNotFollowed'), ('superseded bundle still governs', [('pose-mcp/internal/pose/review_dispatch.go', 'if prepared.BundleDigest != plan.Bundle.BundleDigest {', 'if false && prepared.BundleDigest != plan.Bundle.BundleDigest {')], './internal/pose', 'TestADispatchStopsGoverningWhenItsBundleIsSuperseded'), ('unknown shared snapshot is clean', [('pose-mcp/internal/pose/git_spawn_cache.go', 'if w != nil && w.unknown != "" {', 'if false {')], './internal/pose', 'TestAnUnreadableSnapshotIsNotClean'), ('unknown bundle path is clean', [('pose-mcp/internal/pose/review_bundle.go', 'return true, " (git status unknown: " + err.Error() + ")"', 'return false, ""')], './internal/pose', 'TestAnUnreadableStatusIsNotACleanPath'), ('unknown obligation status is coherent', [('pose-mcp/internal/pose/obligation_projection.go', 'snap.Coherent = false\n\t\tsnap.Limitations = append(snap.Limitations, "git status could not be read', '_ = snap\n\t\tsnap.Limitations = append(snap.Limitations, "git status could not be read')], './internal/pose', 'TestTheProjectionIsNotCoherentWhenStatusCannotBeRead')]
M += [('persisted relevance skips materialized consumed inputs', [('pose-mcp/internal/pose/review_dispatch.go', 'if err := verifyBundleMaterialization(s.Root, bundle); err != nil {\n\t\treturn "persisted review inputs no longer materialize: " + err.Error()\n\t}', '// mutation: skip persisted materialization')], './internal/pose', 'TestB07PersistedConsumedInputLink|TestB07PersistedRelevantSubmoduleHiddenEdit'), ('closeout exemption accepts deleted or replaced output', [('pose-mcp/internal/pose/closeout_plan.go', 'generated = err == nil', 'generated = true; _ = err')], './internal/pose', 'TestB07CloseoutOutputsMustRemainRegular'), ('run history uses timestamp text', [('pose-mcp/internal/pose/review_dispatch.go', 'return a.Before(b)', 'return out[i].StartedAt < out[j].StartedAt')], './internal/pose', 'TestB07R4BRunOrdering')]
results = []
for name, edits, pkgs, run in M:
    if args.only and args.only not in name:
        continue
    with tempfile.TemporaryDirectory(prefix="b07-mut-") as tmp:
        repo = pathlib.Path(tmp) / "repo"
        shutil.copytree(root, repo, ignore=shutil.ignore_patterns(".git", "node_modules", "*.test"))
        work = repo / "pose-mcp"
        for f, old, new in edits:
            p = repo / f
            cur = p.read_text()
            assert cur.count(old) == 1, (name, f, cur.count(old))
            p.write_text(cur.replace(old, new))
        r = subprocess.run(['go', 'test', pkgs, '-run', run, '-count=1'], cwd=work, capture_output=True, text=True, timeout=180)
        out = r.stdout + r.stderr
        build_error = 'build failed' in out or 'setup failed' in out
        verdict = 'killed' if r.returncode and '--- FAIL:' in out and not build_error else ('build-error' if build_error else 'survived')
        results.append({'mutation': name, 'tests': run, 'verdict': verdict})
        print(verdict, name, flush=True)
pathlib.Path(args.output).write_text(json.dumps({'mutations': results, 'killed': sum(r['verdict'] == 'killed' for r in results), 'total': len(results)}, indent=2, ensure_ascii=False) + '\n')

if any(r['verdict'] != 'killed' for r in results):
    raise SystemExit(1)
