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
 ("campo desconhecido é aceito", [(RM, 'decoder.DisallowUnknownFields()', '')], P, "TestAManifestIsValidOnlyIf"),
 ("conteúdo após o manifest é aceito", [(RM, 'if decoder.More() {', 'if false && decoder.More() {')], P, "TestAManifestIsValidOnlyIf"),
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
 ("notas do worktree substituem as da tag", [(RM, 'notes, err := gitBlobAt(root, commit, ".pose/changelogs/"+tag+".md")', 'notes, err := os.ReadFile(filepath.Join(root, ".pose", "changelogs", tag+".md"))')], P, "TestTheSnapshotIsVerifiedAgainstTheTaggedTree"),
 ("digest das notas não é comparado", [(RM, 'case ReleaseDigest(string(notes)) != manifest.NotesDigest:', 'case false && ReleaseDigest(string(notes)) != manifest.NotesDigest:')], P, "TestTheSnapshotIsVerifiedAgainstTheTaggedTree"),
 ("digest de fragment não é comparado", [(RM, 'case ReleaseDigest(string(blob)) != f.Digest:', 'case false && ReleaseDigest(string(blob)) != f.Digest:')], P, "TestTheSnapshotIsVerifiedAgainstTheTaggedTree"),
 ("fragment extra na árvore da tag é aceito", [(RM, 'if !frozen[name] {', 'if false && !frozen[name] {')], P, "TestTheSnapshotIsVerifiedAgainstTheTaggedTree"),
 ("fonte de versão da tag não é relida", [(RM, 'if got != value {', 'if false && got != value {')], P, "TestTheSnapshotIsVerifiedAgainstTheTaggedTree"),
 ("fonte não recomputável vira falha", [(RM, 'return "", "the version evidence is not re-verifiable from the manifest alone (kind " + fmt.Sprintf("%q", kind) + ")"', 'return "the version evidence is not re-verifiable", ""')], P, "TestAVersionSourceThatCannotBeRecomputedIsALimit"),
 ("commit posterior invalida o cut", [(RM, 'commit, annotated, err := ResolveReleaseTag(root, tag)\n\tif err != nil {\n\t\treturn ReleaseSnapshot{Tag: tag}, err\n\t}', 'commit, annotated, err := ResolveReleaseTag(root, tag)\n\tif err != nil {\n\t\treturn ReleaseSnapshot{Tag: tag}, err\n\t}\n\tif head, herr := gitOutput(root, "rev-parse", "HEAD"); herr == nil && strings.TrimSpace(head) != commit {\n\t\treturn ReleaseSnapshot{Tag: tag, Commit: commit}, ErrReleaseSnapshotMissing\n\t}')], P, "TestACorrectCutStaysIntactAfterEvidenceAndTheNextCycle"),
 ("status marca tagged sem snapshot", [(RL, 'case snapErr != nil:', 'case snapErr != nil && false:')], P, "TestATagWithoutTheSnapshotInItsTree"),
 ("check não verifica o snapshot da tag", [(CL, 'gaps = append(gaps, releaseTagSnapshotGaps(root, target)...)', '')], C, "TestReleaseCheckReadsTheTagsOwnTree"),
 ("limites do snapshot não aparecem no JSON", [(CL, '"gaps": gaps, "snapshot_limits": tagLimits}', '"gaps": gaps, "snapshot_limits": len(tagLimits) * 0}')], C, "TestReleaseCheckDisclosesTheVersionSources"),
]

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
        r = subprocess.run(f"cd {work} && go test {pkgs} -run '{run}' -count=1", shell=True, capture_output=True, text=True, timeout=900)
        out = r.stdout + r.stderr
        build_error = 'build failed' in out or 'setup failed' in out
        verdict = 'killed' if r.returncode and not build_error else ('build-error' if build_error else 'survived')
        results.append({'mutation': name, 'tests': run, 'verdict': verdict})
        print(verdict, name, flush=True)
pathlib.Path(args.output).write_text(json.dumps({'mutations': results, 'killed': sum(r['verdict'] == 'killed' for r in results), 'total': len(results)}, indent=2, ensure_ascii=False) + '\n')
