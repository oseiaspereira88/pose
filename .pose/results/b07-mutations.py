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
 ("prefixo autoriza arquivo rastreado", [(GS, 'return !tracked && derivedRecordName(strings.TrimPrefix(p, ".pose/reports/history/"), historyRecordRE)', 'return derivedRecordName(strings.TrimPrefix(p, ".pose/reports/history/"), historyRecordRE)')], P, "TestDerivedRegenerationIsAListNotAPrefix"),
 ("política vira regeneração por prefixo", [(GS, 'case strings.HasPrefix(p, ".pose/results/"):\n\t\treturn !tracked', 'case strings.HasPrefix(p, ".pose/policy/"):\n\t\treturn true\n\tcase strings.HasPrefix(p, ".pose/results/"):\n\t\treturn !tracked')], P, "TestDerivedRegenerationIsAListNotAPrefix"),
 ("run termina mesmo com divergência", [(RD, 'if len(changed) > 0 {\n\t\treturn finish(ReviewRunFailed, "the run changed files in the sealed worktree: "', 'if false {\n\t\treturn finish(ReviewRunFailed, "the run changed files in the sealed worktree: "')], C, "TestDispatchJudgesTheContent"),
 ("worktree não é verificado antes do run", [(RD, '} else if len(diverged) > 0 {\n\t\treturn finish(ReviewRunFailed, "the sealed worktree does not hold the sealed content: "', '} else if false {\n\t\treturn finish(ReviewRunFailed, "the sealed worktree does not hold the sealed content: "')], C, "TestTheWorktreeIsComparedWithTheSealedCommit"),
 ("commit alheio invalida o run", [(RD, 'if reason := s.dispatchNoLongerGoverns(plan); reason != "" {\n\t\treturn finish(ReviewRunStale, reason)', 'if head, _ := ReadGitHead(s.Root); head != plan.SealedCommit {\n\t\treturn finish(ReviewRunStale, "stale: HEAD moved")\n\t}\n\tif reason := s.dispatchNoLongerGoverns(plan); reason != "" {\n\t\treturn finish(ReviewRunStale, reason)')], C, "TestAnUnrelatedCommitDuringTheRun"),
 ("snapshot desconhecido é limpo", [(SC, 'if w != nil && w.unknown != "" {', 'if false {')], P, "TestAnUnreadableSnapshotIsNotClean"),
 ("status por path desconhecido é limpo", [(RB, 'return true, " (git status unknown: " + err.Error() + ")"', 'return false, ""')], P, "TestAnUnreadableStatusIsNotACleanPath"),
 ("projeção declara coerência com status ilegível", [(OP, 'snap.Coherent = false\n\t\tsnap.Limitations = append(snap.Limitations, "git status could not be read', '_ = snap\n\t\tsnap.Limitations = append(snap.Limitations, "git status could not be read')], P, "TestTheProjectionIsNotCoherentWhenStatusCannotBeRead"),
 ("inputs do bundle não são comparados com os de hoje", [(RD, 'if prepared.BundleDigest != plan.Bundle.BundleDigest {\n\t\treturn "stale: the bundle was superseded', 'if false {\n\t\treturn "stale: the bundle was superseded')], P + " " + C, "TestB07IndependentSameSecondSupersession|TestB07IndependentSourceChangedDuringRun|TestADispatchStopsGoverning"),
 ("checkout do commit não é comparado com os inputs selados", [(RD, '} else if differing := sealedInputsThatDiffer(plan.Bundle, prepared, worktree); len(differing) > 0 {', '} else if differing := sealedInputsThatDiffer(plan.Bundle, prepared, worktree); false && len(differing) > 0 {')], C, "TestB07IndependentBundleInputsNotCommit"),
 ("run inicia sem rever a pertinência", [(RD, 'return finish(ReviewRunFailed, "the run was not started: "+reason)', 'reason = ""')], C, "TestB07ARunDoesNotStart"),
 ("policy alterada durante o run não é stale", [(RD, 'now != plan.PolicyDigest {', 'false {')], C, "TestARunWhosePolicyChangedWhileItRanIsStale"),
 ("backslash vira separador", [(GS, 'if p == "" || path.Clean(p) != p {', 'p = strings.ReplaceAll(p, "\\\\", "/")\n\tif p == "" || path.Clean(p) != p {')], P, "TestB07BackslashIsA"),
 ("state é regeneração por prefixo", [(GS, 'return derivedStateFiles[rest]', 'return true')], P, "TestB07IndependentDerivedRules"),
 ("remoção de arquivo derivado é aceita", [(GS, 'if lerr != nil {\n\t\t\tdiverged = append(diverged, name)\n\t\t\tcontinue\n\t\t}\n\t\t// A directory above', 'if lerr != nil {\n\t\t\tif allowed == nil || !allowed(name, true) {\n\t\t\t\tdiverged = append(diverged, name)\n\t\t\t}\n\t\t\tcontinue\n\t\t}\n\t\t// A directory above')], P, "TestB07DerivedNamesAreExact"),
 ("gitlink selado ausente é aceito", [(GS, 'if linkAbove(worktree, name, checkedDirs) || !gitlinkMaterialized(full, object) {', 'if linkAbove(worktree, name, checkedDirs) || false {')], P, "TestB07IndependentGitlink"),
 ("closeout aceita matriz e resultado selado", [(CP, 'return IsDerivedRegeneration(path, tracked)\n}', 'return strings.HasPrefix(path, ".pose/indexes/") || strings.HasPrefix(path, ".pose/results/") || IsDerivedRegeneration(path, tracked)\n}')], P, "TestB07IndependentCloseoutPrefix"),
 ("rejeição no mesmo instante perde para a aprovação", [(RC, 'if aApproves != bApproves {\n\t\t\treturn aApproves\n\t\t}', '_, _ = aApproves, bApproves')], P, "TestB07IndependentNegativeSameSecond"),
 ("link na policy é ignorado", [(OP, 'return errors.New("not a regular file: " + p)', 'return nil')], P, "TestB07ALinkInAPolicyDirectory"),
 ("perda de coerência não é mudança", [(OP, 'if !now.Coherent {', 'if false {')], P, "TestB07IndependentSnapshotRevalidation"),
 ("subdiretório é comparado como raiz", [(GS, 'worktree = top', '_ = top')], P, "TestB07IndependentSubdirectory"),
 ("erro de leitura vira edição", [(GS, 'return nil, &GitUnknownError{Op: "read " + name, Cause: rerr.Error()}', 'diverged = append(diverged, name)\n\t\t\t\tcontinue')], P, "TestB07IndependentReadErrorDiagnostic"),
 ("subject do checkout não é comparado com o selado", [(RD, 'err != nil || digestBytes(bytes.ReplaceAll(raw, []byte("\\r\\n"), []byte("\\n"))) != entry.Digest', 'err != nil || false && digestBytes(bytes.ReplaceAll(raw, []byte("\\r\\n"), []byte("\\n"))) != entry.Digest')], C, "TestB07R2SealedSubjectNotCheckout"),
 ("blockers da preparação da origem são ignorados", [(RD, 'if len(prepared.Blockers) > 0 {\n\t\treturn "stale: the scope cannot be prepared', 'if false && len(prepared.Blockers) > 0 {\n\t\treturn "stale: the scope cannot be prepared')], C, "TestB07R2UnreadableSourceStatus"),
 ("conteúdo do submódulo não é comparado", [(GS, 'return derr == nil && len(diverged) == 0', '_, _ = diverged, derr\n\treturn true')], P, "TestB07R2DirtyGitlink"),
 ("submódulo por link vale como diretório", [(GS, 'info, err := os.Lstat(dir)\n\tif err != nil || !info.IsDir() {', 'info, err := os.Stat(dir)\n\tif err != nil || !info.IsDir() {')], P, "TestB07R2DirtyGitlink"),
 ("ambiente redireciona a verificação", [(GS, 'if !gitRedirectingEnv[name] {', 'if true || !gitRedirectingEnv[name] {')], P, "TestB07R2OtherWorktreeEnvironment|TestGitReadsIgnoreTheVariables"),
 ("bundles empatados: aprovação fica por último", [(RB, 'sort.SliceStable(group, func(i, j int) bool { return !negative[group[i].BundleID] && negative[group[j].BundleID] })', 'sort.SliceStable(group, func(i, j int) bool { return negative[group[i].BundleID] && !negative[group[j].BundleID] })')], P, "TestB07R2NegativeAcrossBundles"),
 ("digest da policy sem enquadramento", [(OP, 'fmt.Fprintf(h, "%d:%s%d:", len(name), name, len(raw))\n\t\th.Write(raw)', 'h.Write([]byte(name))\n\t\th.Write([]byte{0})\n\t\th.Write(raw)\n\t\th.Write([]byte{0})')], P, "TestB07R2PolicyDigestFraming|TestB07R2PolicyChangeWithEqualDigest"),
 ("closeout aceita link como regeneração", [(CP, 'err != nil || !info.Mode().IsRegular() {\n\t\treturn false', '(err != nil || !info.Mode().IsRegular()) && false {\n\t\treturn false')], P, "TestB07R2CloseoutType|TestB07ACloseoutCheckpoint"),
 ("HEAD ilegível não bloqueia o plano", [(CP, 'if headErr != nil && !errors.Is(headErr, ErrGitUnavailable) {', 'if false && headErr != nil && !errors.Is(headErr, ErrGitUnavailable) {')], P, "TestB07R2CloseoutUnknownHead"),
 ("modo executável da origem não é comparado", [(RD, 'if runtime.GOOS != "windows" && (info.Mode().Perm()&0o111 != 0) != (mode == "100755") {', 'if runtime.GOOS == "plan9" {')], C, "TestB07R2SourceMode"),
 ("limpo para sujo não é mudança", [(OP, 'if now.WorktreeDirty != snap.WorktreeDirty {', 'if false {')], P, "TestB07R2SnapshotCleanToDirty"),
 ("registro publicado e depois obsoleto não é substituído", [(RD, 'reason != "" {\n\t\tlate := published', 'reason != "" && false {\n\t\tlate := published')], C, "TestB07R2PublishWindow"), ("bundle ilegível é lido como ausente", [(RD, 'if !read.Coverage.CompleteFor(plan.Brief.Scope) {', 'if false && !read.Coverage.CompleteFor(plan.Brief.Scope) {')], P, "TestB07IndependentLostSupersedingBundle"),
 ("post-check examina gitlinks alheios", [(RD, 'changed, divergeErr := WorktreeDivergenceFor(worktree, plan.SealedCommit, IsDerivedRegeneration, func(p string) bool { return submodules[p] })', 'changed, divergeErr := WorktreeDivergence(worktree, plan.SealedCommit, IsDerivedRegeneration)')], C, "TestB07ASubmoduleTheBundleDoesNotName"),
]
def classify(r, run):
    """killed only when a test of the targeted pattern failed on an assertion; a build,
    a setup or a tool failure is an error, never a kill."""
    import re
    pattern = re.compile(run)
    failed, build_failed = False, False
    for line in (r.stdout or '').splitlines():
        try:
            event = json.loads(line)
        except ValueError:
            continue
        if event.get('Action') == 'fail' and event.get('Test') and pattern.search(event['Test']):
            failed = True
        if event.get('Action') == 'output' and ('[build failed]' in event.get('Output', '') or '[setup failed]' in event.get('Output', '')):
            build_failed = True
        if event.get('Action') == 'fail' and not event.get('Test') and event.get('Package') is None:
            build_failed = True
    if r.returncode == 0:
        return 'survived'
    if build_failed or not failed:
        return 'error'
    return 'killed'


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
        r = subprocess.run(['go', 'test', *pkgs.split(), '-run', run, '-count=1', '-json'], cwd=work, capture_output=True, text=True, timeout=900)
        verdict = classify(r, run)
        results.append({'mutation': name, 'tests': run, 'verdict': verdict})
        print(verdict, name, flush=True)
import sys
pathlib.Path(args.output).write_text(json.dumps({'mutations': results, 'killed': sum(r['verdict'] == 'killed' for r in results), 'total': len(results)}, indent=2, ensure_ascii=False) + '\n')
sys.exit(0 if all(r['verdict'] == 'killed' for r in results) else 1)
