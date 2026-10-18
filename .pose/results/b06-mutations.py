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
DS = "pose-mcp/internal/pose/discovery.go"
LL = "pose-mcp/internal/pose/legacy_ledger.go"
GW = "pose-mcp/internal/pose/governance_waits.go"
CA = "pose-mcp/internal/cli/state_attention.go"
RC = "pose-mcp/internal/pose/review_closeout.go"
RB = "pose-mcp/internal/pose/review_bundle.go"
CF = "pose-mcp/internal/cli/followups.go"
CS = "pose-mcp/internal/cli/state_providers.go"
CR = "pose-mcp/internal/cli/review_closeout.go"

M = [
 ("spec ilegível ignorado", [(FP, 'raw, readErr := os.ReadFile(path)\n\t\t\tif readErr != nil {\n\t\t\t\tlose(path, slug, LossReadError, readErr.Error())\n\t\t\t\tcontinue', 'raw, readErr := os.ReadFile(path)\n\t\t\tif readErr != nil {\n\t\t\t\t_ = slug\n\t\t\t\tcontinue')], "./internal/pose", "TestAnUnreadableSpecIsALoss|TestTheFollowupProducerIsIncomplete"),
 ("não resolvido só o [open]", [(FP, 'return disposition == "" || disposition == "open" || !KnownFollowupDispositions[disposition]', 'return disposition == "open"')], "./internal/pose", "TestUnresolvedFollowupsAreOneSet"),
 ("perda de outro escopo alarga este", [(SC, 'if loss.Scope == "" || loss.Scope == scope {', 'if true {')], "./internal/pose", "TestSourceCoverageScoping|TestAReviewRecordThatNamesAnotherScope"),
 ("perda sem escopo ignorada", [(SC, 'if loss.Scope == "" || loss.Scope == scope {', 'if loss.Scope == scope {')], "./internal/pose", "TestSourceCoverageScoping"),
 ("produtor de follow-ups sem cobertura", [(OP, 'if len(losses) > 0 && coverage != nil {', 'if false && len(losses) > 0 && coverage != nil {')], "./internal/pose", "TestTheFollowupProducerIsIncomplete"),
 ("produtor de follow-ups sem recorte", [(OP, 'if scope == "" || loss.Scope == "" || inScope[loss.Scope] {', 'if true {')], "./internal/pose", "TestTheFollowupProducerIsIncomplete"),
 ("obrigação descartada sem cobertura", [(OP, 'if cov := report.producerState[slot]; cov != nil {', 'if cov := report.producerState[slot]; cov != nil && false {')], "./internal/pose", "TestADroppedObligation"),
 ("sem disposição sem razão própria", [(OP, 'o.ReasonCode = "followup-undisposed"', 'o.ReasonCode = "followup-open"')], "./internal/pose", "TestUnresolvedFollowupsAreOneSet"),
 ("candidatos só do [open]", [(FC, 'if !item.Unresolved() {', 'if item.RawDisposition != "open" {')], "./internal/pose", "TestUnresolvedFollowupsAreOneSet"),
 ("scan com limite de linha", [(TD, 'line = strings.TrimSuffix(line, "\\r")', 'if len(line) > 65536 {\n\t\t\tbreak\n\t\t}\n\t\tline = strings.TrimSuffix(line, "\\r")')], "./internal/pose", "TestAMarkerAfterAVeryLongLineIsFound"),
 ("arquivo acima do limite sem exclusão", [(AS, 'coverage.addExclusion("assessment-scan", "over-size-limit", fmt.Sprintf("files larger than %d bytes are not scanned", maxAssessmentFileSize), filepath.ToSlash(rel))', '_ = rel')], "./internal/pose", "TestTechDebtCoverageNamesWhatItDidNotScan"),
 ("arquivo ilegível sem perda", [(AS, 'raw, err := os.ReadFile(path)\n\t\tif err != nil {\n\t\t\tlost(path, LossReadError, "not scanned: "+err.Error())\n\t\t\treturn nil\n\t\t}', 'raw, err := os.ReadFile(path)\n\t\tif err != nil {\n\t\t\treturn nil\n\t\t}')], "./internal/pose", "TestTechDebtCoverageNamesWhatItDidNotScan"),
 ("scan completo apesar da perda", [(TD, 'ScannedFiles: scanned, Complete: coverage.Complete(), Coverage: coverage,', 'ScannedFiles: scanned, Complete: true, Coverage: coverage,')], "./internal/pose", "TestTechDebtCoverageNamesWhatItDidNotScan"),
 ("relatório sem escopo do scan", [(TD, 'techDebtScopeMarkdown(report))', '"")')], "./internal/pose", "TestTheTechDebtReportSaysWhatItCovered"),
 ("tentativa ilegível ignorada", [(RC, 'coverage.Add(SourceLoss{Producer: "review-attempts", Artifact: rel, Kind: LossReadError, Detail: readErr.Error(), At: at, ID: strings.TrimSuffix(entry.Name(), ".md")})', '_ = rel')], "./internal/pose", "TestAnUnreadableNewerAttemptBlocks|TestWithNoReadableAttempt"),
 ("registro mais novo tratado como antigo", [(RC, 'if lossInstant.Before(selected) ||', 'if lossInstant.After(selected) ||')], "./internal/pose", "TestAnUnreadableNewerAttemptBlocks|TestAnUnreadableOlderAttemptOnlyWarns"),
 ("registro mais antigo bloqueia", [(RC, 'if lossInstant.Before(selected) ||', 'if false ||')], "./internal/pose", "TestAnUnreadableOlderAttemptOnlyWarns"),
 ("histórico incompleto não bloqueia", [(RC, 'if lost := attemptsRead.HistoryUncertainAfter(ref, current.ReviewedAt, current.ReviewID); len(lost) > 0 {', 'if lost := attemptsRead.HistoryUncertainAfter(ref, current.ReviewedAt, current.ReviewID); false && len(lost) > 0 {')], "./internal/pose", "TestAnUnreadableNewerAttemptBlocks"),
 ("sem tentativa legível vira ausência", [(RC, 'if lost := attemptsRead.HistoryUncertainAfter(ref, "", ""); len(lost) > 0 {', 'if lost := attemptsRead.HistoryUncertainAfter(ref, "", ""); false && len(lost) > 0 {')], "./internal/pose", "TestWithNoReadableAttempt"),
 ("registro de outro escopo é perda", [(RC, 'if fm["scope"] != scope {\n\t\t\t\tif fm["scope"] == "" {', 'if fm["scope"] == scope || true {\n\t\t\t\tif true {')], "./internal/pose", "TestAReviewRecordThatNamesAnotherScope"),
 ("atribuição por cabeçalho removida", [(RC, 'loss.Scope, loss.AttributedBy = fm["scope"], "header"', '_ = fm')], "./internal/pose", "TestAnUnscopedUnreadableAttemptIsAttributed"),
 ("formato não suportado como corrupção", [(RC, 'loss.Kind = LossUnsupported', '_ = loss')], "./internal/pose", "TestAnUnscopedUnreadableAttemptIsAttributed"),
 ("atestação ilegível sem atribuição", [(RB, 'found && !ambiguous && value != ""', 'false && found && !ambiguous && value != ""')], "./internal/pose", "TestAnUnreadableAttestationIsNotSkipped|TestAnUnreadableLaterRejection"),
 ("histórico de atestações sem bloqueio", [(RB, 'if len(attestationHistory) > 0 {\n\t\tverification.Blockers = append(verification.Blockers, "attestation history for bundle "', 'if false {\n\t\tverification.Blockers = append(verification.Blockers, "attestation history for bundle "')], "./internal/pose", "TestAnUnreadableLaterRejection"),
 ("atestação ilegível some", [(RB, 'coverage.Add(SourceLoss{Producer: "review-attestations", Artifact: rel, Kind: LossReadError, Detail: err.Error()})', '_ = rel')], "./internal/pose", "TestAnUnreadableLaterRejection"),
 ("bundle ilegível sem aviso", [(RB, 'if bundleLosses := bundlesRead.Coverage.LossesFor(scope); len(bundleLosses) > 0 {', 'if bundleLosses := bundlesRead.Coverage.LossesFor(scope); false && len(bundleLosses) > 0 {')], "./internal/pose", "TestAnUnreadableBundleIsDisclosed"),
 ("produtor de findings sem cobertura", [(OS, 'if len(lost) > 0 && coverage != nil {', 'if false && len(lost) > 0 && coverage != nil {')], "./internal/pose", "TestFindingsProducerIsIncomplete"),
 ("--fail-overdue certifica leitura incompleta", [(CF, 'if failOverdue && (len(overdueEntries) > 0 || !coverage.Complete()) {', 'if failOverdue && len(overdueEntries) > 0 {'), (CF, 'if failOverdue && !coverage.Complete() {', 'if false {')], "./internal/cli", "TestFailOverdueRefusesToCertify"),
 ("cabeçalho humano sem cobertura", [(CF, 'if !coverage.Complete() {\n\t\t// What the read could not see is part of the result', 'if false {\n\t\t// What the read could not see is part of the result')], "./internal/cli", "TestFollowupsReportsASpecItCouldNotRead"),
 ("estado do projeto só conta [open]", [(CS, 'if !followupUnresolved(f) {', 'if f.RawDisposition != "open" {')], "./internal/cli", "TestTheBacklogAndTheProjectState"),
 ("estado do projeto sem cobertura", [(CS, 'if !coverage.Complete() {\n\t\tlines = append(lines, "- cobertura: INCOMPLETA', 'if false {\n\t\tlines = append(lines, "- cobertura: INCOMPLETA')], "./internal/cli", "TestFollowupsReportsASpecItCouldNotRead"),
 ("candidatos afirmam cobertura completa", [(CF, '"coverage_complete": result.Coverage.Complete(), "coverage": result.Coverage', '"coverage_complete": true, "coverage": result.Coverage')], "./internal/cli", "TestFollowupsReportsASpecItCouldNotRead"),
 ("backlog JSON afirma cobertura completa", [(CF, '"coverage_complete": coverage.Complete(), "coverage": coverage,', '"coverage_complete": true, "coverage": coverage,')], "./internal/cli", "TestFollowupsReportsASpecItCouldNotRead|TestFollowupParity"),
 ("CLI do tech-debt sem escopo", [("pose-mcp/internal/cli/assess.go", 'scope.Field("techdebt.scan", fmt.Sprintf("%d source file(s) scanned, none unreadable", report.ScannedFiles))', '_ = scope')], "./internal/cli", "TestAssessTechDebtStatesItsScope"),
 ("review record escolhe de histórico com buraco", [(CR, 'if lost := attemptsRead.HistoryUncertainAfter(ref, lastAt, lastID); err == nil && len(lost) > 0 {', 'if lost := attemptsRead.HistoryUncertainAfter(ref, lastAt, lastID); false && err == nil && len(lost) > 0 {')], "./internal/cli", "TestReviewRecordRefusesToSupersede"),
 ("Attention não projeta sem disposição", [(OP, 'if !item.Unresolved() {\n\t\t\tcontinue\n\t\t}\n\t\tif scope != "" && !inScope[item.Spec] {', 'if item.RawDisposition != "open" {\n\t\t\tcontinue\n\t\t}\n\t\tif scope != "" && !inScope[item.Spec] {')], "./internal/mcpserver ./internal/pose", "TestFollowupParityAcrossCLIMCPAndAttention|TestUnresolvedFollowupsAreOneSet"),

 ("aprovação retida sobre rejeição ilegível", [(RC, 'if lost := read.Coverage.LossesFor(bundles[i].BundleID); len(lost) > 0 {\n\t\t\treturn nil, nil, lost\n\t\t}', 'if lost := read.Coverage.LossesFor(bundles[i].BundleID); len(lost) > 0 && false {\n\t\t\treturn nil, nil, lost\n\t\t}')], "./internal/pose", "TestAClosedScopeDoesNotRetain"),
 ("retenção sem bloqueio no review-check", [(RC, 'if len(retainLost) > 0 {\n\t\t\t\teval.Blockers = append(eval.Blockers, "the approval of "', 'if false {\n\t\t\t\teval.Blockers = append(eval.Blockers, "the approval of "')], "./internal/pose", "TestAClosedScopeDoesNotRetain"),
 ("sniff ignora o caminho estrutural", [(SC, 'if !f.object || !strings.EqualFold(f.key, path[i]) {', 'if !f.object || (!strings.EqualFold(f.key, path[i]) && false) {')], "./internal/pose", "TestSniffingAttributesOnlyTheStructuralPath|TestADecoyInATruncatedRecord"),
 ("JSON sem cabeçalho tratado como outro bundle", [(RB, 'if header.BundleID == "" {', 'if false {')], "./internal/pose", "TestValidJSONWithoutAHeader"),
 ("bundle sem escopo tratado como outro escopo", [(RB, 'if header.Payload.Scope.Ref == "" {', 'if false {')], "./internal/pose", "TestValidJSONWithoutAHeader"),
 ("mesmo segundo é anterior", [(RC, '(lossInstant.Equal(selected) && loss.ID != "" && id != "" && loss.ID < id)', 'lossInstant.Equal(selected)')], "./internal/pose", "TestAnUnreadableAttemptOfTheSameSecond"),
 ("tentativa antiga malformada falha a leitura", [(RC, 'loss := SourceLoss{Producer: "review-attempts", Artifact: rel, Kind: LossParseError, Detail: err.Error(), At: at, ID: strings.TrimSuffix(entry.Name(), ".md")}', 'if scope != "" {\n\t\t\t\treturn ReviewAttemptsRead{}, err\n\t\t\t}\n\t\t\tloss := SourceLoss{Producer: "review-attempts", Artifact: rel, Kind: LossParseError, Detail: err.Error(), At: at, ID: strings.TrimSuffix(entry.Name(), ".md")}')], "./internal/pose", "TestAnOldMalformedAttemptOfTheScopeOnlyWarns"),
 ("registro sem campos obrigatórios aceito", [(RC, 'if strings.TrimSpace(value) == "" {\n\t\t\t\t\tmissing = append(missing, name)', 'if strings.TrimSpace(value) == "" && false {\n\t\t\t\t\tmissing = append(missing, name)')], "./internal/pose", "TestARecordWithoutItsRequiredFields"),
 ("findings sem recorte de escopo", [(OS, 'concerns := scope == "" || loss.Scope == ""', 'concerns := true')], "./internal/pose", "TestAFindingsLossInAnotherScope"),
 ("discovery com limite de linha", [(DS, 'line := strings.TrimSpace(rawLine)', 'if len(rawLine) > 65536 {\n\t\t\tbreak\n\t\t}\n\t\tline := strings.TrimSpace(rawLine)')], "./internal/pose", "TestComponentDiscoveryFindsAMarker"),
 ("discovery verified apesar da perda", [(DS, 'if score >= 0.95 && state.ScanComplete {', 'if score >= 0.95 {')], "./internal/pose", "TestComponentDiscoveryFindsAMarker"),
 ("discovery ignora arquivo ilegível", [(DS, 'coverage.Add(SourceLoss{Producer: "component-discovery", Artifact: relOf(path), Kind: LossReadError, Detail: "not scanned: " + inspectErr.Error()})', '_ = relOf')], "./internal/pose", "TestComponentDiscoveryFindsAMarker"),
 ("diretório pulado sem exclusão", [(AS, 'coverage.addExclusion("assessment-scan", "skipped-directory", "directories that hold dependencies, build output or tool state are not scanned", entry.Name())', '_ = entry')], "./internal/pose", "TestSkippedDirectoriesAndLinks"),
 ("symlink sem exclusão", [(AS, 'coverage.addExclusion("assessment-scan", "symlink", "links are not followed, so their targets are scanned only where they live", filepath.ToSlash(rel))', '_ = rel')], "./internal/pose", "TestSkippedDirectoriesAndLinks"),
 ("layout split perdido", [(FP, 'if len(files) > 0 {\n\t\t\t\tbody, readErr := readSplitSpecBody(files)', 'if len(files) > 0 && false {\n\t\t\t\tbody, readErr := readSplitSpecBody(files)')], "./internal/pose", "TestTheLegacySplitLayout"),
 ("frontmatter truncado certificado", [(FP, 'lose(src.path, src.slug, LossParseError, "the frontmatter is not closed")', '_ = src')], "./internal/pose", "TestATruncatedHeaderIsALoss"),
 ("disposição desconhecida resolve", [(FP, 'return disposition == "" || disposition == "open" || !KnownFollowupDispositions[disposition]', 'return disposition == "" || disposition == "open"')], "./internal/pose", "TestATruncatedHeaderIsALoss|TestAttentionCarriesTheFollowup"),
 ("Attention sem diagnóstico de metadata", [(OP, 'o.Observation.Limitations = append(o.Observation.Limitations, "ownership metadata is malformed: "+item.MetaErr)', '_ = item')], "./internal/pose", "TestAttentionCarriesTheFollowup"),
 ("Attention humano sem detalhe", [(CA, 'if c.State != pose.CoverageStateCurrent && c.Detail != "" {', 'if false {')], "./internal/cli", "TestAttentionTextNamesTheUnreadArtifact"),
 ("candidatos ignoram --fail-overdue", [(CF, 'if failOverdue {\n\t\t\t// The gate belongs to the backlog', 'if false {\n\t\t\t// The gate belongs to the backlog')], "./internal/cli", "TestCandidatesRefuseTheOverdueGate"),
 ("classificação de dívida sem perda de backlog", [(TD, '	for _, loss := range documentLosses {\n		coverage.Add(loss)\n	}', '	_ = documentLosses')], "./internal/pose", "TestTheDebtClassificationSaysWhen"),
 ("rework sem cobertura", [(GW, 'report.Complete = report.Coverage.Complete()', 'report.Complete = true')], "./internal/pose", "TestGovernanceConsumersDisclose"),
 ("inventário sem cobertura", [("pose-mcp/internal/pose/legacy_inventory.go", 'inv.Complete = inv.Coverage.Complete()', 'inv.Complete = true')], "./internal/pose", "TestGovernanceConsumersDisclose"),
 ("governança efetiva sem histórico incompleto", [("pose-mcp/internal/pose/effective_governance.go", 'ctx.HistoryIncomplete = bundlesRead.Coverage.Describe(scope)', '_ = bundlesRead')], "./internal/pose", "TestGovernanceConsumersDisclose"),
 ("ledger ilegível como ausência", [(LL, 'lose(LossParseError, "the ledger is not valid JSON")\n\t\t\tcontinue', 'continue')], "./internal/pose", "TestAnUnreadableLegacyLedger"),
 ("ledger de versão desconhecida como ausência", [(LL, 'lose(LossUnsupported, "not a legacy attestations ledger of a version this engine reads")\n\t\t\tcontinue', 'continue')], "./internal/pose", "TestAnUnreadableLegacyLedger"),
 ("produtor vazio escapa da cobertura", [(OP, 'slot := o.producerSlot\n\t\t\tif slot == "" {\n\t\t\t\tslot = o.Source.Producer\n\t\t\t}', 'slot := o.Source.Producer')], "./internal/pose", "TestAnInvalidElementWithoutAProducer"),

 ("R2 retenção ignora erro do loader", [(RC, 'return nil, nil, []SourceLoss{{Producer: "review-attestations", Artifact: bundles[i].BundleID, Scope: ref, AttributedBy: "bundle", Kind: LossInvalid, Detail: attErr.Error()}}', 'continue')], "./internal/pose", "TestARejectionThatFailsItsDigest"),
 ("R2 cabeçalho repetido filtrado como outro bundle", [(RB, 'if _, _, ambiguous := sniffJSONString(raw, "bundle_id"); ambiguous {', 'if false {')], "./internal/pose", "TestARepeatedHeaderNameInValidJSON"),
 ("R2 escopo repetido filtrado", [(RB, 'if _, _, ambiguous := sniffJSONString(raw, "payload", "scope", "ref"); ambiguous {', 'if false {')], "./internal/pose", "TestARepeatedHeaderNameInValidJSON"),
 ("R2 sniff só vê duplicata no valor", [(SC, 'if seen[n-1] > 1 {', 'if seen[n-1] > 1 && false {')], "./internal/pose", "TestSniffingSeesARepeatedName"),
 ("R2 seção split quebrada é ausência", [(FP, 'lose(section, slug, LossReadError, serr.Error())', '_ = slug')], "./internal/pose", "TestAnUnresolvableSpecEntry"),
 ("R2 perda do próprio milestone ignorada", [(OS, 'concerns := scope == "" || loss.Scope == "" || loss.Scope == scope', 'concerns := scope == "" || loss.Scope == ""')], "./internal/pose", "TestAFindingsLossOfTheRequestedMilestone"),
 ("R2 consolidado certifica scan perdido", [(DS, 'if st.ScanIncomplete() {\n\t\t\tincomplete = append(incomplete, st.ComponentSlug)\n\t\t}', '_ = incomplete')], "./internal/pose", "TestTheConsolidatedAssessmentDeclares"),
 ("R2 project state certifica scan perdido", [("pose-mcp/internal/cli/assess.go", 'if partial > 0 || unreadable > 0 {', 'if false {')], "./internal/cli", "TestTheProjectStateDoesNotCertify"),
 ("R2 diagnóstico não chega ao backlog humano", [(CF, 'if entry.DispositionErr != "" {\n\t\t\trender(stdout, stderr)', 'if false {\n\t\t\trender(stdout, stderr)')], "./internal/cli", "TestHumanOutputsCarryTheParserDiagnostics"),
 ("R2 Attention humano sem limites", [(CA, 'if len(o.Observation.Limitations) > 0 {\n\t\tline +=', 'if false {\n\t\tline +=')], "./internal/cli", "TestHumanOutputsCarryTheParserDiagnostics"),
 ("R2 roadmap ilegível some da reconciliação", [(TD, 'lose(".pose/roadmaps/"+name, LossReadError, statErr.Error())', '_ = statErr')], "./internal/pose", "TestTheDebtClassificationSaysWhenARoadmap"),
 ("R2 project state ilegível ignorado", [(TD, 'lose(".pose/state/project-state.md", LossReadError, err.Error())', '_ = err')], "./internal/pose", "TestTheDebtClassificationSaysWhenARoadmap"),
 ("R2 diretório de ledger ilegível some", [(LL, 'coverage.Add(SourceLoss{Producer: "legacy-ledger", Artifact: legacyLedgerDir, Kind: LossReadError, Detail: err.Error()})', '_ = err')], "./internal/pose", "TestAnUnreadableLedgerDirectory"),
 ("R2 decisão desconhecida aceita", [(RC, '} else if !knownReviewDecisions[a.Decision] {', '} else if false {')], "./internal/pose", "TestAReviewRecordWithInvalidValues"),
 ("R2 timestamp inválido aceito", [(RC, '} else if _, timeErr := time.Parse(time.RFC3339, a.ReviewedAt); timeErr != nil {', '} else if _, timeErr := time.Parse(time.RFC3339, a.ReviewedAt); timeErr != nil && false {')], "./internal/pose", "TestAReviewRecordWithInvalidValues"),
 ("R2 cabeçalho aberto aceito", [(RC, '!frontmatterClosed(string(raw)) {\n\t\t\t\terr = fmt.Errorf("pose: review record header is not closed")', '!frontmatterClosed(string(raw)) && false {\n\t\t\t\terr = fmt.Errorf("pose: review record header is not closed")')], "./internal/pose", "TestAReviewRecordWithInvalidValues"),
 ("R2 ledger planejado sobre histórico incompleto", [(LL, 'if !read.Coverage.Complete() {\n\t\treturn nil, fmt.Errorf("pose: the attestation history cannot be read completely', 'if false {\n\t\treturn nil, fmt.Errorf("pose: the attestation history cannot be read completely')], "./internal/pose", "TestALegacyLedgerIsNotPlannedOverAnIncomplete"),
 ("R2 spec.md quebrado ignorado", [(FP, 'lose(specPath, slug, LossReadError, readErr.Error())\n\t\t\t\t\tcontinue', 'continue')], "./internal/pose", "TestAnUnresolvableSpecEntry"),
 ("R2 raiz de specs quebrada é ausência", [(FP, '!os.IsNotExist(err) || lerr == nil {', '!os.IsNotExist(err) || (lerr == nil && false) {')], "./internal/pose", "TestAnUnresolvableSpecEntry"),
 ("R2 symlink de diretório sem exclusão", [(AS, 'coverage.addExclusion("assessment-scan", "symlink", "links are not followed, so their targets are scanned only where they live", filepath.ToSlash(rel))\n\t\t\treturn nil', '_ = rel\n\t\t\treturn nil')], "./internal/pose", "TestADirectorySymlinkIsADeclared"),
 ("R3 nomes sem distinção de caixa", [(SC, 'strings.EqualFold(t, path[n-1]) && ancestorsOK(n-1)', 't == path[n-1] && ancestorsOK(n-1)'), (SC, 'if found {\n\t\t\t\t\treturn "", false, true\n\t\t\t\t}', 'if found {\n\t\t\t\t\tcontinue\n\t\t\t\t}')], "./internal/pose", "TestCaseFoldedHeaderNames"),
 ("R3 retenção ignora bundle posterior perdido", [(RC, 'if lost := bundlesRead.Coverage.LossesFor(ref); len(lost) > 0 {', 'if lost := bundlesRead.Coverage.LossesFor(ref); len(lost) > 0 && false {')], "./internal/pose", "TestALostLaterBundleBlocks"),
 ("R3 perda comparada como texto", [(RC, 'if lossInstant.Before(selected) ||', 'if loss.At < at ||')], "./internal/pose", "TestReviewTimesAreComparedAsInstants"),
 ("R3 ordenação por texto", [(RC, 'return reviewInstantLess(attempts[i].ReviewedAt, attempts[i].ReviewID, attempts[j].ReviewedAt, attempts[j].ReviewID)', 'return attempts[i].ReviewedAt < attempts[j].ReviewedAt')], "./internal/pose", "TestAttemptsAreReadInInstantOrder"),
 ("R3 linha truncada vira lista vazia", [(RC, 'if section != "" && strings.HasPrefix(line, "-") {', 'if false && section != "" && strings.HasPrefix(line, "-") {')], "./internal/pose", "TestATruncatedFindingLine"),
 ("R3 link de pasta de spec perdido some", [(FP, 'if _, statErr := os.Stat(path); statErr != nil {\n\t\t\t\tlose(path, specSlugFromPath(path+"/spec.md"), LossReadError, statErr.Error())\n\t\t\t}', '_ = path')], "./internal/pose", "TestADanglingSpecFolderLink"),
 ("R3 raiz de reviews quebrada é vazia", [(RC, 'if _, lerr := os.Lstat(dir); lerr == nil {', 'if _, lerr := os.Lstat(dir); lerr == nil && false {')], "./internal/pose", "TestADanglingReviewsRoot"),
 ("R3 raiz de roadmaps quebrada é ausência", [(TD, '!errors.Is(dirErr, os.ErrNotExist) || lerr == nil {', '!errors.Is(dirErr, os.ErrNotExist) || (lerr == nil && false) {')], "./internal/pose", "TestTheDebtClassificationDoesNotTakeALostRoot"),
 ("R3 state inacessível é ausência", [(TD, 'lerr == nil || !errors.Is(lerr, os.ErrNotExist) {', 'lerr == nil {')], "./internal/pose", "TestTheDebtClassificationDoesNotTakeALostRoot"),
 ("R3 project state publica dívida de relatório incompleto", [("pose-mcp/internal/cli/assess.go", 'if !report.Complete && len(report.Coverage.Losses) > 0 {', 'if false {')], "./internal/cli", "TestTheProjectStateKeepsTheDebt"),
 ("R3 roadmap não herda milestone", [(OS, 'concerns = concerns || strings.HasPrefix(loss.Scope, "milestone:"+roadmap+"/")', '_ = roadmap')], "./internal/pose", "TestARoadmapQueryInherits"),
 ("R3 aviso corta a tabela", [(DS, '	if len(incomplete) > 0 {\n		sort.Strings(incomplete)\n		md += fmt.Sprintf("\\n> **Varredura incompleta**', '	if len(incomplete) > 0 && false {\n		sort.Strings(incomplete)\n		md += fmt.Sprintf("\\n> **Varredura incompleta**')], "./internal/pose", "TestTheIncompleteScanWarning|TestTheConsolidatedAssessmentDeclares"),
 ("R4 item cortado fora do colchete some", [(RC, 'if section != "" && strings.HasPrefix(line, "-") {', 'if section != "" && strings.HasPrefix(line, "-") && strings.Contains(line, "[") {')], "./internal/pose", "TestAnItemCutAnywhere"),
 ("R4 diagnóstico copia a linha", [(RC, '"pose: review %s has a malformed or truncated item in its %s section (line %d of the body)", a.ReviewID, section, lineNo)', '"pose: review %s has a malformed or truncated item %q in its %s section (line %d of the body)", a.ReviewID, line, section, lineNo)')], "./internal/pose", "TestAnItemCutAnywhere"),
 ("R4 atestações ordenadas por texto", [(RB, 'return reviewInstantLess(result[i].AttestedAt, result[i].AttestationID, result[j].AttestedAt, result[j].AttestationID)', 'return result[i].AttestedAt < result[j].AttestedAt')], "./internal/pose", "TestAttestationsAreOrderedByInstant"),
 ("R4 follow-up open vazio some", [(FP, 'if stripped == "" && !(strings.TrimSpace(pending) == "[open]" && status != "done") {', 'if false {')], "./internal/pose", "TestAnEmptyOpenFollowup"),
 ("R4 schema futuro como parse-error", [(RB, 'if futureSchema(raw) {\n\t\t\t\t\t// Well-formed JSON of a format this engine does not read.', 'if false {\n\t\t\t\t\t// Well-formed JSON of a format this engine does not read.')], "./internal/pose", "TestAFutureSchemaIsUnsupported"),
 ("R4 estado de componente ilegível some", [("pose-mcp/internal/cli/assess.go", 'bySlug[slug] = unreadableComponentState(slug, filepath.Join(store.ComponentStateDir(), entry.Name()), err)\n\t\t\tcontinue', 'continue')], "./internal/cli", "TestAnUnreadablePersistedComponentState"),
 ("R5 bundle sem sealed_at é aceito", [(RB, 'if _, ok := reviewInstant(bundle.SealedAt); !ok {', 'if _, ok := reviewInstant(bundle.SealedAt); !ok && false {')], "./internal/pose", "TestABundleWithoutASealedInstant"),
 ("R5 follow-up anotado sem texto some", [(FP, 'if stripped == "" && !(strings.TrimSpace(pending) == "[open]" && status != "done") {', 'if stripped == "" && status == "done" {')], "./internal/pose", "TestAnAnnotatedFollowupThatLost"),
 ("R5 placeholder de draft vira perda", [(FP, 'if stripped == "" && !(strings.TrimSpace(pending) == "[open]" && status != "done") {', 'if stripped == "" {')], "./internal/pose", "TestAnAnnotatedFollowupThatLost|TestAnEmptyOpenFollowup"),
 ("R5 estado null é componente", [(DS, 'if state.ComponentSlug == "" {', 'if false {')], "./internal/pose", "TestAPersistedComponentStateIsValidated"),
 ("R5 estado de schema futuro é lido", [(DS, 'if state.SchemaVersion > 1 {', 'if false {')], "./internal/pose", "TestAPersistedComponentStateIsValidated"),
 ("R5 replay ordena bundles por texto", [("pose-mcp/internal/pose/governance_replay.go", 'reviewInstantLess(old.SealedAt, old.BundleID, bundle.SealedAt, bundle.BundleID)', 'bundle.SealedAt > old.SealedAt')], "./internal/pose", "TestNoConsumerOrdersTheBundleHistoryByText"),
 ("R5 rework ordena bundles por texto", [("pose-mcp/internal/pose/governance_waits.go", 'return reviewInstantLess(list[i].SealedAt, list[i].BundleID, list[j].SealedAt, list[j].BundleID)', 'return list[i].SealedAt < list[j].SealedAt')], "./internal/pose", "TestNoConsumerOrdersTheBundleHistoryByText"),
 ("R5 tipo de perda do estado é sempre read-error", [("pose-mcp/internal/cli/assess.go", 'case errors.Is(cause, pose.ErrComponentStateParse):', 'case false:')], "./internal/cli", "TestThePersistedStateLossKeepsTheKind"),
 ("R6 escopo repetido é de outro escopo", [(RC, 'if repeated := FrontmatterRepeats(string(raw), "scope"); len(repeated) > 0 {', 'if repeated := FrontmatterRepeats(string(raw), "scope"); len(repeated) > 0 && false {')], "./internal/pose", "TestARecordThatNamesItsScopeTwice"),
 ("R6 leitura global aceita campo repetido", [(RC, 'if repeated := FrontmatterRepeats(string(raw), "schema_version", "review_id", "scope", "scope_digest", "plan_digest", "profile", "reviewer", "decision", "reviewed_at", "supersedes", "evidence_refs"); len(repeated) > 0 {', 'if repeated := FrontmatterRepeats(string(raw), "schema_version", "review_id", "scope", "scope_digest", "plan_digest", "profile", "reviewer", "decision", "reviewed_at", "supersedes", "evidence_refs"); len(repeated) > 0 && false {')], "./internal/pose", "TestARecordThatNamesItsScopeTwice"),
 ("R6 discover publica estado só do que varreu", [("pose-mcp/internal/cli/assess.go", 'if c.Status == "unreadable" {', 'if false {')], "./internal/cli", "TestADiscoveryOfOneComponentKeeps"),
 ("R7 campo repetido fora da lista é lido", [(RC, '"schema_version", "review_id", "scope", "scope_digest", "plan_digest", "profile", "reviewer", "decision", "reviewed_at", "supersedes", "evidence_refs"); len(repeated) > 0 {', '"review_id", "scope", "decision", "reviewed_at"); len(repeated) > 0 {')], "./internal/pose", "TestARepeatedAttemptFieldIsALoss"),
 ("R7 escopo repetido atribuído ao último valor", [(RC, 'fm["scope"] != "" && len(FrontmatterRepeats(string(raw), "scope")) == 0 {', 'fm["scope"] != "" {')], "./internal/pose", "TestARepeatedAttemptFieldIsALoss"),
 ("R7 estado do projeto soma componente removido", [("pose-mcp/internal/cli/assess.go", 'if c.Status == "unreadable" {', 'if true {')], "./internal/cli", "TestAssessDiscoverUpdatesProjectStateFromObservedData"),
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
