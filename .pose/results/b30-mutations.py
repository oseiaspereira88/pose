"""Production mutations of B30 (spec pose-spec-facts-evidence-binding).

Each mutation reverts one guarantee in a COPY of the repository (the working tree is
never edited), runs the named tests there and must make them fail on an assertion,
not on a build error. Run from the repository root:

    python3 -I .pose/results/b30-mutations.py --output <file.json> [--only <text>]
"""
import argparse, json, pathlib, shutil, subprocess, tempfile

parser = argparse.ArgumentParser(description=__doc__)
parser.add_argument('--output', required=True)
parser.add_argument('--only', default='', help='run only the mutations whose name contains this text')
args = parser.parse_args()
root = pathlib.Path('.').resolve()
SF = "pose-mcp/internal/pose/spec_facts.go"
SC = "pose-mcp/internal/cli/specs_cmd.go"
P = "./internal/pose"
C = "./internal/cli"

M = [
 ("módulo de outro spec vira associado por componente", [(SF, 'case scoped && scopedValue != "" && scopedValue == expected && covered:', 'case scoped || modules[normalizeFactPath(r.Module)]:')], P, "TestFactsSeparateAssociatedEvidenceFromCandidates"),
 ("sem componentes todo resultado é associado", [(SF, 'case scoped && scopedValue != "" && scopedValue == expected && covered:', 'case scoped || len(modules) == 0:')], P, "TestFactsSeparateAssociatedEvidenceFromCandidates"),
 ("candidato sem limite", [(SF, 'fact.Limits = append(fact.Limits, "the result carries no scope provenance and its module is not declared by the spec: it is not evidence of this spec")', '_ = fact')], P, "TestFactsSeparateAssociatedEvidenceFromCandidates"),
 ("módulo declarado sem provenance perde o limite", [(SF, 'fact.Limits = append(fact.Limits, "the module is declared by the spec, but the result carries no scope provenance: it is not shown to cover this spec\'s delta")', '_ = fact')], P, "TestFactsSeparateAssociatedEvidenceFromCandidates"),
 ("resultado de outro spec não é contado", [(SF, 'otherSpecs++\n\t\t\tcontinue', 'continue')], P, "TestFactsSeparateAssociatedEvidenceFromCandidates"),
 ("head do resultado não é comparado", [(SF, 'case heads[head]:\n\t\treturn "same-head"', 'case heads[head]:\n\t\treturn "other-head"')], P, "TestFactsKeepAResultOfAnotherHead"),
 ("head desconhecido é tratado como igual", [(SF, 'case head == "":\n\t\treturn "unknown-head"', 'case head == "":\n\t\treturn "same-head"')], P, "TestFactsKeepAResultOfAnotherHead"),
 ("resultado de outro head perde a fonte", [(SF, 'Binding: SpecFactBinding{GitHead: r.GitHead, ProvenanceDigest: r.ProvenanceDigest,', 'Binding: SpecFactBinding{ProvenanceDigest: r.ProvenanceDigest,')], P, "TestFactsKeepAResultOfAnotherHead"),
 ("índice com input alterado é fresh", [(SF, 'case len(stale) > 0:', 'case len(stale) > 0 && false:')], P, "TestFactsSayHowFreshTheIndexIs"),
 ("índice não publicado é fresh", [(SF, 'return SpecFactsFreshness{Status: SpecFactsUnknown, Reason: view.Reason}', 'return SpecFactsFreshness{Status: SpecFactsFresh}')], P, "TestFactsSayHowFreshTheIndexIs"),
 ("índice stale sem próximo passo", [(SF, 'run `pose index` before relying on these facts', 'ok')], P, "TestFactsSayHowFreshTheIndexIs"),
 ("saída humana imprime candidato como check", [(SC, 'out.Field("facts.candidate", factLine', 'out.Field("facts.check", factLine')], C, "TestSpecsFactsLabelsCandidates"),
 ("saída humana sem frescor", [(SC, 'out.Field("facts.freshness", freshness)', '_ = freshness')], C, "TestSpecsFactsLabelsCandidates"),
 ("chave presente vale como associação", [(SF, 'case scoped && scopedValue != "" && scopedValue == expected && covered:', 'case scoped && covered:')], P, "TestFactsAssociateOnlyAScopeProvenanceThatMatches"),
 ("módulo alheio é associado", [(SF, 'case scoped && scopedValue != "" && scopedValue == expected && covered:', 'case scoped && scopedValue != "" && scopedValue == expected:')], P, "TestFactsAssociateOnlyAModuleThatCovers"),
 ("resultados do mesmo módulo e check colapsam", [(SF, 'identity := strings.Join([]string{r.ID, r.Module, r.Check, r.Outcome, r.GitHead, r.Report}, "\\x00")', 'identity := r.Module + "|" + r.Check')], P, "TestFactsKeepEveryDistinctResult"),
 ("geração sem bindings é fresh", [(SF, 'case len(unbound) > 0:', 'case len(unbound) > 0 && false:')], P, "TestFactsAGenerationThatBindsNothingIsUnknown"),
 ("arquivo de índice divergente é fresh", [(SF, 'case len(diverged) > 0:', 'case len(diverged) > 0 && false:')], P, "TestFactsAreFreshOnlyForWhatIsVerified"),
 ("arquivo de resultados trocado é fresh", [(SF, 'reason != "" {\n\t\tdegradeFreshness(&facts.Freshness, SpecFactsStale, "validation-results"', 'reason != "" && false {\n\t\tdegradeFreshness(&facts.Freshness, SpecFactsStale, "validation-results"')], P, "TestFactsAreFreshOnlyForWhatIsVerified"),
 ("commit do spec fora do índice é fresh", [(SF, 'case missing > 0:', 'case missing > 0 && false:')], P, "TestFactsSeeACommitOfTheSpec"),
 ("saída humana omite limites", [(SC, 'if len(c.Limits) > 0 {', 'if false {')], C, "TestSpecsFactsLabelsCandidates"),
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
    with tempfile.TemporaryDirectory(prefix="b30-mut-") as tmp:
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
