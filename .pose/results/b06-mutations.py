"""Production mutations of B06 (spec pose-source-coverage-integrity).

Each mutation reverts one guarantee in a COPY of the repository (the working tree is
never edited), runs the named tests there and must make them fail on an assertion,
not on a build error. Run from the repository root:

    python3 -I .pose/results/b06-mutations.py --output <file.json> [--only <text>]
"""
import argparse, json, pathlib, shutil, subprocess, tempfile

parser = argparse.ArgumentParser(description=__doc__)
parser.add_argument('--output', required=True)
parser.add_argument('--only', default='', help='run only the mutations whose name contains this text')
args = parser.parse_args()
root = pathlib.Path('.').resolve()
FP = "pose-mcp/internal/pose/followups_parse.go"
FC = "pose-mcp/internal/pose/followup_candidates.go"
OP = "pose-mcp/internal/pose/obligation_projection.go"
OS = "pose-mcp/internal/pose/obligation_sources.go"
TD = "pose-mcp/internal/pose/techdebt.go"
AS = "pose-mcp/internal/pose/assessment_scan.go"
SC = "pose-mcp/internal/pose/source_coverage.go"
RC = "pose-mcp/internal/pose/review_closeout.go"
RB = "pose-mcp/internal/pose/review_bundle.go"
CF = "pose-mcp/internal/cli/followups.go"
CS = "pose-mcp/internal/cli/state_providers.go"
CR = "pose-mcp/internal/cli/review_closeout.go"

M = [
 ("spec ilegível ignorado", [(FP, 'coverage.Add(SourceLoss{Producer: "followups", Artifact: filepath.ToSlash(rel), Scope: specSlugFromPath(path), AttributedBy: "path", Kind: LossReadError, Detail: err.Error()})', '_ = rel')], "./internal/pose", "TestAnUnreadableSpecIsALoss|TestTheFollowupProducerIsIncomplete"),
 ("não resolvido só o [open]", [(FP, 'return f.RawDisposition == "" || f.RawDisposition == "open"', 'return f.RawDisposition == "open"')], "./internal/pose", "TestUnresolvedFollowupsAreOneSet"),
 ("perda de outro escopo alarga este", [(SC, 'if loss.Scope == "" || loss.Scope == scope {', 'if true {')], "./internal/pose", "TestSourceCoverageScoping|TestAReviewRecordThatNamesAnotherScope"),
 ("perda sem escopo ignorada", [(SC, 'if loss.Scope == "" || loss.Scope == scope {', 'if loss.Scope == scope {')], "./internal/pose", "TestSourceCoverageScoping"),
 ("produtor de follow-ups sem cobertura", [(OP, 'if len(losses) > 0 && coverage != nil {', 'if false && len(losses) > 0 && coverage != nil {')], "./internal/pose", "TestTheFollowupProducerIsIncomplete"),
 ("produtor de follow-ups sem recorte", [(OP, 'if scope == "" || loss.Scope == "" || inScope[loss.Scope] {', 'if true {')], "./internal/pose", "TestTheFollowupProducerIsIncomplete"),
 ("obrigação descartada sem cobertura", [(OP, 'if cov := report.producerState[o.Source.Producer]; cov != nil {', 'if cov := report.producerState[o.Source.Producer]; cov != nil && false {')], "./internal/pose", "TestADroppedObligation"),
 ("sem disposição sem razão própria", [(OP, 'o.ReasonCode = "followup-undisposed"', 'o.ReasonCode = "followup-open"')], "./internal/pose", "TestUnresolvedFollowupsAreOneSet"),
 ("candidatos só do [open]", [(FC, 'if !item.Unresolved() {', 'if item.RawDisposition != "open" {')], "./internal/pose", "TestUnresolvedFollowupsAreOneSet"),
 ("scan com limite de linha", [(TD, 'line = strings.TrimSuffix(line, "\\r")', 'if len(line) > 65536 {\n\t\t\tbreak\n\t\t}\n\t\tline = strings.TrimSuffix(line, "\\r")')], "./internal/pose", "TestAMarkerAfterAVeryLongLineIsFound"),
 ("arquivo acima do limite sem exclusão", [(AS, 'coverage.addExclusion("assessment-scan", "over-size-limit", fmt.Sprintf("files larger than %d bytes are not scanned", maxAssessmentFileSize), filepath.ToSlash(rel))', '_ = rel')], "./internal/pose", "TestTechDebtCoverageNamesWhatItDidNotScan"),
 ("arquivo ilegível sem perda", [(AS, 'raw, err := os.ReadFile(path)\n\t\tif err != nil {\n\t\t\tlost(path, LossReadError, "not scanned: "+err.Error())\n\t\t\treturn nil\n\t\t}', 'raw, err := os.ReadFile(path)\n\t\tif err != nil {\n\t\t\treturn nil\n\t\t}')], "./internal/pose", "TestTechDebtCoverageNamesWhatItDidNotScan"),
 ("scan completo apesar da perda", [(TD, 'ScannedFiles: scanned, Complete: coverage.Complete(), Coverage: coverage,', 'ScannedFiles: scanned, Complete: true, Coverage: coverage,')], "./internal/pose", "TestTechDebtCoverageNamesWhatItDidNotScan"),
 ("relatório sem escopo do scan", [(TD, 'techDebtScopeMarkdown(report))', '"")')], "./internal/pose", "TestTheTechDebtReportSaysWhatItCovered"),
 ("tentativa ilegível ignorada", [(RC, 'coverage.Add(SourceLoss{Producer: "review-attempts", Artifact: rel, Kind: LossReadError, Detail: readErr.Error(), At: at})', '_ = rel')], "./internal/pose", "TestAnUnreadableNewerAttemptBlocks|TestWithNoReadableAttempt"),
 ("registro mais novo tratado como antigo", [(RC, 'if loss.At != "" && at != "" && loss.At <= at {', 'if loss.At != "" && at != "" && loss.At >= at {')], "./internal/pose", "TestAnUnreadableNewerAttemptBlocks|TestAnUnreadableOlderAttemptOnlyWarns"),
 ("registro mais antigo bloqueia", [(RC, 'if loss.At != "" && at != "" && loss.At <= at {', 'if false {')], "./internal/pose", "TestAnUnreadableOlderAttemptOnlyWarns"),
 ("histórico incompleto não bloqueia", [(RC, 'if lost := attemptsRead.HistoryUncertainAfter(ref, current.ReviewedAt); len(lost) > 0 {', 'if lost := attemptsRead.HistoryUncertainAfter(ref, current.ReviewedAt); false && len(lost) > 0 {')], "./internal/pose", "TestAnUnreadableNewerAttemptBlocks"),
 ("sem tentativa legível vira ausência", [(RC, 'if lost := attemptsRead.HistoryUncertainAfter(ref, ""); len(lost) > 0 {', 'if lost := attemptsRead.HistoryUncertainAfter(ref, ""); false && len(lost) > 0 {')], "./internal/pose", "TestWithNoReadableAttempt"),
 ("registro de outro escopo é perda", [(RC, 'if fm["scope"] != scope {\n\t\t\t\tif fm["scope"] == "" {', 'if fm["scope"] == scope || true {\n\t\t\t\tif true {')], "./internal/pose", "TestAReviewRecordThatNamesAnotherScope"),
 ("atribuição por cabeçalho removida", [(RC, 'loss.Scope, loss.AttributedBy = fm["scope"], "header"', '_ = fm')], "./internal/pose", "TestAnUnscopedUnreadableAttemptIsAttributed"),
 ("formato não suportado como corrupção", [(RC, 'loss.Kind = LossUnsupported', '_ = loss')], "./internal/pose", "TestAnUnscopedUnreadableAttemptIsAttributed"),
 ("atestação ilegível sem atribuição", [(RB, 'loss := SourceLoss{Producer: "review-attestations", Artifact: rel, Kind: LossParseError, Detail: "the record is not valid JSON"}\n\t\t\t\tif m := attestationBundleSniffRE.FindSubmatch(raw); m != nil {', 'loss := SourceLoss{Producer: "review-attestations", Artifact: rel, Kind: LossParseError, Detail: "the record is not valid JSON"}\n\t\t\t\tif m := attestationBundleSniffRE.FindSubmatch(raw); m != nil && false {')], "./internal/pose", "TestAnUnreadableAttestationIsNotSkipped|TestAnUnreadableLaterRejection"),
 ("histórico de atestações sem bloqueio", [(RB, 'if len(attestationHistory) > 0 {\n\t\tverification.Blockers = append(verification.Blockers, "attestation history for bundle "', 'if false {\n\t\tverification.Blockers = append(verification.Blockers, "attestation history for bundle "')], "./internal/pose", "TestAnUnreadableLaterRejection"),
 ("atestação ilegível some", [(RB, 'coverage.Add(SourceLoss{Producer: "review-attestations", Artifact: rel, Kind: LossReadError, Detail: err.Error()})', '_ = rel')], "./internal/pose", "TestAnUnreadableLaterRejection"),
 ("bundle ilegível sem aviso", [(RB, 'if bundleLosses := bundlesRead.Coverage.LossesFor(scope); len(bundleLosses) > 0 {', 'if bundleLosses := bundlesRead.Coverage.LossesFor(scope); false && len(bundleLosses) > 0 {')], "./internal/pose", "TestAnUnreadableBundleIsDisclosed"),
 ("produtor de findings sem cobertura", [(OS, 'markCoverageIncomplete(coverage, read.Coverage.Describe(""))', '_ = coverage')], "./internal/pose", "TestFindingsProducerIsIncomplete"),
 ("--fail-overdue certifica leitura incompleta", [(CF, 'if failOverdue && (len(overdueEntries) > 0 || !coverage.Complete()) {', 'if failOverdue && len(overdueEntries) > 0 {'), (CF, 'if failOverdue && !coverage.Complete() {', 'if false {')], "./internal/cli", "TestFailOverdueRefusesToCertify"),
 ("cabeçalho humano sem cobertura", [(CF, 'if !coverage.Complete() {\n\t\t// What the read could not see is part of the result', 'if false {\n\t\t// What the read could not see is part of the result')], "./internal/cli", "TestFollowupsReportsASpecItCouldNotRead"),
 ("estado do projeto só conta [open]", [(CS, 'if !followupUnresolved(f) {', 'if f.RawDisposition != "open" {')], "./internal/cli", "TestTheBacklogAndTheProjectState"),
 ("estado do projeto sem cobertura", [(CS, 'if !coverage.Complete() {\n\t\tlines = append(lines, "- cobertura: INCOMPLETA', 'if false {\n\t\tlines = append(lines, "- cobertura: INCOMPLETA')], "./internal/cli", "TestFollowupsReportsASpecItCouldNotRead"),
 ("candidatos afirmam cobertura completa", [(CF, '"coverage_complete": result.Coverage.Complete(), "coverage": result.Coverage', '"coverage_complete": true, "coverage": result.Coverage')], "./internal/cli", "TestFollowupsReportsASpecItCouldNotRead"),
 ("backlog JSON afirma cobertura completa", [(CF, '"coverage_complete": coverage.Complete(), "coverage": coverage,', '"coverage_complete": true, "coverage": coverage,')], "./internal/cli", "TestFollowupsReportsASpecItCouldNotRead|TestFollowupParity"),
 ("CLI do tech-debt sem escopo", [("pose-mcp/internal/cli/assess.go", 'scope.Field("techdebt.scan", fmt.Sprintf("%d source file(s) scanned, none unreadable", report.ScannedFiles))', '_ = scope')], "./internal/cli", "TestAssessTechDebtStatesItsScope"),
 ("review record escolhe de histórico com buraco", [(CR, 'if lost := attemptsRead.HistoryUncertainAfter(ref, lastAt); err == nil && len(lost) > 0 {', 'if lost := attemptsRead.HistoryUncertainAfter(ref, lastAt); false && err == nil && len(lost) > 0 {')], "./internal/cli", "TestReviewRecordRefusesToSupersede"),
 ("Attention não projeta sem disposição", [(OP, 'if !item.Unresolved() {\n\t\t\tcontinue\n\t\t}\n\t\tif scope != "" && !inScope[item.Spec] {', 'if item.RawDisposition != "open" {\n\t\t\tcontinue\n\t\t}\n\t\tif scope != "" && !inScope[item.Spec] {')], "./internal/mcpserver ./internal/pose", "TestFollowupParityAcrossCLIMCPAndAttention|TestUnresolvedFollowupsAreOneSet"),
]
results = []
for name, edits, pkgs, run in M:
    if args.only and args.only not in name:
        continue
    with tempfile.TemporaryDirectory(prefix="b06-mut-") as tmp:
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
