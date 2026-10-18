"""Production mutations of B38 (spec pose-release-manifest-tag-integrity).

Each mutation reverts one guarantee in a COPY of the repository (the working tree is
never edited), runs the named tests there and must make them fail on an assertion,
not on a build error. Run from the repository root:

    python3 -I .pose/results/b38-mutations.py --output <file.json> [--only <text>]
"""
import argparse, json, pathlib, shutil, subprocess, tempfile

parser = argparse.ArgumentParser(description=__doc__)
parser.add_argument('--output', required=True)
parser.add_argument('--only', default='', help='run only the mutations whose name contains this text')
args = parser.parse_args()
root = pathlib.Path('.').resolve()
RM = "pose-mcp/internal/pose/release_manifest_integrity.go"
RL = "pose-mcp/internal/pose/release_lifecycle.go"
CL = "pose-mcp/internal/cli/release_lifecycle.go"
P = "./internal/pose"
C = "./internal/cli"

M = [
 ("campo desconhecido é aceito", [(RM, 'if !releaseManifestFields[name] {', 'if false && !releaseManifestFields[name] {')], P, "TestAManifestIsValidOnlyIf"),
 ("schema desconhecido é aceito", [(RM, 'if manifest.SchemaVersion != ReleaseManifestSchemaVersion {\n\t\tadd("schema_version"', 'if manifest.SchemaVersion != ReleaseManifestSchemaVersion && false {\n\t\tadd("schema_version"')], P, "TestAManifestIsValidOnlyIf"),
 ("versão de outro diretório é aceita", [(RM, '} else if version != "" && manifest.Version != version {', '} else if false {')], P, "TestAManifestIsValidOnlyIf"),
 ("projeção specs adulterada é aceita", [(RM, 'if !equalStrings(manifest.Specs, specs) {', 'if false && !equalStrings(manifest.Specs, specs) {')], P, "TestAManifestIsValidOnlyIf"),
 ("projeção categories adulterada é aceita", [(RM, 'if !equalCounts(manifest.Categories, categories) {', 'if false && !equalCounts(manifest.Categories, categories) {')], P, "TestAManifestIsValidOnlyIf"),
 ("projeção breaking adulterada é aceita", [(RM, 'if manifest.Breaking != breaking {', 'if false && manifest.Breaking != breaking {')], P, "TestAManifestIsValidOnlyIf"),
 ("digest de entrada adulterado é aceito", [(RM, 'if ReleaseDigest(input) != manifest.ReleaseInputDigest {', 'if false && ReleaseDigest(input) != manifest.ReleaseInputDigest {')], P, "TestAManifestIsValidOnlyIf"),
 ("fragment duplicado é aceito", [(RM, 'case seen[f.Spec]:', 'case false && seen[f.Spec]:')], P, "TestAManifestIsValidOnlyIf"),
 ("caminho de fragment fora do diretório é aceito", [(RM, 'if f.Path == "" || strings.ContainsAny(f.Path, `/\\`) || f.Path == "." || f.Path == ".." {', 'if false {')], P, "TestAManifestIsValidOnlyIf"),
 ("fonte de versão com caminho fora do projeto é aceita", [(RM, 'if filepath.IsAbs(source) || strings.Contains(source, "\\\\") || path.Clean(source) != source || strings.HasPrefix(source, "../") {', 'if (filepath.IsAbs(source) || strings.Contains(source, "\\\\") || path.Clean(source) != source || strings.HasPrefix(source, "../")) && false {')], P, "TestAManifestIsValidOnlyIf"),
 ("tag é lida do worktree", [(RM, 'raw, err := gitBlobAt(root, commit, manifestPath)', 'raw, err := os.ReadFile(filepath.Join(root, filepath.FromSlash(manifestPath)))')], P, "TestATagWithoutTheSnapshotInItsTree"),
 ("tag anotada não é resolvida ao commit", [(RM, '"refs/tags/"+tag+"^{commit}"', '"refs/tags/"+tag')], P, "TestLightweightAndAnnotatedTagsResolve"),
 ("notas do worktree substituem as da tag", [(RM, 'notes, err := gitBlobAt(root, commit, ".pose/changelogs/"+tag+".md")', 'notes, err := os.ReadFile(filepath.Join(root, ".pose", "changelogs", tag+".md"))')], P, "TestTheSnapshotIsVerifiedAgainstTheTaggedTree|TestF02"),
 ("digest das notas não é comparado", [(RM, 'case ReleaseDigest(string(notes)) != manifest.NotesDigest:', 'case false && ReleaseDigest(string(notes)) != manifest.NotesDigest:')], P, "TestTheSnapshotIsVerifiedAgainstTheTaggedTree|TestF02|TestR2Notes"),
 ("digest de fragment não é comparado", [(RM, 'case ReleaseDigest(string(blob)) != f.Digest:', 'case false && ReleaseDigest(string(blob)) != f.Digest:')], P, "TestTheSnapshotIsVerifiedAgainstTheTaggedTree|TestF02"),
 ("fragment extra na árvore da tag é aceito", [(RM, 'if _, ok := frozen[name]; !ok {', 'if _, ok := frozen[name]; false && !ok {')], P, "TestTheSnapshotIsVerifiedAgainstTheTaggedTree|TestF02"),
 ("fonte não recomputável vira falha", [(RM, 'return "", "the version evidence is the engine\'s own version file: it exists in the tagged tree and its value was not re-read"', 'return "the version evidence is the engine\'s own version file", ""')], P, "TestAVersionSourceThatCannotBeRecomputedIsALimit"),
 ("commit posterior invalida o cut", [(RM, 'commit, annotated, err := ResolveReleaseTag(root, tag)\n\tif err != nil {\n\t\treturn ReleaseSnapshot{Tag: tag}, err\n\t}', 'commit, annotated, err := ResolveReleaseTag(root, tag)\n\tif err != nil {\n\t\treturn ReleaseSnapshot{Tag: tag}, err\n\t}\n\tif head, herr := gitOutput(root, "rev-parse", "HEAD"); herr == nil && strings.TrimSpace(head) != commit {\n\t\treturn ReleaseSnapshot{Tag: tag, Commit: commit}, ErrReleaseSnapshotMissing\n\t}')], P, "TestACorrectCutStaysIntactAfterEvidenceAndTheNextCycle"),
 ("status marca tagged sem snapshot", [(RL, 'case snapErr != nil:', 'case snapErr != nil && false:')], P, "TestATagWithoutTheSnapshotInItsTree"),
 ("check não verifica o snapshot da tag", [(CL, 'tagGaps, tagged := releaseTagState(root, target)', '_, tagged := releaseTagState(root, target)\n\ttagGaps := []string{}')], C, "TestReleaseCheckReadsTheTagsOwnTree"),
 ("limites do snapshot não aparecem no JSON", [(CL, '"gaps": gaps, "snapshot_limits": tagLimits}', '"gaps": gaps, "snapshot_limits": len(tagLimits) * 0}')], C, "TestReleaseCheckDisclosesTheVersionSources"),
 ("chave repetida é aceita", [(RM, 'if err := rejectDuplicateJSONKeys(raw, false, true); err != nil {', 'if err := rejectDuplicateJSONKeys(raw, false, true); false && err != nil {')], P, "TestF01"),
 ("campo obrigatório ausente é aceito", [(RM, 'if _, ok := fields[name]; !ok && name != "previous_version" {', 'if _, ok := fields[name]; false && !ok && name != "previous_version" {')], P, "TestF01"),
 ("campo null é aceito", [(RM, '} else if string(bytes.TrimSpace(value)) == "null" {', '} else if false && string(bytes.TrimSpace(value)) == "null" {')], P, "TestF01"),
 ("categoria do manifest não é recomposta do blob", [(RM, 'case strings.TrimSpace(fm["category"]) == "" || strings.TrimSpace(fm["category"]) != f.Category:', 'case false:')], P, "TestF02"),
 ("corpo do manifest não é recomposto do blob", [(RM, 'case body != f.Body:', 'case false:')], P, "TestF02"),
 ("dois specs no mesmo arquivo passam", [(RM, 'if other, dup := frozen[f.Path]; dup {', 'if other, dup := frozen[f.Path]; false && dup {')], P, "TestF02"),
 ("digest de policy não é vinculado à policy da tag", [(RM, '} else if ReleaseDigest(policy) != manifest.PolicyDigest {', '} else if false {')], P, "TestF03"),
 ("árvore sem policy passa", [(RM, 'case errors.Is(perr, os.ErrNotExist):\n\t\tgap("the tagged tree holds no release policy', 'case errors.Is(perr, os.ErrNotExist) && false:\n\t\tgap("the tagged tree holds no release policy')], P, "TestF03"),
 ("evidência de outra versão passa", [(RM, 'case manifest.Version != "" && evidence["value"] != manifest.Version:', 'case false:')], P, "TestF04"),
 ("fonte de versão ausente da árvore passa", [(RM, 'return "version source " + source + " is missing from the tagged tree", ""', 'return "", ""')], P, "TestF04"),
 ("status avança uma release de snapshot quebrado", [(RL, 'if projection.State == "prepared" && snap.Intact() {', 'if projection.State == "prepared" {')], P, "TestF05"),
 ("diretório de releases ilegível vira vazio", [(RL, 'return nil, fmt.Errorf("pose: the release directory could not be read: %w", err)', 'entries = nil')], P, "TestF06"),
 ("arquivo no lugar do arquivo morto passa", [(RM, 'return nil, true, fmt.Errorf("pose: %s of the tagged tree is a %s, not a directory", dir, kind)', 'return nil, false, nil')], P, "TestF06"),
 ("rc.2 depois de rc.10", [(RL, 'case xn && yn:', 'case false:')], P, "TestF10"),
 ("versão final antes do release candidate", [(RL, 'case pa == "":\n\t\treturn 1', 'case pa == "":\n\t\treturn -1')], P, "TestF10"),
 ("previous mais novo que a release passa", [(RM, 'CompareReleasePrecedence(manifest.PreviousVersion, manifest.Version) >= 0', 'CompareReleasePrecedence(manifest.PreviousVersion, manifest.Version) >= 0 && false')], P, "TestF10"),
 ("branch homônimo vale como tag", [(CL, 'commit, _, err := posemodel.ResolveReleaseTag(root, target)\n', 'out, err := exec.Command("git", "-C", root, "rev-parse", "--verify", target+"^{commit}").Output()\n\tcommit := strings.TrimSpace(string(out))\n')], C, "TestF07"),
 ("policy de hoje invalida o corte da tag", [(CL, '\tif tagged {\n\t\t// A tagged release is judged', '\tif tagged && false {\n\t\t// A tagged release is judged')], C, "TestF08"),
 ("worktree com outro corte passa", [("pose-mcp/internal/pose/release_lifecycle.go", '} else if ReleaseDigest(string(rawManifest)) != snap.ManifestDigest {', '} else if false && ReleaseDigest(string(rawManifest)) != snap.ManifestDigest {')], C, "TestF09"),
 ("record aceita tag de árvore quebrada", [(CL, 'if gaps, _ := releaseTagState(root, target); len(gaps) > 0 {', 'if gaps, _ := releaseTagState(root, target); false && len(gaps) > 0 {')], C, "TestF05"),
 ("fragment aninhado com null é aceito", [(RM, 'case string(bytes.TrimSpace(value)) == "null":\n\t\t\t\tadd(fmt.Sprintf("fragments[%d].%s", i, name), "is null")', 'case false && string(bytes.TrimSpace(value)) == "null":\n\t\t\t\tadd(fmt.Sprintf("fragments[%d].%s", i, name), "is null")')], P, "TestR2NestedFragmentObjectsAreStrict"),
 ("fragment aninhado sem campo obrigatório é aceito", [(RM, 'if _, ok := obj[name]; !ok && name != "refs" {', 'if _, ok := obj[name]; false && !ok && name != "refs" {')], P, "TestR2NestedFragmentObjectsAreStrict"),
 ("testemunho de archival não confere a spec do blob", [("pose-mcp/internal/pose/release_archive.go", 'err == nil && named && ReleaseDigest(string(raw)) == fragment.Digest', 'err == nil && (named || true) && ReleaseDigest(string(raw)) == fragment.Digest')], P, "TestR2ArchivalWitnessIsTheSpecOfTheBlob"),
 ("status não confere o worktree com o corte", [(RL, 'projection.Gaps = append(projection.Gaps, TaggedReleaseWorktreeGaps(s.Root, entry.Name(), manifest, snap)...)', '_ = TaggedReleaseWorktreeGaps(s.Root, entry.Name(), manifest, snap)')], P, "TestR2StatusSeesTheFilesOfTheCutThatWereLost"),
 ("fonte JSON que diz outra versão passa", [(RM, 'if got != value || got != manifest.Version {', 'if false {')], P, "TestR2VersionEvidenceFollowsThePolicyOfTheCut"),
 ("arquivo qualquer vale como versão do motor", [(RM, '} else if source != EngineVersionFile || kind != "" {', '} else if false {')], P, "TestR2VersionEvidenceFollowsThePolicyOfTheCut"),
 ("policy do corte não é validada", [(RM, 'if verr := ValidateReleasePolicy(policy); verr != nil {', 'if verr := ValidateReleasePolicy(policy); false && verr != nil {')], P, "TestR2PoliciesOfTheCutAreValidated"),
 ("categoria fora da policy do corte passa", [(RM, 'if clErr == nil && !changelog.ValidCategories()[f.Category] {', 'if false && clErr == nil && !changelog.ValidCategories()[f.Category] {')], P, "TestR2PoliciesOfTheCutAreValidated"),
 ("notas arbitrárias passam", [(RM, 'case clErr == nil && string(notes) != RenderReleaseNotes(manifest.Version, manifest.Fragments, changelog):', 'case false:')], P, "TestR2NotesAreTheRenderingOfTheFragments"),
 ("listagem de tags que falha vira vazia", [(RL, 'if tagsErr != nil && !gitNotARepository(s.Root) {', 'if false && tagsErr != nil && !gitNotARepository(s.Root) {')], P, "TestR2GitFailuresAreNotAbsence"),
 ("prepare sobrescreve manifest inválido", [(CL, 'case !errors.Is(loadErr, os.ErrNotExist):', 'case false && !errors.Is(loadErr, os.ErrNotExist):')], C, "TestR2PrepareDoesNotOverwrite"),
 ("evidência do candidato sem a fonte da policy", [(CL, 'if evidence["source"] != manifest.VersionEvidence["source"] || evidence["kind"] != manifest.VersionEvidence["kind"] {', 'if false {')], C, "TestR2PreparedEvidenceNamesTheSourceOfThePolicy"),
 ("controle em texto livre recusa história real", [(RM, 'if err := rejectDuplicateJSONKeys(raw, false, true); err != nil {', 'if err := rejectDuplicateJSONKeys(raw, true, true); err != nil {')], P, "TestR4MultilineBodyIsFrozenAndLoads|TestEveryHistoricalRelease"),
 ("link no lugar do arquivo de versão", [(RM, 'if kind != "blob" || (mode != "100644" && mode != "100755") {', 'if kind != "blob" {')], P, "TestR4ALinkInPlaceOfTheVersionSource"),
 ("candidato que perdeu arquivos é sólido", [(RL, 'projection.Gaps = append(projection.Gaps, ReleaseFilesGaps(s.Root, entry.Name(), manifest)...)\n\t\t\tif projection.State', '_ = ReleaseFilesGaps(s.Root, entry.Name(), manifest)\n\t\t\tif projection.State')], P, "TestR4APreparedStatusSeesTheFilesItLost"),
 ("verified sem solidez deixa de dever", [("pose-mcp/internal/pose/obligation_sources.go", 'if (release.State == "verified" && release.Sound()) || release.State == "yanked" {', 'if release.State == "verified" || release.State == "yanked" {')], P, "TestR4AnUnsoundVerifiedReleaseStillOwes"),
 ("archival não confere o corte da tag", [("pose-mcp/internal/pose/release_archive.go", 'Intact:         boundToTag && err == nil', 'Intact:         (boundToTag || true) && err == nil')], P, "TestR4ArchivalIsBoundToTheCutOfTheTag"),
 ("fallback de policy após erro de leitura", [(RL, 'if err != nil && errors.Is(err, os.ErrNotExist) {\n\t\t// The legacy place', 'if err != nil {\n\t\t// The legacy place')], P, "TestR4ALostReleaseDirectoryAndPolicyAreNotAbsence"),
 ("diretório de releases dangling é vazio", [(RL, 'if _, lerr := os.Lstat(base); !errors.Is(err, os.ErrNotExist) || lerr == nil {', 'if _, lerr := os.Lstat(base); !errors.Is(err, os.ErrNotExist) && lerr == nil {')], P, "TestR4ALostReleaseDirectoryAndPolicyAreNotAbsence"),
 ("cutoff inválido é aceito", [(RL, 'if err := ValidateReleaseVersion(policy.LegacyCutoff); err != nil {', 'if err := ValidateReleaseVersion(policy.LegacyCutoff); false && err != nil {')], P, "TestR4TheCutoffIsAVersionAndTheChangelogSchemaIsSupported"),
 ("schema futuro do changelog é aceito", [(RM, 'if policy.SchemaVersion != 1 {\n\t\treturn policy, fmt.Errorf("the changelog policy schema_version', 'if policy.SchemaVersion != 1 && false {\n\t\treturn policy, fmt.Errorf("the changelog policy schema_version')], P, "TestR4TheCutoffIsAVersionAndTheChangelogSchemaIsSupported"),
 ("notas preparadas não são a renderização", [(CL, '} else if string(notes) != posemodel.RenderReleaseNotes(manifest.Version, manifest.Fragments, posemodel.LoadChangelogPolicy(root)) {', '} else if false {')], C, "TestR4PreparedNotesAreTheRendering"),
 ("release notes ignora os gaps", [(CL, 'if manifest == nil || len(gaps) > 0 || posemodel.ReleaseDigest(string(raw)) != manifest.NotesDigest {', 'if manifest == nil || len(gaps) > 0 && false || posemodel.ReleaseDigest(string(raw)) != manifest.NotesDigest {')], C, "TestR4PreparedNotesAreTheRendering"),
 ("eventos de tag sem a tag são válidos", [(CL, 'if !tagged && projection.State != "prepared" && projection.State != "failed" {', 'if false && !tagged && projection.State != "prepared" && projection.State != "failed" {')], C, "TestR4EventsThatNameATagThatIsNotThere"),
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
    with tempfile.TemporaryDirectory(prefix="b38-mut-") as tmp:
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
