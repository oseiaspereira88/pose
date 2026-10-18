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
 ("módulo de outro spec vira associado por componente", [(SF, 'case scoped:\n\t\t\tfact.Classification, fact.Basis = SpecFactAssociated, "scope-provenance"', 'case scoped || modules[r.Module]:\n\t\t\tfact.Classification, fact.Basis = SpecFactAssociated, "scope-provenance"')], P, "TestFactsSeparateAssociatedEvidenceFromCandidates"),
 ("sem componentes todo resultado é associado", [(SF, 'case scoped:\n\t\t\tfact.Classification, fact.Basis = SpecFactAssociated, "scope-provenance"', 'case scoped || len(modules) == 0:\n\t\t\tfact.Classification, fact.Basis = SpecFactAssociated, "scope-provenance"')], P, "TestFactsSeparateAssociatedEvidenceFromCandidates"),
 ("candidato sem limite", [(SF, 'fact.Limits = append(fact.Limits, "the result carries no scope provenance and its module is not declared by the spec: it is not evidence of this spec")', '_ = fact')], P, "TestFactsSeparateAssociatedEvidenceFromCandidates"),
 ("módulo declarado sem provenance perde o limite", [(SF, 'fact.Limits = append(fact.Limits, "the module is declared by the spec, but the result carries no scope provenance: it is not shown to cover this spec\'s delta")', '_ = fact')], P, "TestFactsSeparateAssociatedEvidenceFromCandidates"),
 ("resultado de outro spec não é contado", [(SF, 'otherSpecs++\n\t\t\tcontinue', 'continue')], P, "TestFactsSeparateAssociatedEvidenceFromCandidates"),
 ("head do resultado não é comparado", [(SF, 'case heads[head]:\n\t\treturn "same-head"', 'case heads[head]:\n\t\treturn "other-head"')], P, "TestFactsKeepAResultOfAnotherHead"),
 ("head desconhecido é tratado como igual", [(SF, 'case head == "":\n\t\treturn "unknown-head"', 'case head == "":\n\t\treturn "same-head"')], P, "TestFactsKeepAResultOfAnotherHead"),
 ("resultado de outro head perde a fonte", [(SF, 'Binding: SpecFactBinding{GitHead: r.GitHead, ProvenanceDigest: r.ProvenanceDigest,', 'Binding: SpecFactBinding{ProvenanceDigest: r.ProvenanceDigest,')], P, "TestFactsKeepAResultOfAnotherHead"),
 ("índice com input alterado é fresh", [(SF, 'if len(stale) > 0 {\n\t\t\treturn SpecFactsFreshness{Status: SpecFactsStale', 'if len(stale) > 0 && false {\n\t\t\treturn SpecFactsFreshness{Status: SpecFactsStale')], P, "TestFactsSayHowFreshTheIndexIs"),
 ("índice não publicado é fresh", [(SF, 'return SpecFactsFreshness{Status: SpecFactsUnknown, Reason: view.Reason}', 'return SpecFactsFreshness{Status: SpecFactsFresh}')], P, "TestFactsSayHowFreshTheIndexIs"),
 ("índice stale sem próximo passo", [(SF, 'run `pose index` before relying on these facts', 'ok')], P, "TestFactsSayHowFreshTheIndexIs"),
 ("saída humana imprime candidato como check", [(SC, 'out.Field("facts.candidate", c.Module', 'out.Field("facts.check", c.Module')], C, "TestSpecsFactsLabelsCandidates"),
 ("saída humana sem frescor", [(SC, 'out.Field("facts.freshness", freshness)', '_ = freshness')], C, "TestSpecsFactsLabelsCandidates"),
]

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
        r = subprocess.run(f"cd {work} && go test {pkgs} -run '{run}' -count=1", shell=True, capture_output=True, text=True, timeout=900)
        out = r.stdout + r.stderr
        build_error = 'build failed' in out or 'setup failed' in out
        verdict = 'killed' if r.returncode and not build_error else ('build-error' if build_error else 'survived')
        results.append({'mutation': name, 'tests': run, 'verdict': verdict})
        print(verdict, name, flush=True)
pathlib.Path(args.output).write_text(json.dumps({'mutations': results, 'killed': sum(r['verdict'] == 'killed' for r in results), 'total': len(results)}, indent=2, ensure_ascii=False) + '\n')
