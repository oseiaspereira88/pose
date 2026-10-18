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
RC = "pose-mcp/internal/pose/review_closeout.go"
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
 ("matriz de validação é regeneração", [(GS, 'return derivedIndexFile(rest)', 'return strings.HasSuffix(rest, ".json") && !strings.Contains(rest, "/")')], P, "TestDerivedRegenerationIsAListNotAPrefix"),
 ("prefixo autoriza arquivo rastreado", [(GS, 'return !tracked\n\t}\n\treturn false', 'return true\n\t}\n\treturn false')], P, "TestDerivedRegenerationIsAListNotAPrefix"),
 ("política vira regeneração por prefixo", [(GS, 'case strings.HasPrefix(p, ".pose/results/"),', 'case strings.HasPrefix(p, ".pose/policy/"):\n\t\treturn true\n\tcase strings.HasPrefix(p, ".pose/results/"),')], P, "TestDerivedRegenerationIsAListNotAPrefix"),
 ("run termina mesmo com divergência", [(RD, 'if len(changed) > 0 {\n\t\treturn finish(ReviewRunFailed, "the run changed files in the sealed worktree: "', 'if false {\n\t\treturn finish(ReviewRunFailed, "the run changed files in the sealed worktree: "')], C, "TestDispatchJudgesTheContent"),
 ("worktree não é verificado antes do run", [(RD, '} else if len(diverged) > 0 {\n\t\treturn finish(ReviewRunFailed, "the sealed worktree does not hold the sealed content: "', '} else if false {\n\t\treturn finish(ReviewRunFailed, "the sealed worktree does not hold the sealed content: "')], C, "TestTheWorktreeIsComparedWithTheSealedCommit"),
 ("commit alheio invalida o run", [(RD, 'if reason := s.dispatchNoLongerGoverns(plan); reason != "" {\n\t\treturn finish(ReviewRunStale, reason)', 'if head, _ := ReadGitHead(s.Root); head != plan.SealedCommit {\n\t\treturn finish(ReviewRunStale, "stale: HEAD moved")\n\t}\n\tif reason := s.dispatchNoLongerGoverns(plan); reason != "" {\n\t\treturn finish(ReviewRunStale, reason)')], C, "TestAnUnrelatedCommitDuringTheRun"),
 ("snapshot desconhecido é limpo", [(SC, 'if w != nil && w.unknown != "" {', 'if false {')], P, "TestAnUnreadableSnapshotIsNotClean"),
 ("status por path desconhecido é limpo", [(RB, 'return true, " (git status unknown: " + err.Error() + ")"', 'return false, ""')], P, "TestAnUnreadableStatusIsNotACleanPath"),
 ("projeção declara coerência com status ilegível", [(OP, 'snap.Coherent = false\n\t\tsnap.Limitations = append(snap.Limitations, "git status could not be read', '_ = snap\n\t\tsnap.Limitations = append(snap.Limitations, "git status could not be read')], P, "TestTheProjectionIsNotCoherentWhenStatusCannotBeRead"),
 ("inputs do bundle não são comparados com os de hoje", [(RD, 'if prepared.BundleDigest != plan.Bundle.BundleDigest {\n\t\treturn "stale: the bundle was superseded', 'if false {\n\t\treturn "stale: the bundle was superseded')], P + " " + C, "TestB07IndependentSameSecondSupersession|TestB07IndependentSourceChangedDuringRun|TestADispatchStopsGoverning"),
 ("checkout do commit não é comparado com os inputs selados", [(RD, '} else if differing := sealedInputsThatDiffer(plan.Bundle, prepared); len(differing) > 0 {', '} else if differing := sealedInputsThatDiffer(plan.Bundle, prepared); false && len(differing) > 0 {')], C, "TestB07IndependentBundleInputsNotCommit"),
 ("run inicia sem rever a pertinência", [(RD, 'return finish(ReviewRunFailed, "the run was not started: "+reason)', 'reason = ""')], C, "TestB07ARunDoesNotStart"),
 ("policy alterada durante o run não é stale", [(RD, 'now != plan.PolicyDigest {', 'false {')], C, "TestARunWhosePolicyChangedWhileItRanIsStale"),
 ("backslash vira separador", [(GS, 'if p == "" || path.Clean(p) != p {', 'p = strings.ReplaceAll(p, "\\\\", "/")\n\tif p == "" || path.Clean(p) != p {')], P, "TestB07BackslashIsA"),
 ("state é regeneração por prefixo", [(GS, 'return derivedStateFiles[rest]', 'return true')], P, "TestB07IndependentDerivedRules"),
 ("remoção de arquivo derivado é aceita", [(GS, 'if lerr != nil {\n\t\t\tdiverged = append(diverged, name)\n\t\t\tcontinue\n\t\t}\n\t\tif allowed', 'if lerr != nil {\n\t\t\tif allowed == nil || !allowed(name, true) {\n\t\t\t\tdiverged = append(diverged, name)\n\t\t\t}\n\t\t\tcontinue\n\t\t}\n\t\tif allowed')], P, "TestB07DerivedNamesAreExact"),
 ("gitlink selado ausente é aceito", [(GS, 'if !gitlinkMaterialized(full, object) {', 'if false {')], P, "TestB07IndependentGitlink"),
 ("closeout aceita matriz e resultado selado", [(CP, 'return IsDerivedRegeneration(path, tracked)\n}', 'return strings.HasPrefix(path, ".pose/indexes/") || strings.HasPrefix(path, ".pose/results/") || IsDerivedRegeneration(path, tracked)\n}')], P, "TestB07IndependentCloseoutPrefix"),
 ("rejeição no mesmo instante perde para a aprovação", [(RC, 'if aApproves != bApproves {\n\t\t\treturn aApproves\n\t\t}', '_, _ = aApproves, bApproves')], P, "TestB07IndependentNegativeSameSecond"),
 ("link na policy é ignorado", [(OP, 'return errors.New("not a regular file: " + p)', 'return nil')], P, "TestB07ALinkInAPolicyDirectory"),
 ("perda de coerência não é mudança", [(OP, 'if snap.Coherent && !now.Coherent {', 'if false {')], P, "TestB07IndependentSnapshotRevalidation"),
 ("subdiretório é comparado como raiz", [(GS, 'worktree = top', '_ = top')], P, "TestB07IndependentSubdirectory"),
 ("erro de leitura vira edição", [(GS, 'return nil, &GitUnknownError{Op: "read " + name, Cause: rerr.Error()}', 'diverged = append(diverged, name)\n\t\t\t\tcontinue')], P, "TestB07IndependentReadErrorDiagnostic"),
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
import sys
pathlib.Path(args.output).write_text(json.dumps({'mutations': results, 'killed': sum(r['verdict'] == 'killed' for r in results), 'total': len(results)}, indent=2, ensure_ascii=False) + '\n')
sys.exit(0 if all(r['verdict'] == 'killed' for r in results) else 1)
