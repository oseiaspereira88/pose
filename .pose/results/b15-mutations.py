"""Production mutations of B15 (spec pose-index-generation-integrity).

Each mutation reverts one guarantee in a COPY of pose-mcp (the working tree is
never edited), runs the named tests there and must make them fail on an
assertion, not on a build error. Run from the repository root:

    python3 -I .pose/results/b15-mutations.py --output <file.json>
"""
import argparse, json, pathlib, shutil, subprocess, tempfile

parser = argparse.ArgumentParser(description=__doc__)
parser.add_argument('--output', required=True)
parser.add_argument('--only', default='', help='run only the mutations whose name contains this text')
args = parser.parse_args()
root = pathlib.Path('.').resolve()
G = "pose-mcp/internal/pose/index_generation.go"
I = "pose-mcp/internal/cli/index.go"
M = [
 ("commit antes dos nomes legados", [(G, "	if err := indexFault(\"before-legacy\"); err != nil {\n		return receipt, err\n	}\n	for _, f := range desc.Files {\n		if err := PublishFile(filepath.Join(indexDir(s.Root), f.Name)", "	if err := indexFault(\"before-legacy\"); err != nil {\n		return receipt, err\n	}\n	if err := PublishFile(filepath.Join(indexDir(s.Root), indexDescriptorName), descBytes, 0o644, PublishAuthoritative, expect); err != nil {\n		return receipt, err\n	}\n	for _, f := range desc.Files {\n		if err := PublishFile(filepath.Join(indexDir(s.Root), f.Name)")], "./internal/pose", "TestIndexGenerationRetryAfterAStopAtEveryPhase|TestIndexGenerationBlockedOutput"),
 ("leitor não confere o digest do arquivo", [(G, "		if sha256Prefixed(raw) == want {\n			return raw, true, nil", "		if true {\n			return raw, true, nil")], "./internal/pose", "TestIndexGenerationUnverifiable|TestIndexGenerationBlockedOutput"),
 ("id não confere com o conteúdo", [(G, "	if want := indexGenerationPrefix + hex.EncodeToString(sum[:8]); want != d.GenerationID {", "	if want := indexGenerationPrefix + hex.EncodeToString(sum[:8]); false && want != d.GenerationID {")], "./internal/pose", "TestIndexGenerationUnverifiable"),
 ("schema desconhecido aceito", [(G, "	if d.SchemaVersion != IndexGenerationSchemaVersion {", "	if false && d.SchemaVersion != IndexGenerationSchemaVersion {")], "./internal/pose", "TestIndexGenerationUnverifiable"),
 ("campo desconhecido aceito", [(G, "	dec.DisallowUnknownFields()\n", "")], "./internal/pose", "TestIndexGenerationUnverifiable"),
 ("conjunto incompleto aceito", [(G, "	if len(d.Files) != len(IndexGenerationFiles) {\n		return fmt.Errorf(\"the descriptor lists", "	if false && len(d.Files) != len(IndexGenerationFiles) {\n		return fmt.Errorf(\"the descriptor lists")], "./internal/pose", "TestIndexGenerationUnverifiable"),
 ("legado lido como corrente", [(G, "	return nil, IndexGenerationUnavailable, fmt.Errorf(\"pose: %w: the generation descriptor kept changing while %s was read\", ErrIndexGenerationUnavailable, name)", "	raw, err := os.ReadFile(filepath.Join(indexDir(s.Root), name))\n	return raw, IndexGenerationCurrent, err"), (G, "		if !matched {\n			return nil, IndexGenerationUnavailable, fmt.Errorf(\"pose: %w: %s does not match generation %s; run `pose index`\", ErrIndexGenerationUnavailable, name, desc.GenerationID)\n		}", "		_ = matched")], "./internal/pose", "TestIndexGenerationBlockedOutput|TestIndexGenerationCompactDelivery|TestIndexGenerationReaders"),
 ("sem descritor lido como conjunto", [(G, "			return IndexGenerationView{Status: IndexGenerationLegacy, Reason: \"no generation descriptor: the indexes were not published as a set\"}", "			return IndexGenerationView{Status: IndexGenerationCurrent, Files: map[string][]byte{}}")], "./internal/pose", "TestIndexGenerationLegacy"),
 ("bindings de entrada ignorados", [(G, "		if !ok || now != v.Descriptor.Inputs[name] {", "		if !ok && false || now != v.Descriptor.Inputs[name] && false {")], "./internal/pose", "TestIndexGenerationDivergentInput"),
 ("mesma intenção reescreve", [(G, "view.Descriptor.GenerationID == desc.GenerationID && len(view.Diverged) == 0 {", "view.Descriptor.GenerationID == desc.GenerationID && len(view.Diverged) == 0 && false {")], "./internal/pose", "TestIndexGenerationPublishesAndReads"),
 ("nome legado divergente ignorado no reenvio", [(G, "view.Descriptor.GenerationID == desc.GenerationID && len(view.Diverged) == 0 {", "view.Descriptor.GenerationID == desc.GenerationID {")], "./internal/pose", "TestIndexGenerationRepublishingRepairs"),
 ("divergência não informada", [(G, "				diverged = append(diverged, f.Name)", "				_ = diverged")], "./internal/pose", "TestIndexGenerationNamesTheLegacyNames"),
 ("cópia imutável sobrescrita", [(G, "PublishFile(filepath.Join(dir, f.Name), outputs[f.Name], 0o644, PublishDerived, PublishExpectation{Mode: ExpectAbsent})", "PublishFile(filepath.Join(dir, f.Name), outputs[f.Name], 0o644, PublishDerived, PublishExpectation{Mode: ExpectAny})")], "./internal/pose", "TestIndexGenerationRefusesToOverwrite"),
 ("sem releitura do descritor", [(G, "		if againErr == nil && again.GenerationID == desc.GenerationID && string(againRaw) == string(raw) {", "		if (againErr == nil && again.GenerationID == desc.GenerationID && string(againRaw) == string(raw)) || true {")], "./internal/pose", "TestIndexGenerationReaderStartsOver"),
 ("ReadIndex sem releitura do descritor", [(G, "		if againErr != nil || again.GenerationID != desc.GenerationID || string(againRaw) != string(descRaw) {\n			continue\n		}", "		_, _, _ = again, againRaw, againErr\n		_ = descRaw")], "./internal/pose", "TestIndexGenerationReaderStartsOver"),
 ("retenção remove a geração corrente", [(G, "		if !indexGenerationIDRE.MatchString(e.Name()) || live[e.Name()] {", "		if !indexGenerationIDRE.MatchString(e.Name()) {")], "./internal/pose", "TestIndexGenerationRetention"),
 ("retenção ignora a carência", [(G, "		if i < indexGenerationKeep-1 || time.Since(g.mod) < indexGenerationGrace {", "		if i < indexGenerationKeep-1 {")], "./internal/pose", "TestIndexGenerationRetention"),
 ("retenção sem limite", [(G, "		if i < indexGenerationKeep-1 || time.Since(g.mod) < indexGenerationGrace {", "		if i < 0 || g.mod.IsZero() {")], "./internal/pose", "TestIndexGenerationRetention"),
 ("retenção remove o que não é geração", [(G, "		if !indexGenerationIDRE.MatchString(e.Name()) || live[e.Name()] {", "		if live[e.Name()] {"), (G, "		if err != nil || !info.IsDir() || info.Mode()&os.ModeSymlink != 0 {\n			continue\n		}", "		if err != nil {\n			continue\n		}")], "./internal/pose", "TestIndexGenerationRetention"),
 ("falha de retenção vira falha de publicação", [(G, "	receipt.CleanupError = s.pruneIndexGenerations(desc.GenerationID)\n	return receipt, nil\n}\n\n// ReplaceIndex", "	if cleanup := s.pruneIndexGenerations(desc.GenerationID); cleanup != nil {\n		return receipt, cleanup\n	}\n	return receipt, nil\n}\n\n// ReplaceIndex")], "./internal/pose", "TestIndexGenerationFailedCleanup"),
 ("ReplaceIndex sobre conjunto inválido", [(G, "	if view.Status == IndexGenerationUnavailable {\n		return IndexGenerationReceipt{}, fmt.Errorf(\"pose: %w: %s\", ErrIndexGenerationUnavailable, view.Reason)\n	}", "	if false {\n		return IndexGenerationReceipt{}, nil\n	}")], "./internal/pose", "TestIndexGenerationReplaceIndex"),
 ("pose index escreve arquivo a arquivo", [(I, "	receipt, e := store.PublishIndexGeneration(encoded, posepkg.IndexInputBindings(root))\n	if e != nil {", "	var e error\n	for name, b := range encoded {\n		if e = writeAtomic(filepath.Join(dir, name), b, 0o644); e != nil {\n			break\n		}\n	}\n	receipt := posepkg.IndexGenerationReceipt{}\n	if e != nil {")], "./internal/cli", "TestIndexPublishes|TestIndexWithABlocked|TestIndexJSONReports"),
 ("--verify aceita conjunto velho", [(I, "	if len(stale) > 0 {\n		out.RecordField(\"index_generation_stale_inputs\"", "	if len(stale) > 0 && false {\n		out.RecordField(\"index_generation_stale_inputs\"")], "./internal/cli", "TestIndexVerifyNames"),
 ("--verify ignora divergência", [(I, "	if len(view.Diverged) > 0 {\n		out.RecordField(\"index_generation_diverged\"", "	if len(view.Diverged) > 0 && false {\n		out.RecordField(\"index_generation_diverged\"")], "./internal/cli", "TestIndexVerifyReportsAHandEdited"),
 ("refresh escreve o arquivo sozinho", [(I, "	receipt, _, err := posepkg.Store{Root: root}.ReplaceIndexOrLegacy(\"delivery-integrity.json\", data)", "	var receipt posepkg.IndexGenerationReceipt\n	err := os.WriteFile(filepath.Join(root, \".pose\", \"indexes\", \"delivery-integrity.json\"), data, 0o644)")], "./internal/cli", "TestRefreshDeliveryIndex"),
 ("grafo de entrega lido sem conferir", [("pose-mcp/internal/pose/delivery_integrity.go", "	raw, _, err := s.ReadIndex(\"delivery-integrity.json\")\n	if errors.Is(err, ErrIndexGenerationUnavailable) {\n		return DeliveryIntegrityGraph{}, err\n	}", "	raw, err := os.ReadFile(filepath.Join(s.Root, \".pose\", \"indexes\", \"delivery-integrity.json\"))\n	_ = errors.Is")], "./internal/pose ./internal/mcpserver", "TestIndexGenerationCompactDelivery|ReadsAPublishedGeneration"),
 ("mapa de componentes lido sem conferir", [("pose-mcp/internal/pose/review_plan.go", "	if errors.Is(mapErr, ErrIndexGenerationUnavailable) {\n		return nil, fmt.Errorf(\"review component map is unavailable: %w\", mapErr)\n	}", "")], "./internal/pose", "TestIndexGenerationReadersRefuse"),
 ("snapshot de obrigações liga índice inválido", [("pose-mcp/internal/pose/obligation_projection.go", "		snap.IndexDigest = \"unavailable\"", "		snap.IndexDigest = \"sha256:00\"")], "./internal/pose", "TestIndexGenerationReadersRefuse"),
 ("plano sem aviso de caminhos observados", [("pose-mcp/internal/pose/review_plan.go", "		context.Warnings = append(context.Warnings, \"observed paths were not used: \"+err.Error())\n", "")], "./internal/pose", "TestIndexGenerationReadersRefuse"),
 ("suggest usa mapa inválido", [("pose-mcp/internal/cli/suggest.go", "	raw, _, e := posepkg.Store{Root: root}.ReadIndex(\"repo-map.json\")", "	_ = posepkg.Store{}\n	raw, e := os.ReadFile(filepath.Join(root, \".pose\", \"indexes\", \"repo-map.json\"))")], "./internal/cli", "TestSuggestDoesNotInfer"),
 ("descritor sincronizado como derivado", [(G, "descBytes, 0o644, PublishAuthoritative, expect)", "descBytes, 0o644, PublishDerived, expect)")], "./internal/pose", "TestIndexGenerationCommitSyncs"),
 ("descritor sem CAS", [(G, "	expect := ExpectBytes(prevRaw, prevExists)", "	expect := PublishExpectation{Mode: ExpectAny}\n	_ = prevRaw")], "./internal/pose", "TestIndexGenerationConcurrentPublishersCommit"),
 ("descritor distribuído pelo update", [("pose-mcp/internal/scaffold/distpolicy/distpolicy.go", "	if parts[1] == \"indexes\" && len(parts) == 3 && parts[2] == \"generation.json\" {\n		return false\n	}", "")], "./internal/scaffold/distpolicy", "TestIndexGenerationDescriptorIsNever"),
 ("publicação sem trava", [(G, "	unlock, err := lockIndexGeneration(s.Root, \"index.publish\")\n	if err != nil {\n		return IndexGenerationReceipt{}, err\n	}\n	defer unlock()", "")], "./internal/pose", "TestIndexGenerationPublishersNeverAccept"),
 ("trava de dono desconhecido quebrada", [(G, "		if busy.State != string(LockOwnerDead) || attempt > 0 {", "		if busy.State == string(LockOwnerActive) || attempt > 0 {"), (G, "	if again.State != string(LockOwnerDead) || again.Owner == nil || seen.Owner == nil || *again.Owner != *seen.Owner {", "	if false && again.State == \"\" && seen != nil {")], "./internal/pose", "TestIndexGenerationLockIsNeverBroken"),
 ("trava de dono morto não recuperada", [(G, "		if busy.State != string(LockOwnerDead) || attempt > 0 {", "		if true {")], "./internal/pose", "TestIndexGenerationLockOfADeadOwner"),
 ("ReplaceIndex sem trava", [(G, "	unlock, err := lockIndexGeneration(s.Root, \"index.replace\")\n	if err != nil {\n		return IndexGenerationReceipt{}, err\n	}\n	defer unlock()", "")], "./internal/pose", "TestIndexGenerationReplaceIndexNeverReverts"),
 ("sem marcador de primeira publicação", [(G, "	if !prevExists {\n		if err := PublishFile(filepath.Join(indexDir(s.Root), indexPendingName)", "	if false {\n		if err := PublishFile(filepath.Join(indexDir(s.Root), indexPendingName)")], "./internal/pose", "TestIndexGenerationInterruptedFirstPublication"),
 ("leitor ignora o marcador", [(G, "func indexPending(root string) bool {\n", "func indexPending(root string) bool {\n	if true {\n		return false\n	}\n")], "./internal/pose", "TestIndexGenerationInterruptedFirstPublication"),
 ("plano sem geração fixada", [("pose-mcp/internal/pose/review_plan.go", "	// One generation for everything the plan reads from the indexes, unless the\n	// operation that asks for the plan already fixed one.\n	if s.indexPin == nil {\n		s = s.PinIndexes()\n	}\n", ""), ("pose-mcp/internal/pose/review_plan.go", "	// One generation for the whole plan: the component map and the observed\n	// paths are read from the same set, even if a publication commits meanwhile.\n	if s.indexPin == nil {\n		s = s.PinIndexes()\n	}\n", "")], "./internal/pose", "TestIndexGenerationAReviewPlanReadsTheSetOnce"),
 ("ReadIndex ignora a geração fixada", [(G, "	if pin := s.indexPin; pin != nil {", "	if pin := s.indexPin; pin != nil && false {")], "./internal/pose", "TestIndexGenerationPinnedStore"),
 ("retenção não lê o descritor vivo", [(G, "	if desc, _, exists, readErr := readIndexDescriptor(s.Root); readErr == nil && exists {\n		live[desc.GenerationID] = true\n	}", "")], "./internal/pose", "TestIndexGenerationRetentionProtectsTheDescriptors"),
 ("retenção segue link na base", [(G, "	if err := ensureOwnedGenerationsDir(s.Root); err != nil {\n		return err\n	}\n	base := filepath.Join(indexDir(s.Root), indexGenerationsDir)\n	entries", "	base := filepath.Join(indexDir(s.Root), indexGenerationsDir)\n	entries")], "./internal/pose", "TestIndexGenerationRetentionNeverFollowsALinked"),
 ("publicação escreve por link na base", [(G, "	if err := ensureOwnedGenerationsDir(s.Root); err != nil {\n		return receipt, err\n	}", "")], "./internal/pose", "TestIndexGenerationRetentionNeverFollowsALinked"),
 ("descritor com bytes depois do valor", [(G, "	if _, err := dec.Token(); err != io.EOF {", "	if _, err := dec.Token(); err != io.EOF && false {")], "./internal/pose", "TestIndexGenerationDescriptorWithTrailingBytes"),
 ("CLI promete a geração anterior após commit", [(I, "		case receipt.Committed:", "		case false:")], "./internal/cli", "TestIndexAnUnsyncedCommit"),
 ("CLI sem aviso de trava", [(I, "		case errors.Is(e, posepkg.ErrIndexGenerationBusy):", "		case false:")], "./internal/cli", "TestIndexWithAnotherPublication"),
 ("refresh descarta a retenção", [(I, "		if receipt.CleanupError != nil {\n			render(io.Discard, stderr)", "		if false {\n			render(io.Discard, stderr)")], "./internal/cli", "TestRefreshDeliveryIndexReportsAFailedRetention"),
 ("arquivos ocultos distribuídos pelo update", [("pose-mcp/internal/scaffold/distpolicy/distpolicy.go", "	if parts[1] == \"indexes\" && len(parts) == 3 && strings.HasPrefix(parts[2], \".\") {\n		return false\n	}", "")], "./internal/scaffold/distpolicy", "TestIndexGenerationDescriptorIsNever"),
 ("pin legado atravessa a primeira publicação", [(G, "		if indexPending(s.Root) || indexDescriptorExists(s.Root) {\n			return nil, IndexGenerationUnavailable, fmt.Errorf(\"pose: %w: the indexes were published as a set while this operation was reading them; run it again\", ErrIndexGenerationUnavailable)\n		}\n", ""), (G, "		if indexPending(s.Root) || indexDescriptorExists(s.Root) {\n			return nil, IndexGenerationUnavailable, fmt.Errorf(\"pose: %w: the indexes were published as a set while %s was read; run the operation again\", ErrIndexGenerationUnavailable, name)\n		}\n", "")], "./internal/pose", "TestIndexGenerationALegacyPin"),
 ("substituição sem vínculo ao descritor lido", [(G, "	if basedOn != nil {\n		if !prevExists", "	if false {\n		if !prevExists")], "./internal/pose", "TestIndexGenerationReplaceIndexIsRefused"),
 ("refresh nega o commit efetivo", [(I, "	case receipt.Committed:\n		render(io.Discard, stderr)", "	case false:\n		render(io.Discard, stderr)")], "./internal/cli", "TestRefreshDeliveryIndexNamesACommitted"),
 ("substituição legada fora da trava", [(G, "	unlock, err := lockIndexGeneration(s.Root, \"index.replace\")\n	if err != nil {\n		return IndexGenerationReceipt{}, false, err\n	}\n	defer unlock()", "")], "./internal/pose", "TestIndexGenerationALegacyReplacement"),
 ("bundle sem geração fixada", [("pose-mcp/internal/pose/review_bundle.go", "	// One index generation for the plan and for the evidence read after it.\n	if s.indexPin == nil {\n		s = s.PinIndexes()\n	}\n", "")], "./internal/pose", "TestIndexGenerationABundleFixes"),
]
results = []
for name, edits, pkgs, run in M:
    if args.only and args.only not in name:
        continue
    with tempfile.TemporaryDirectory(prefix="b15-mut-") as tmp:
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
