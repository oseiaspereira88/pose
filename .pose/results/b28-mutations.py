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
 ("todo backend vira Go", [(SG, 'if lang == "go" || strings.HasSuffix(norm, ".go") || goMod {', 'if true || goMod {')], C, "TestSuggestDoesNotTurnEveryBackendIntoGo"),
 ("regra ausente é anunciada como instalada", [(SG, 'case state == ruleFileReadable:', 'case true:')], C, "TestSuggestSeparatesDeclaredFromInstalledRules"),
 ("regra de extensão não indica o comando", [(SG, 'st.NextStep = "pose extension install " + byFile[file]', 'st.NextStep = ""')], C, "TestSuggestSeparatesDeclaredFromInstalledRules"),
 ("saída humana anuncia arquivo inexistente", [(SG, 'case ruleExtensionNeeded:\n\t\t\tpaths = append(paths, st.Rule+" (not installed: "+st.NextStep+")")', 'case ruleExtensionNeeded:\n\t\t\tpaths = append(paths, ".pose/rules/"+st.Rule+".md")')], C, "TestSuggestSeparatesDeclaredFromInstalledRules"),
 ("migrate promete apply futuro", [(MG, 'no apply is available in this engine; the only mode is', 'a 7.0 apply will carry its own recovery; the only mode is')], C, "TestMigrateSaysThereIsNoApplyWithoutPromisingOne"),
 ("migrate não diz que dry-run não escreve", [(MG, 'which reads the repository, writes nothing and reports', 'which reads the repository and reports')], C, "TestMigrateSaysThereIsNoApplyWithoutPromisingOne"),
 ("módulo raiz não se aplica", [(SG, 'if !(p == "" || norm == p || strings.HasPrefix(norm, p+"/")) || len(p) <= best {', 'if !(p != "" && (norm == p || strings.HasPrefix(norm, p+"/"))) || len(p) <= best {')], C, "TestSuggestRootModuleIsTheFallback"),
 ("caminho não normalizado", [(SG, 'p = normalizeModulePath(p)\n', '')], C, "TestSuggestRootModuleIsTheFallback"),
 ("kubernetes sem extensão conhecida", [("pose-mcp/internal/cli/rule_extension_resolver.go", '"pose-rule-kubernetes":          "kubernetes.md",', '')], C, "TestSuggestKubernetesRuleIsAnnounced"),
 ("diretório é regra instalada", [("pose-mcp/internal/cli/rule_extension_resolver.go", 'if !info.Mode().IsRegular() {', 'if false {')], C, "TestSuggestAnUnreadableRuleIsNot"),
 ("go.mod diretório vale como módulo Go", [(SG, 'err == nil && info.Mode().IsRegular() {', 'err == nil && info != nil {')], C, "TestSuggestAnUnreadableRuleIsNot"),
 ("configuração malformada vira conjunto vazio", [(SG, 'return nil, fmt.Errorf("expected a list of strings, got %T", value)', 'return nil, nil')], C, "TestSuggestRefusesAMalformedRulesConfiguration"),
 ("elemento que não é texto é descartado", [(SG, 'return nil, fmt.Errorf("element %d is %T, not a string", i, item)', 'continue')], C, "TestSuggestRefusesAMalformedRulesConfiguration"),
 ("rules_by_domain que não é objeto é aceito", [(SG, 'return nil, nil, fmt.Errorf("rules_by_domain: expected an object, got %T", raw)', 'break')], C, "TestSuggestRefusesAMalformedRulesConfiguration"),
 ("identificador de regra não é validado", [(SG, 'if verr := posepkg.ValidateName(rule); verr != nil {', 'if verr := error(nil); verr != nil {')], C, "TestSuggestRefusesARuleIdentifierThatIsAPath"),
 ("índice ilegível vira ausência de domínio", [(SG, 'return "", "repo-map-unavailable", "the repo-map could not be read ("+e.Error()+")"', 'return "", "repo-map-no-match", ""')], C, "TestSuggestSaysWhyItHasNoDomainForAPath|TestSuggestDomainUsesMostSpecificModule"),
 ("inventário ignora diretório ilegível", [("pose-mcp/internal/cli/distributed_instructions_parity_test.go", 'if err != nil {\n\t\t\treturn err\n\t\t}\n\t\tif !d.IsDir() && strings.HasSuffix', 'if err != nil {\n\t\t\treturn nil\n\t\t}\n\t\tif !d.IsDir() && strings.HasSuffix')], C, "TestInventoryNeverSkipsWhatItCannotRead"),
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
    with tempfile.TemporaryDirectory(prefix="b28-mut-") as tmp:
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
