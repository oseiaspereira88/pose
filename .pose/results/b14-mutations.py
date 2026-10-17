"""Production mutations of B14 (spec pose-writer-coordination-and-atomic-publication).

Each mutation reverts one guarantee in a copy-free way (edit, run the named
tests, restore) and must make those tests fail on an assertion, not on a
build error. Run from the repository root: python3 -I .pose/results/b14-mutations.py --output <file.json>
"""
import argparse, json, pathlib, subprocess

parser = argparse.ArgumentParser(description=__doc__)
parser.add_argument('--output', required=True)
args = parser.parse_args()
root = pathlib.Path('.').resolve()
M = [
 ("staging fixo", [("pose-mcp/internal/pose/writer_publication.go", 'tmp, err := os.CreateTemp(dir, "."+filepath.Base(path)+".*.tmp")', 'tmp, err := os.OpenFile(filepath.Join(dir, "."+filepath.Base(path)+".tmp"), os.O_CREATE|os.O_TRUNC|os.O_WRONLY, 0o644)')], "./internal/pose", "TestConcurrentPublishers"),
 ("CAS ignorado", [("pose-mcp/internal/pose/writer_publication.go", "		if found != expect.Digest {", "		if found != expect.Digest && false {")], "./internal/pose", "TestPublicationsKeep|TestTheActionPolicy"),
 ("imutável sobrescrito", [("pose-mcp/internal/pose/writer_publication.go", "		if err := linkFile(name, path); err != nil {", "		if err := os.Rename(name, path); err != nil {")], "./internal/pose", "TestPublicationsKeep"),
 ("staging próprio não limpo", [("pose-mcp/internal/pose/writer_publication.go", "		if !published {\n			os.Remove(name)", "		if !published && false {\n			os.Remove(name)")], "./internal/pose", "TestAFailedPublication|TestPublicationsKeep"),
 ("transfer sem trava de spec", [("pose-mcp/internal/pose/spec_transfer.go", "	specLocks, err := lockTransferSpecs(stores, plan)", "	specLocks, err := func() (func(), error) { return func() {}, nil }()")], "./internal/pose", "TestATransferWaits"),
 ("close sem trava", [("pose-mcp/internal/cli/review_closeout.go", '	if scope.Kind == "spec" {\n		unlock, err := posemodel.LockSpecLifecycle(', '	if scope.Kind == "never" {\n		unlock, err := posemodel.LockSpecLifecycle(')], "./internal/cli", "TestACloseWaits"),
 ("close sem digest julgado", [("pose-mcp/internal/cli/review_closeout.go", "	if judged != \"\" {\n		expect.Digest = judged", "	if judged != \"\" && false {\n		expect.Digest = judged")], "./internal/cli", "TestACloseNever|TestACloseIsBound"),
 ("state sem CAS", [("pose-mcp/internal/cli/state_refresh.go", "pose.PublishAuthoritative, pose.ExpectBytes(preimage, exists))", "pose.PublishAuthoritative, pose.PublishExpectation{Mode: pose.ExpectAny, Digest: string(preimage[:0])})")], "./internal/cli", "TestARefreshNever"),
 ("history como falha genérica", [("pose-mcp/internal/cli/state_refresh.go", "		partial := &stateRefreshPartialError{State: store.StatePath(), Err: err}", "		return failedRefresh(root, opts, key, started, err)\n		partial := &stateRefreshPartialError{State: store.StatePath(), Err: err}")], "./internal/cli", "TestARefreshThatPublished"),
 ("history colado", [("pose-mcp/internal/cli/state.go", "		if last[0] != '\\n' {", "		if last[0] != '\\n' && false {")], "./internal/cli", "TestAHistoryEntry"),
 ("policy sem CAS", [("pose-mcp/internal/pose/principal_keys.go", "PublishAuthoritative, d.read)", "PublishAuthoritative, PublishExpectation{Mode: ExpectAny})")], "./internal/pose", "TestTheActionPolicy"),
 ("start ignora trava de close", [("pose-mcp/internal/pose/start_lock.go", "		if errors.Is(err, os.ErrExist) {\n			return nil, inspectStartLock(root, slug)", "		if errors.Is(err, os.ErrExist) {\n			return func() {}, nil")], "./internal/pose", "TestStartRefusesWhile"),
 ("hook parcial marca pendente", [("pose-mcp/internal/cli/state_hooks.go", "	if errors.As(err, &partial) {\n		return nil\n	}", "	if errors.As(err, &partial) && false {\n		return nil\n	}")], "./internal/cli", "TestAHookRefresh"),
 ("validate não avisa parcial", [("pose-mcp/internal/cli/state.go", "ok && last.Result == \"partial\" {", "ok && last.Result == \"never\" {")], "./internal/cli", "TestAHookRefresh"),
 ("pendente sem CAS", [("pose-mcp/internal/cli/state_hooks.go", "pose.PublishAuthoritative, pose.ExpectBytes(raw, true))\n	if err == nil", "pose.PublishAuthoritative, pose.PublishExpectation{Mode: pose.ExpectAny})\n	if err == nil")], "./internal/cli", "TestEveryStateWriter"),
 ("assess sem CAS", [("pose-mcp/internal/cli/assess.go", "pose.PublishAuthoritative, pose.ExpectBytes(raw, true))", "pose.PublishAuthoritative, pose.PublishExpectation{Mode: pose.ExpectAny})")], "./internal/cli", "TestEveryStateWriter"),
 ("close sem revalidar autorização", [("pose-mcp/internal/cli/review_closeout.go", "	} else if !current.Review.Approved || hasActionRequestPending(current.Diagnostics) {", "	} else if false && (!current.Review.Approved || hasActionRequestPending(current.Diagnostics)) {")], "./internal/cli", "TestACloseRevalidates"),
 ("durabilidade como erro comum", [("pose-mcp/internal/pose/writer_publication.go", "			return &PublishDurabilityError{Path: path, Err: err}", "			return err")], "./internal/pose", "TestAnUnsynced"),
 ("sem sync de arquivo", [("pose-mcp/internal/pose/writer_publication.go", "	if err == nil && class == PublishAuthoritative {\n		if PublishSyncHook != nil {", "	if err == nil && class == \"never\" {\n		if PublishSyncHook != nil {")], "./internal/pose", "TestAuthoritativePublicationsSync"),
 ("sem sync de diretório", [("pose-mcp/internal/pose/writer_publication.go", "	if class == PublishAuthoritative {\n		err := error(nil)", "	if class == \"never\" {\n		err := error(nil)")], "./internal/pose", "TestAuthoritativePublicationsSync"),
 ("imutável sem fallback", [("pose-mcp/internal/pose/writer_publication.go", "				return publishExclusive(path, data, perm, class)", "				return err")], "./internal/pose", "TestAnImmutablePublicationWorks"),
 ("trava sempre start-in-progress", [("pose-mcp/internal/pose/start_lock.go", "		if !strings.HasPrefix(owner.Operation, \"start.\") {\n			e.Code = lifecycleLockHeld", "		if !strings.HasPrefix(owner.Operation, \"start.\") && false {\n			e.Code = lifecycleLockHeld")], "./internal/cli", "TestACloseWaits"),
 ("staging órfão bloqueia closeout", [("pose-mcp/internal/pose/writer_publication.go", "	if !strings.HasPrefix(base, \".\") || !strings.HasSuffix(base, \".tmp\") {\n		return false", "	if true {\n		return false")], "./internal/pose", "TestStagingFiles"),
 ("backfill sem CAS", [("pose-mcp/internal/cli/artifact_integrity.go", "posemodel.PublishAuthoritative, posemodel.ExpectBytes(raw, true))\n		unlock()", "posemodel.PublishAuthoritative, posemodel.PublishExpectation{Mode: posemodel.ExpectAny})\n		unlock()")], "./internal/cli", "TestTheBackfill"),
 ("policies de adoção sem CAS", [("pose-mcp/internal/pose/capability_catalog.go", "PublishAuthoritative, d.reviewRead)", "PublishAuthoritative, PublishExpectation{Mode: ExpectAny})")], "./internal/pose", "TestAdoptionPolicies"),
]
results = []
for name, edits, pkgs, run in M:
    saved = {}
    for f, old, new in edits:
        p = root / f
        cur = p.read_text()
        saved.setdefault(f, cur)
        assert cur.count(old) == 1, (name, f, cur.count(old))
        p.write_text(cur.replace(old, new))
    try:
        r = subprocess.run(f"cd {root}/pose-mcp && go test {pkgs} -run '{run}' -count=1", shell=True, capture_output=True, text=True, timeout=900)
        out = r.stdout + r.stderr
        build_error = 'build failed' in out or 'setup failed' in out
        verdict = 'killed' if r.returncode and not build_error else ('build-error' if build_error else 'survived')
        results.append({'mutation': name, 'tests': run, 'verdict': verdict})
        print(verdict, name)
    finally:
        for f, src in saved.items():
            (root / f).write_text(src)
pathlib.Path(args.output).write_text(json.dumps({'mutations': results, 'killed': sum(r['verdict'] == 'killed' for r in results), 'total': len(results)}, indent=2, ensure_ascii=False) + '\n')
