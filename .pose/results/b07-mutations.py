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

M = [
 ("status ilegível é tratado como limpo", [(CP, 'if err != nil {\n\t\treturn nil, err\n\t}\n\tvar dirty []string', 'if err != nil {\n\t\treturn nil, nil\n\t}\n\tvar dirty []string')], P, "TestGitStateThatCannotBeReadIsUnknownNotClean"),
 ("repositório danificado é tratado como ausente", [(GS, ' && !hasGitMetadata(root) {', ' {')], P, "TestGitStateThatCannotBeReadIsUnknownNotClean"),
 ("diretório sem repositório é tratado como desconhecido", [(CP, 'if errors.Is(err, ErrGitUnavailable) {', 'if errors.Is(err, ErrGitUnavailable) && false {')], P, "TestGitStateThatCannotBeReadIsUnknownNotClean"),
 ("conteúdo não é comparado", [(GS, 'gitBlobID(format, content) != object ||', '(gitBlobID(format, content) != object && false) ||')], P, "TestAssumeUnchangedAndSkipWorktree|TestWorktreeDivergenceReadsPaths"),
 ("bit de execução ignorado", [(GS, 'executable != (mode == "100755")', '(executable != (mode == "100755")) && false')], P, "TestWorktreeDivergenceReadsPaths"),
 ("link não é comparado", [(GS, 'info.Mode()&os.ModeSymlink == 0 || rerr != nil || gitBlobID(format, []byte(target)) != object', '(info.Mode()&os.ModeSymlink == 0 || rerr != nil || gitBlobID(format, []byte(target)) != object) && false')], P, "TestWorktreeDivergenceReadsPaths"),
 ("arquivo removido não é divergência", [(GS, 'if lerr != nil {\n\t\t\tdiverged = append(diverged, name)\n\t\t\tcontinue\n\t\t}', 'if lerr != nil {\n\t\t\tcontinue\n\t\t}')], P, "TestWorktreeDivergenceReadsPaths"),
 ("arquivo indexado fora do commit não é divergência", [(GS, '"ls-files", "--cached", "--others"', '"ls-files", "--others"')], P, "TestWorktreeDivergenceReadsPaths"),
 ("rename perde o caminho de origem", [(GS, 'entry.OrigPath = fields[i]', '_ = fields[i]')], P, "TestWorktreeDivergenceReadsPaths"),
 ("matriz de validação é regeneração", [(GS, 'if rest == "validation-matrix.json" {\n\t\t\treturn false\n\t\t}', '')], P, "TestDerivedRegenerationIsAListNotAPrefix"),
 ("prefixo autoriza arquivo rastreado", [(GS, 'return !tracked\n\t}\n\treturn false', 'return true\n\t}\n\treturn false')], P, "TestDerivedRegenerationIsAListNotAPrefix"),
 ("política vira regeneração por prefixo", [(GS, 'strings.HasPrefix(p, ".pose/assessments/"), strings.HasPrefix(p, ".pose/state/"):', 'strings.HasPrefix(p, ".pose/assessments/"), strings.HasPrefix(p, ".pose/state/"), strings.HasPrefix(p, ".pose/policy/"):')], P, "TestDerivedRegenerationIsAListNotAPrefix"),
 ("run termina mesmo com divergência", [(RD, 'if len(changed) > 0 {\n\t\treturn finish(ReviewRunFailed, "the run changed files in the sealed worktree: "', 'if false {\n\t\treturn finish(ReviewRunFailed, "the run changed files in the sealed worktree: "')], C, "TestDispatchJudgesTheContent"),
 ("worktree não é verificado antes do run", [(RD, '} else if len(diverged) > 0 {\n\t\treturn finish(ReviewRunFailed, "the sealed worktree does not hold the sealed content: "', '} else if false {\n\t\treturn finish(ReviewRunFailed, "the sealed worktree does not hold the sealed content: "')], C, "TestTheWorktreeIsComparedWithTheSealedCommit"),
 ("política alterada durante o run não é stale", [(RD, 'plan.PolicyDigest != "" && directoryDigest', 'false && directoryDigest')], C, "TestARunWhosePolicyChangedWhileItRanIsStale"),
 ("bundle superado não é stale", [(RD, 'n > 0 && bundles[n-1].BundleID != plan.Bundle.BundleID', 'false')], P, "TestADispatchStopsGoverning"),
 ("commit alheio invalida o run", [(RD, 'if reason := s.dispatchNoLongerGoverns(plan); reason != "" {', 'if head, _ := ReadGitHead(s.Root); head != plan.SealedCommit {\n\t\treturn finish(ReviewRunStale, "stale: HEAD moved")\n\t}\n\tif reason := s.dispatchNoLongerGoverns(plan); reason != "" {')], C, "TestAnUnrelatedCommitDuringTheRun"),
 ("snapshot desconhecido é limpo", [(SC, 'if w != nil && w.unknown != "" {', 'if false {')], P, "TestAnUnreadableSnapshotIsNotClean"),
 ("status por path desconhecido é limpo", [(RB, 'return true, " (git status unknown: " + err.Error() + ")"', 'return false, ""')], P, "TestAnUnreadableStatusIsNotACleanPath"),
 ("projeção declara coerência com status ilegível", [(OP, 'snap.Coherent = false\n\t\tsnap.Limitations = append(snap.Limitations, "git status could not be read', '_ = snap\n\t\tsnap.Limitations = append(snap.Limitations, "git status could not be read')], P, "TestTheProjectionIsNotCoherentWhenStatusCannotBeRead"),
]
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
        r = subprocess.run(f"cd {work} && go test {pkgs} -run '{run}' -count=1", shell=True, capture_output=True, text=True, timeout=900)
        out = r.stdout + r.stderr
        build_error = 'build failed' in out or 'setup failed' in out
        verdict = 'killed' if r.returncode and not build_error else ('build-error' if build_error else 'survived')
        results.append({'mutation': name, 'tests': run, 'verdict': verdict})
        print(verdict, name, flush=True)
pathlib.Path(args.output).write_text(json.dumps({'mutations': results, 'killed': sum(r['verdict'] == 'killed' for r in results), 'total': len(results)}, indent=2, ensure_ascii=False) + '\n')
