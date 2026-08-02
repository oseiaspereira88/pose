# Harne8 Technical Debt & Governed Backlog Report

> **Gerado por**: POSE Technical Debt Engine (`pose assess tech-debt`)
> **Data de Avaliação**: 2026-08-02T17:09:55Z
> **Baseline Commit**: 3eaa7db815cf

---

## 1. Resumo Executivo da Dívida Técnica

- **Total de Marcadores Encontrados**: 77
- **TODOs**: 38 | **FIXMEs**: 13 | **Panics**: 4 | **Stubs**: 22
- **Dívidas Cobertas por Specs/Follow-ups**: 0
- **Dívidas Não-cobertas (Pendentes de Atribuição)**: 77
- **Recomendações**: 31 Follow-ups | 46 Specs | 0 Roadmaps

---

## 2. Detalhamento das Ocorrências e Links de Código-Fonte

| ID | Marcador | Componente | Arquivo e Linha | Trecho do Código | Recomendação POSE |
|---|---|---|---|---|---|
| DEBT-001 | `STUB` | `mcp-enforce` | [gate_test.go:12](file:///home/go/IdeaProjects/harne8/pose-dist/mcp-enforce/gate_test.go#L12) | `// opaFixture starts a minimal OPA-compatible HTTP stub that return...` | 📌 Sugere Follow-up |
| DEBT-002 | `STUB` | `mcp-enforce` | [gate_test.go:79](file:///home/go/IdeaProjects/harne8/pose-dist/mcp-enforce/gate_test.go#L79) | `stub := opaFixture(t, http.StatusOK, map[string]any{` | 📌 Sugere Follow-up |
| DEBT-003 | `STUB` | `mcp-enforce` | [gate_test.go:82](file:///home/go/IdeaProjects/harne8/pose-dist/mcp-enforce/gate_test.go#L82) | `g := NewPolicyGate(PolicyConfig{OPAURL: stub.URL, HTTPClient: stub....` | 📌 Sugere Follow-up |
| DEBT-004 | `STUB` | `mcp-enforce` | [gate_test.go:93](file:///home/go/IdeaProjects/harne8/pose-dist/mcp-enforce/gate_test.go#L93) | `stub := opaFixture(t, http.StatusOK, map[string]any{` | 📌 Sugere Follow-up |
| DEBT-005 | `STUB` | `mcp-enforce` | [gate_test.go:96](file:///home/go/IdeaProjects/harne8/pose-dist/mcp-enforce/gate_test.go#L96) | `g := NewPolicyGate(PolicyConfig{OPAURL: stub.URL, HTTPClient: stub....` | 📌 Sugere Follow-up |
| DEBT-006 | `STUB` | `mcp-enforce` | [gate_test.go:110](file:///home/go/IdeaProjects/harne8/pose-dist/mcp-enforce/gate_test.go#L110) | `stub := opaFixture(t, http.StatusOK, map[string]any{"result": nil})` | 📌 Sugere Follow-up |
| DEBT-007 | `STUB` | `mcp-enforce` | [gate_test.go:111](file:///home/go/IdeaProjects/harne8/pose-dist/mcp-enforce/gate_test.go#L111) | `g := NewPolicyGate(PolicyConfig{OPAURL: stub.URL, HTTPClient: stub....` | 📌 Sugere Follow-up |
| DEBT-008 | `STUB` | `mcp-enforce` | [gate_test.go:125](file:///home/go/IdeaProjects/harne8/pose-dist/mcp-enforce/gate_test.go#L125) | `stub := opaFixture(t, http.StatusInternalServerError, nil)` | 📌 Sugere Follow-up |
| DEBT-009 | `STUB` | `mcp-enforce` | [gate_test.go:126](file:///home/go/IdeaProjects/harne8/pose-dist/mcp-enforce/gate_test.go#L126) | `g := NewPolicyGate(PolicyConfig{OPAURL: stub.URL, HTTPClient: stub....` | 📌 Sugere Follow-up |
| DEBT-010 | `TODO` | `pose-mcp-internal-cli` | [assess.go:578](file:///home/go/IdeaProjects/harne8/pose-dist/pose-mcp/internal/cli/assess.go#L578) | `fmt.Fprintf(stdout, "  - %-30s LOC: %d (prod) / %d (test) \| Debt T...` | 📌 Sugere Follow-up |
| DEBT-011 | `TODO` | `pose-mcp-internal-cli` | [assess.go:579](file:///home/go/IdeaProjects/harne8/pose-dist/pose-mcp/internal/cli/assess.go#L579) | `res.ComponentSlug, res.Metrics.LOCProduction, res.Metrics.LOCTests,...` | 📌 Sugere Follow-up |
| DEBT-012 | `TODO` | `pose-mcp-internal-cli` | [assess.go:607](file:///home/go/IdeaProjects/harne8/pose-dist/pose-mcp/internal/cli/assess.go#L607) | `- saude_de_codigo: TODOs=0 FIXMEs=0 panics=0 stubs=0` | 📌 Sugere Follow-up |
| DEBT-013 | `TODO` | `pose-mcp-internal-cli` | [assess.go:703](file:///home/go/IdeaProjects/harne8/pose-dist/pose-mcp/internal/cli/assess.go#L703) | `fmt.Fprintf(stdout, "  - Total Debt Markers: %d (TODOs: %d, FIXMEs:...` | 📌 Sugere Follow-up |
| DEBT-014 | `TODO` | `pose-mcp-internal-cli` | [assess.go:704](file:///home/go/IdeaProjects/harne8/pose-dist/pose-mcp/internal/cli/assess.go#L704) | `report.Summary.TotalMarkers, report.Summary.TODOs, report.Summary.F...` | 📌 Sugere Follow-up |
| DEBT-015 | `TODO` | `pose-mcp-internal-cli` | [cli.go:395](file:///home/go/IdeaProjects/harne8/pose-dist/pose-mcp/internal/cli/cli.go#L395) | `Todos os comandos executam no binário Go, sem fallbacks Bash ou Py...` | 📌 Sugere Follow-up |
| DEBT-016 | `TODO` | `pose-mcp-internal-cli` | [doctor.go:497](file:///home/go/IdeaProjects/harne8/pose-dist/pose-mcp/internal/cli/doctor.go#L497) | `fmt.Fprintln(stdout, cliText(locale, "Result: SUCCESS — recheck c...` | 📌 Sugere Follow-up |
| DEBT-017 | `STUB` | `pose-mcp-internal-cli` | [extension_test.go:74](file:///home/go/IdeaProjects/harne8/pose-dist/pose-mcp/internal/cli/extension_test.go#L74) | `// fakeSignedInstall stubs signature verification to always succeed...` | 📌 Sugere Follow-up |
| DEBT-018 | `PANIC` | `pose-mcp-internal-cli` | [state_hooks_test.go:36](file:///home/go/IdeaProjects/harne8/pose-dist/pose-mcp/internal/cli/state_hooks_test.go#L36) | `panic("boom")` | 📌 Sugere Follow-up |
| DEBT-019 | `STUB` | `pose-mcp-internal-mcpserver` | [policy_test.go:18](file:///home/go/IdeaProjects/harne8/pose-dist/pose-mcp/internal/mcpserver/policy_test.go#L18) | `// opaFixture starts a minimal OPA-compatible HTTP stub that return...` | 📌 Sugere Follow-up |
| DEBT-020 | `STUB` | `pose-mcp-internal-mcpserver` | [policy_test.go:40](file:///home/go/IdeaProjects/harne8/pose-dist/pose-mcp/internal/mcpserver/policy_test.go#L40) | `stub := opaFixture(t, http.StatusOK, map[string]any{` | 📌 Sugere Follow-up |
| DEBT-021 | `STUB` | `pose-mcp-internal-mcpserver` | [policy_test.go:46](file:///home/go/IdeaProjects/harne8/pose-dist/pose-mcp/internal/mcpserver/policy_test.go#L46) | `g := NewPolicyGate(PolicyConfig{OPAURL: stub.URL, HTTPClient: stub....` | 📌 Sugere Follow-up |
| DEBT-022 | `TODO` | `pose-mcp-internal-mcpserver` | [server.go:1968](file:///home/go/IdeaProjects/harne8/pose-dist/pose-mcp/internal/mcpserver/server.go#L1968) | `"technical debt markers (TODO, FIXME, stub, panic), detected langua...` | 📌 Sugere Follow-up |
| DEBT-023 | `TODO` | `pose-mcp-internal-mcpserver` | [server.go:2045](file:///home/go/IdeaProjects/harne8/pose-dist/pose-mcp/internal/mcpserver/server.go#L2045) | `"description": "Audit technical debt markers (TODO, FIXME, stub, pa...` | 📌 Sugere Follow-up |
| DEBT-024 | `TODO` | `pose-mcp-internal-mcpserver-testdata` | [tool-catalog.golden.json:875](file:///home/go/IdeaProjects/harne8/pose-dist/pose-mcp/internal/mcpserver/testdata/tool-catalog.golden.json#L875) | `"description": "Perform a deep discovery audit of a repository comp...` | 📌 Sugere Follow-up |
| DEBT-025 | `TODO` | `pose-mcp-internal-mcpserver-testdata` | [tool-catalog.golden.json:954](file:///home/go/IdeaProjects/harne8/pose-dist/pose-mcp/internal/mcpserver/testdata/tool-catalog.golden.json#L954) | `"description": "Audit technical debt markers (TODO, FIXME, stub, pa...` | 📌 Sugere Follow-up |
| DEBT-026 | `STUB` | `pose-mcp-internal-mcpserver` | [validate_orchestration_test.go:36](file:///home/go/IdeaProjects/harne8/pose-dist/pose-mcp/internal/mcpserver/validate_orchestration_test.go#L36) | `// stubExecutor counts Submit calls and returns a fixed execution i...` | 📌 Sugere Follow-up |
| DEBT-027 | `STUB` | `pose-mcp-internal-mcpserver` | [validate_orchestration_test.go:38](file:///home/go/IdeaProjects/harne8/pose-dist/pose-mcp/internal/mcpserver/validate_orchestration_test.go#L38) | `type stubExecutor struct {` | 📌 Sugere Follow-up |
| DEBT-028 | `STUB` | `pose-mcp-internal-mcpserver` | [validate_orchestration_test.go:44](file:///home/go/IdeaProjects/harne8/pose-dist/pose-mcp/internal/mcpserver/validate_orchestration_test.go#L44) | `func (s *stubExecutor) Submit(_ context.Context, _ ApprovedValidati...` | 📌 Sugere Follow-up |
| DEBT-029 | `STUB` | `pose-mcp-internal-mcpserver` | [validate_orchestration_test.go:109](file:///home/go/IdeaProjects/harne8/pose-dist/pose-mcp/internal/mcpserver/validate_orchestration_test.go#L109) | `exec := &stubExecutor{id: "exec-123"}` | 📌 Sugere Follow-up |
| DEBT-030 | `STUB` | `pose-mcp-internal-mcpserver` | [validate_orchestration_test.go:142](file:///home/go/IdeaProjects/harne8/pose-dist/pose-mcp/internal/mcpserver/validate_orchestration_test.go#L142) | `if _, err := o.submit(context.Background(), req.ID, &stubExecutor{i...` | 📌 Sugere Follow-up |
| DEBT-031 | `TODO` | `pose-mcp-internal-pose` | [discovery.go:25](file:///home/go/IdeaProjects/harne8/pose-dist/pose-mcp/internal/pose/discovery.go#L25) | `TODOs  int `json:"todos"`` | 📜 Sugere Spec |
| DEBT-032 | `FIXME` | `pose-mcp-internal-pose` | [discovery.go:26](file:///home/go/IdeaProjects/harne8/pose-dist/pose-mcp/internal/pose/discovery.go#L26) | `FIXMEs int `json:"fixmes"`` | 📜 Sugere Spec |
| DEBT-033 | `STUB` | `pose-mcp-internal-pose` | [discovery.go:28](file:///home/go/IdeaProjects/harne8/pose-dist/pose-mcp/internal/pose/discovery.go#L28) | `Stubs  int `json:"stubs"`` | 📜 Sugere Spec |
| DEBT-034 | `TODO` | `pose-mcp-internal-pose` | [discovery.go:258](file:///home/go/IdeaProjects/harne8/pose-dist/pose-mcp/internal/pose/discovery.go#L258) | `state.TechnicalDebt.TODOs += debt.TODOs` | 📜 Sugere Spec |
| DEBT-035 | `FIXME` | `pose-mcp-internal-pose` | [discovery.go:259](file:///home/go/IdeaProjects/harne8/pose-dist/pose-mcp/internal/pose/discovery.go#L259) | `state.TechnicalDebt.FIXMEs += debt.FIXMEs` | 📜 Sugere Spec |
| DEBT-036 | `TODO` | `pose-mcp-internal-pose` | [discovery.go:273](file:///home/go/IdeaProjects/harne8/pose-dist/pose-mcp/internal/pose/discovery.go#L273) | `score -= float64(state.TechnicalDebt.TODOs) * 0.005` | 📜 Sugere Spec |
| DEBT-037 | `FIXME` | `pose-mcp-internal-pose` | [discovery.go:274](file:///home/go/IdeaProjects/harne8/pose-dist/pose-mcp/internal/pose/discovery.go#L274) | `score -= float64(state.TechnicalDebt.FIXMEs) * 0.010` | 📜 Sugere Spec |
| DEBT-038 | `TODO` | `pose-mcp-internal-pose` | [discovery.go:323](file:///home/go/IdeaProjects/harne8/pose-dist/pose-mcp/internal/pose/discovery.go#L323) | `if strings.Contains(upper, "TODO") {` | 📜 Sugere Spec |
| DEBT-039 | `TODO` | `pose-mcp-internal-pose` | [discovery.go:324](file:///home/go/IdeaProjects/harne8/pose-dist/pose-mcp/internal/pose/discovery.go#L324) | `debt.TODOs++` | 📜 Sugere Spec |
| DEBT-040 | `FIXME` | `pose-mcp-internal-pose` | [discovery.go:326](file:///home/go/IdeaProjects/harne8/pose-dist/pose-mcp/internal/pose/discovery.go#L326) | `if strings.Contains(upper, "FIXME") {` | 📜 Sugere Spec |
| DEBT-041 | `FIXME` | `pose-mcp-internal-pose` | [discovery.go:327](file:///home/go/IdeaProjects/harne8/pose-dist/pose-mcp/internal/pose/discovery.go#L327) | `debt.FIXMEs++` | 📜 Sugere Spec |
| DEBT-042 | `PANIC` | `pose-mcp-internal-pose` | [discovery.go:329](file:///home/go/IdeaProjects/harne8/pose-dist/pose-mcp/internal/pose/discovery.go#L329) | `if strings.Contains(line, "panic(") {` | 📜 Sugere Spec |
| DEBT-043 | `STUB` | `pose-mcp-internal-pose` | [discovery.go:332](file:///home/go/IdeaProjects/harne8/pose-dist/pose-mcp/internal/pose/discovery.go#L332) | `if strings.Contains(line, "stub") \|\| strings.Contains(line, "unim...` | 📜 Sugere Spec |
| DEBT-044 | `TODO` | `pose-mcp-internal-pose` | [discovery.go:384](file:///home/go/IdeaProjects/harne8/pose-dist/pose-mcp/internal/pose/discovery.go#L384) | `> **Saúde de Código**: %d TODOs \| %d FIXMEs \| %d Panics \| %d S...` | 📜 Sugere Spec |
| DEBT-045 | `TODO` | `pose-mcp-internal-pose` | [discovery.go:399](file:///home/go/IdeaProjects/harne8/pose-dist/pose-mcp/internal/pose/discovery.go#L399) | `- **TODOs**: %d` | 📜 Sugere Spec |
| DEBT-046 | `FIXME` | `pose-mcp-internal-pose` | [discovery.go:400](file:///home/go/IdeaProjects/harne8/pose-dist/pose-mcp/internal/pose/discovery.go#L400) | `- **FIXMEs**: %d` | 📜 Sugere Spec |
| DEBT-047 | `TODO` | `pose-mcp-internal-pose` | [discovery.go:403](file:///home/go/IdeaProjects/harne8/pose-dist/pose-mcp/internal/pose/discovery.go#L403) | ``, state.ComponentSlug, state.Path, state.Path, state.DiscoveredAt,...` | 📜 Sugere Spec |
| DEBT-048 | `TODO` | `pose-mcp-internal-pose` | [discovery.go:431](file:///home/go/IdeaProjects/harne8/pose-dist/pose-mcp/internal/pose/discovery.go#L431) | `var totalTODOs, totalFIXMEs, totalPanics, totalStubs int` | 📜 Sugere Spec |
| DEBT-049 | `TODO` | `pose-mcp-internal-pose` | [discovery.go:438](file:///home/go/IdeaProjects/harne8/pose-dist/pose-mcp/internal/pose/discovery.go#L438) | `totalTODOs += st.TechnicalDebt.TODOs` | 📜 Sugere Spec |
| DEBT-050 | `FIXME` | `pose-mcp-internal-pose` | [discovery.go:439](file:///home/go/IdeaProjects/harne8/pose-dist/pose-mcp/internal/pose/discovery.go#L439) | `totalFIXMEs += st.TechnicalDebt.FIXMEs` | 📜 Sugere Spec |
| DEBT-051 | `TODO` | `pose-mcp-internal-pose` | [discovery.go:487](file:///home/go/IdeaProjects/harne8/pose-dist/pose-mcp/internal/pose/discovery.go#L487) | `- **Dívidas Técnicas em Aberto**: %d TODOs \| %d FIXMEs \| %d Pan...` | 📜 Sugere Spec |
| DEBT-052 | `TODO` | `pose-mcp-internal-pose` | [discovery.go:495](file:///home/go/IdeaProjects/harne8/pose-dist/pose-mcp/internal/pose/discovery.go#L495) | `\| # \| Componente Slug \| Caminho do Módulo \| Linguagens \| LOC ...` | 📜 Sugere Spec |
| DEBT-053 | `TODO` | `pose-mcp-internal-pose` | [discovery.go:497](file:///home/go/IdeaProjects/harne8/pose-dist/pose-mcp/internal/pose/discovery.go#L497) | ``, now, commit, len(states), totalProd, totalTests, totalProd+total...` | 📜 Sugere Spec |
| DEBT-054 | `TODO` | `pose-mcp-internal-pose` | [discovery.go:505](file:///home/go/IdeaProjects/harne8/pose-dist/pose-mcp/internal/pose/discovery.go#L505) | `i+1, st.ComponentSlug, st.Path, langs, st.Metrics.LOCProduction, st...` | 📜 Sugere Spec |
| DEBT-055 | `TODO` | `pose-mcp-internal-pose` | [discovery_test.go:19](file:///home/go/IdeaProjects/harne8/pose-dist/pose-mcp/internal/pose/discovery_test.go#L19) | `code := "package main\n\nfunc main() {\n  // TODO: test\n  println(...` | 📜 Sugere Spec |
| DEBT-056 | `TODO` | `pose-mcp-internal-pose` | [discovery_test.go:36](file:///home/go/IdeaProjects/harne8/pose-dist/pose-mcp/internal/pose/discovery_test.go#L36) | `if state.TechnicalDebt.TODOs != 1 {` | 📜 Sugere Spec |
| DEBT-057 | `TODO` | `pose-mcp-internal-pose` | [discovery_test.go:37](file:///home/go/IdeaProjects/harne8/pose-dist/pose-mcp/internal/pose/discovery_test.go#L37) | `t.Errorf("expected 1 TODO, got %d", state.TechnicalDebt.TODOs)` | 📜 Sugere Spec |
| DEBT-058 | `TODO` | `pose-mcp-internal-pose` | [techdebt.go:16](file:///home/go/IdeaProjects/harne8/pose-dist/pose-mcp/internal/pose/techdebt.go#L16) | `Marker         string `json:"marker"` // TODO, FIXME, HACK, PANIC, ...` | 📜 Sugere Spec |
| DEBT-059 | `TODO` | `pose-mcp-internal-pose` | [techdebt.go:29](file:///home/go/IdeaProjects/harne8/pose-dist/pose-mcp/internal/pose/techdebt.go#L29) | `TODOs                int `json:"todos"`` | 📜 Sugere Spec |
| DEBT-060 | `FIXME` | `pose-mcp-internal-pose` | [techdebt.go:30](file:///home/go/IdeaProjects/harne8/pose-dist/pose-mcp/internal/pose/techdebt.go#L30) | `FIXMEs               int `json:"fixmes"`` | 📜 Sugere Spec |
| DEBT-061 | `STUB` | `pose-mcp-internal-pose` | [techdebt.go:32](file:///home/go/IdeaProjects/harne8/pose-dist/pose-mcp/internal/pose/techdebt.go#L32) | `Stubs                int `json:"stubs"`` | 📜 Sugere Spec |
| DEBT-062 | `TODO` | `pose-mcp-internal-pose` | [techdebt.go:112](file:///home/go/IdeaProjects/harne8/pose-dist/pose-mcp/internal/pose/techdebt.go#L112) | `case "TODO":` | 📜 Sugere Spec |
| DEBT-063 | `TODO` | `pose-mcp-internal-pose` | [techdebt.go:113](file:///home/go/IdeaProjects/harne8/pose-dist/pose-mcp/internal/pose/techdebt.go#L113) | `summary.TODOs++` | 📜 Sugere Spec |
| DEBT-064 | `FIXME` | `pose-mcp-internal-pose` | [techdebt.go:114](file:///home/go/IdeaProjects/harne8/pose-dist/pose-mcp/internal/pose/techdebt.go#L114) | `case "FIXME":` | 📜 Sugere Spec |
| DEBT-065 | `FIXME` | `pose-mcp-internal-pose` | [techdebt.go:115](file:///home/go/IdeaProjects/harne8/pose-dist/pose-mcp/internal/pose/techdebt.go#L115) | `summary.FIXMEs++` | 📜 Sugere Spec |
| DEBT-066 | `FIXME` | `pose-mcp-internal-pose` | [techdebt.go:133](file:///home/go/IdeaProjects/harne8/pose-dist/pose-mcp/internal/pose/techdebt.go#L133) | `} else if item.Marker == "FIXME" \|\| item.Marker == "PANIC" {` | 📜 Sugere Spec |
| DEBT-067 | `TODO` | `pose-mcp-internal-pose` | [techdebt.go:182](file:///home/go/IdeaProjects/harne8/pose-dist/pose-mcp/internal/pose/techdebt.go#L182) | `if strings.Contains(upper, "TODO") {` | 📜 Sugere Spec |
| DEBT-068 | `TODO` | `pose-mcp-internal-pose` | [techdebt.go:183](file:///home/go/IdeaProjects/harne8/pose-dist/pose-mcp/internal/pose/techdebt.go#L183) | `marker = "TODO"` | 📜 Sugere Spec |
| DEBT-069 | `FIXME` | `pose-mcp-internal-pose` | [techdebt.go:184](file:///home/go/IdeaProjects/harne8/pose-dist/pose-mcp/internal/pose/techdebt.go#L184) | `} else if strings.Contains(upper, "FIXME") {` | 📜 Sugere Spec |
| DEBT-070 | `FIXME` | `pose-mcp-internal-pose` | [techdebt.go:185](file:///home/go/IdeaProjects/harne8/pose-dist/pose-mcp/internal/pose/techdebt.go#L185) | `marker = "FIXME"` | 📜 Sugere Spec |
| DEBT-071 | `PANIC` | `pose-mcp-internal-pose` | [techdebt.go:186](file:///home/go/IdeaProjects/harne8/pose-dist/pose-mcp/internal/pose/techdebt.go#L186) | `} else if strings.Contains(line, "panic(") {` | 📜 Sugere Spec |
| DEBT-072 | `STUB` | `pose-mcp-internal-pose` | [techdebt.go:188](file:///home/go/IdeaProjects/harne8/pose-dist/pose-mcp/internal/pose/techdebt.go#L188) | `} else if strings.Contains(line, "unimplemented!") \|\| strings.Con...` | 📜 Sugere Spec |
| DEBT-073 | `TODO` | `pose-mcp-internal-pose` | [techdebt.go:251](file:///home/go/IdeaProjects/harne8/pose-dist/pose-mcp/internal/pose/techdebt.go#L251) | `- **TODOs**: %d \| **FIXMEs**: %d \| **Panics**: %d \| **Stubs**: %d` | 📜 Sugere Spec |
| DEBT-074 | `TODO` | `pose-mcp-internal-pose` | [techdebt.go:262](file:///home/go/IdeaProjects/harne8/pose-dist/pose-mcp/internal/pose/techdebt.go#L262) | ``, report.EvaluatedAt, report.BaselineCommit, report.Summary.TotalM...` | 📜 Sugere Spec |
| DEBT-075 | `TODO` | `pose-mcp-internal-pose` | [techdebt.go:298](file:///home/go/IdeaProjects/harne8/pose-dist/pose-mcp/internal/pose/techdebt.go#L298) | `"1. **Follow-ups Rápidos**: Para marcações locais (como TODOs em...` | 📜 Sugere Spec |
| DEBT-076 | `TODO` | `pose-mcp-internal-pose` | [techdebt.go:299](file:///home/go/IdeaProjects/harne8/pose-dist/pose-mcp/internal/pose/techdebt.go#L299) | `"2. **Novas Specs**: Para componentes com alta densidade de TODOs (...` | 📜 Sugere Spec |
| DEBT-077 | `PANIC` | `pose-mcp-internal-scaffold` | [scaffold.go:23](file:///home/go/IdeaProjects/harne8/pose-dist/pose-mcp/internal/scaffold/scaffold.go#L23) | `panic(err) // impossible: dist is embedded at compile time` | 📌 Sugere Follow-up |

---

## 3. Matriz de Recomendações de Ação POSE

1. **Follow-ups Rápidos**: Para marcações locais (como TODOs em componentes como `graphforge-web` e `site`), registrar itens no backlog de follow-ups POSE (`project-state.md`).
2. **Novas Specs**: Para componentes com alta densidade de TODOs (como `graphforge-graphforge-web`), criar specs dedicadas em `.pose/specs/`.
3. **Roadmap Extensions**: Para dívidas arquiteturais ou acoplamentos sistêmicos, registrar novos marcos em `.pose/roadmaps/`.
