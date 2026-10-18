"""Production mutations of B28 (spec pose-distributed-instructions-parity).

Each mutation reverts one guarantee in a COPY of the repository (the working tree is
never edited), runs the named tests there and must make them fail on an assertion,
not on a build error. Run from the repository root:

    python3 -I .pose/results/b28-mutations.py --output <file.json> [--only <text>]
"""
import argparse, json, pathlib, shutil, subprocess, tempfile

parser = argparse.ArgumentParser(description=__doc__)
parser.add_argument('--output', required=True)
parser.add_argument('--only', default='', help='run only the mutations whose name contains this text')
args = parser.parse_args()
root = pathlib.Path('.').resolve()
SG = "pose-mcp/internal/cli/suggest.go"
MG = "pose-mcp/internal/cli/migrate_dryrun.go"
C = "./internal/cli"

M = [
 ("workflow volta a pedir discovery incondicional", [(".pose/workflows/feature.md", "`pose assess discover --if-stale [--component <dir>]` to inspect", "`pose assess discover [--component <dir>]` to inspect")], C, "TestDistributedInstructionsAskForDiscoveryOnlyWhenStale"),
 ("espelho embarcado volta a pedir discovery incondicional", [("pose-mcp/internal/scaffold/dist/locales/pt-BR/.pose/rules/delivery-evidence.md", "`pose assess discover --if-stale --update-state`", "`pose assess discover --update-state`")], C, "TestDistributedInstructionsAskForDiscoveryOnlyWhenStale|TestAFreshInstallReceivesInstructions"),
 ("espelho embarcado diverge da fonte", [("pose-mcp/internal/scaffold/dist/.pose/workflows/feature.md", "to inspect metrics", "to inspect the metrics")], C, "TestTheEmbeddedMirrorsOfTheWorkflowsAndRulesMatchTheSources"),
 ("alias go não resolve", [(SG, '"go": "backend-go", "golang": "backend-go", "backend-go": "backend-go",', '"backend-go": "backend-go",')], C, "TestSuggestResolvesDomainAliasesThroughOneTaxonomy"),
 ("alias não é divulgado", [(SG, 'if aliased && source == "explicit" {', 'if aliased && false && source == "explicit" {')], C, "TestSuggestResolvesDomainAliasesThroughOneTaxonomy"),
 ("domínio desconhecido não é reportado", [(SG, 'if !domainKnown {\n\t\t\t\tpayload["domain_status"]', 'if false {\n\t\t\t\tpayload["domain_status"]')], C, "TestSuggestResolvesDomainAliasesThroughOneTaxonomy"),
 ("todo backend vira Go", [(SG, 'if _, goMod := os.Stat(filepath.Join(root, filepath.FromSlash(p), "go.mod")); lang == "go" || strings.HasSuffix(norm, ".go") || goMod == nil {', 'if _, goMod := os.Stat(filepath.Join(root, filepath.FromSlash(p), "go.mod")); true || lang == "go" || strings.HasSuffix(norm, ".go") || goMod == nil {')], C, "TestSuggestDoesNotTurnEveryBackendIntoGo"),
 ("regra ausente é anunciada como instalada", [(SG, 'case ruleExtensionInstalled(root, file):', 'case true:')], C, "TestSuggestSeparatesDeclaredFromInstalledRules"),
 ("regra de extensão não indica o comando", [(SG, 'st.NextStep = "pose extension install " + byFile[file]', 'st.NextStep = ""')], C, "TestSuggestSeparatesDeclaredFromInstalledRules"),
 ("saída humana anuncia arquivo inexistente", [(SG, 'case ruleExtensionNeeded:\n\t\t\tpaths = append(paths, st.Rule+" (not installed: "+st.NextStep+")")', 'case ruleExtensionNeeded:\n\t\t\tpaths = append(paths, ".pose/rules/"+st.Rule+".md")')], C, "TestSuggestSeparatesDeclaredFromInstalledRules"),
 ("migrate promete apply futuro", [(MG, 'no apply is available in this engine; the only mode is', 'a 7.0 apply will carry its own recovery; the only mode is')], C, "TestMigrateSaysThereIsNoApplyWithoutPromisingOne"),
 ("migrate não diz que dry-run não escreve", [(MG, 'which reads the repository, writes nothing and reports', 'which reads the repository and reports')], C, "TestMigrateSaysThereIsNoApplyWithoutPromisingOne"),
]

results = []
for name, edits, pkgs, run in M:
    if args.only and args.only not in name:
        continue
    with tempfile.TemporaryDirectory(prefix="b28-mut-") as tmp:
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
