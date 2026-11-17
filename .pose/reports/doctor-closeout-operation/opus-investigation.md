Fiz só leitura, sem editar nada, sem rodar testes nem o doctor. Li `AGENTS.md` (raiz e `pose-dist/`) e os quatro arquivos pedidos, além de `federated_acceptance.go`, `spec.go`, `roots.go` e `review_closeout.go`. A sua hipótese se confirma, com um detalhe a mais que explica o custo.

## Causa

**1. O resolver perde os snapshots a cada operação.** `withGovernanceReadView` zera `resolver.snapshots` (`governance_read_view.go:2621`), e `rootsForProducer` devolve um `Roots` novo com `governanceStores` vazio (`:2654-2667`). Assim, toda chamada `resolver.Resolve` (`artifact_ref.go:155-157`) recebe de `Roots.storeFor` um `Store{Root, FS}` sem `specs` e sem `reviewView` (`roots.go:75-84`). Nesse store, `GetSpec` → `ListSpecs` → `listSpecs` (`spec.go:305-306`) refaz o parse de toda a frontmatter do catálogo a cada `Resolve`.

O `WithReviewOperationView` só serve catálogo ao store cujo `boundTo` bate: mesma raiz, projeto, ponteiro de resolver e Files. Ou seja, só ao store primário, nunca aos stores de origem que vêm de `Roots`. A I/O provavelmente já sai da captura do read view (o teste existente conta 6 leituras de bytes). O que se repete é CPU de parse, e isso bate com os 210s em `Resolve/GetSpec/ListSpecs`.

**2. O custo se multiplica.** Para cada escopo, `closeoutReviewsOwed` chama `closedByReceipt` → `closeoutTerminalWithDependencies` → `getCloseoutState(ref, true)` → `FederatedRoadmapAcceptance` (`review_closeout.go:2781-2787`). Dentro de `federatedAcceptance`:
- cada aresta chama `Resolve` e `ValidateGraph`;
- `federatedOwnershipConflicts` (`federated_acceptance.go:882-916`) chama `Resolve` para **todo membro de todo roadmap ativo de cada projeto** em `projectRoots`. Ele é chamado uma vez por aceitação, daí os 125s;
- para predecessores locais (`After`/`DependsOn`/`Consumes`, e os milestones de um roadmap), `localScopeClosed` (`:469`) chama `closedByReceipt(pred)` de novo, com uma nova aceitação completa.

O resultado é: escopos × fechamento transitivo dos predecessores × (membros ativos × parse do catálogo). Numa cadeia de k milestones, os fechamentos já são O(k²) antes de contar o resto.

**3. O diff anterior não ataca esse caminho.** Ele compartilha só catálogo e headers locais e não toca no resolver. Sem memo de fechamento, cada predecessor também é refeito por inteiro.

Um ponto já verificado: o `cliGovernedStore` do doctor (`cli/review_closeout.go:23-33`) traz resolver com `Roots`. Por isso `closeoutTerminalWithDependencies` não reconstrói o resolver de ambiente e o `boundTo` primário se mantém.

## Patch sugerido (passo 1, mínimo)

Criar um snapshot novo por operação no ponto de entrada da operação de review. Ele precisa vir **antes** de construir `reviewOperationView`, porque `boundTo` compara o ponteiro do resolver. Já existe precedente em `index_federated_material.go:99`.

```go
// review_operation_view.go
func (s Store) withOperationResolverSnapshots() Store {
	if s.FederatedResolver == nil || s.FederatedResolver.Roots == nil {
		return s
	}
	r := s.FederatedResolver.WithSpecSnapshots() // cópia por valor, sync.Map nova
	s.FederatedResolver = &r
	return s
}

// WithReviewOperationView, após base.withGovernanceReadView(...):
shared := base.withGovernanceReadView(readView, "review-operation")
if keyable {
	shared = shared.withOperationResolverSnapshots()
}
view := &reviewOperationView{... resolver: shared.FederatedResolver, ...}

// WithReviewScopeOperationView: idem, entre withGovernanceReadView e a construção de view.
```

Por que isso preserva os invariantes:
- **Binding:** `view.resolver` já é o ponteiro com snapshots. Os stores de origem continuam sem `reviewView`, então nenhuma checagem de binding muda.
- **Raízes e autorização:** `Authorize` e `Roots` não mudam (é cópia por valor). A autorização roda antes de `StoreFor` (`artifact_ref.go:147`).
- **Frescor entre operações:** `withoutGovernanceReadView` e `withGovernanceReadView` continuam zerando os snapshots, e cada operação ou escopo cria um map novo.
- **Frescor dentro da operação:** as leituras passam pelo `Files` do read view, cuja auditoria final já detecta bytes alterados.
- **Erro:** um `listSpecs` com falha fica fixo dentro da operação e cai como `unknown-spec`, ou seja, fail-closed.
- **Deadline:** não muda.
- **`specSnapshot.bundleScope`:** usa `os.ReadFile` direto, mas não tem chamador. Vale não reativá-lo.

**O que descartei:** pré-popular `governanceStores[mainProject]` com o store que tem `reviewView`. Isso quebra o binding: `federatedAcceptance` reatribui `source.FederatedResolver = &resolver` (`:445`), o `boundTo` falha e `prepareReviewOperationRead` recusa, marcando `contextConflict`.

**Custo residual:** o projeto local passa a ter duas listagens, uma pelo `reviewView` e outra pelo snapshot. Ambas vêm das mesmas capturas de bytes, então é O(1) por operação em vez de O(Resolves).

## Memo de fechamento (passo 2): sim, precisa ser sensível a `closeoutChain`

`localScopeClosed` corta ciclos olhando se o escopo já está em `closeoutChain`. Então o resultado de `closedByReceipt(X)` depende da cadeia: com uma cadeia maior que contenha um escopo visitado por X, o corte pode transformar `true` em `false`. Um memo só por `ref` está errado.

Uma regra segura:
- **Quando gravar:** só se o cálculo não teve corte, sem erro, e só com `reviewView.boundTo(s)` (store primário).
- **O que gravar:** `{closed, visited}`, onde `visited` é o conjunto de escopos consultados em `localScopeClosed` ao longo do cálculo.
- **Quando reutilizar:** só se `chainAtual ∩ visited == ∅`. Uma cadeia disjunta não introduz cortes novos, então o resultado é idêntico.
- **Chave:** `filepath.Clean(root)+"\x00"+ref`. `localFederatedScope` gera escopos sem projeto e a cadeia mistura projetos, então raiz ou projeto precisam entrar na chave.
- **Onde guardar:** em `reviewOperationView` (com mutex), que morre com `closed`. Assim não há invalidação entre operações a fazer.

Para isso, um ponteiro `closeoutTrace{visited map[string]bool; cut bool}` em `Store`:
- `closedByReceipt` cria o trace filho e, no retorno, une `visited`/`cut` ao pai;
- `localScopeClosed` registra o escopo e marca `cut` quando encontra ciclo;
- **obrigatório:** propagar `source.closeoutTrace = s.closeoutTrace` junto com `closeoutChain` em `AggregateDependencyClosed` (`closeout_review_policy.go:353`). Sem isso, um corte em projeto remoto fica invisível e o memo fica incorreto.

Não gravar erros, para não fixar falhas transitórias ou de deadline. Workers concorrentes só deixam a regra mais conservadora.

Recomendo aplicar e medir o passo 1 antes. O passo 2 é o que remove o O(k²), mas é mais delicado.

## Fixtures de regressão

Contar com `reviewViewCountingFiles` não basta, porque o read view já deduplica a I/O. Proponho um contador de teste não exportado em `listSpecs` (por exemplo `listSpecsCalls atomic.Int64`, por raiz).

1. **Contagem do resolver.** Projeto com 20 specs e um roadmap ativo com 3 milestones encadeados via `After`, cada um com 3 membros, e policy schema 5. Rodar `CloseoutReviewsOwed` uma vez e verificar que `listSpecs` roda no máximo uma vez para o `reviewView`, uma para o snapshot por projeto e uma para a auditoria de `validateSpecs`. Hoje cresce com membros × escopos. Repetir com um segundo projeto via `POSE_PROJECT_ROOTS` e verificar ≤ 1 listagem por projeto.
2. **Equivalência.** Na mesma fixture, comparar `reflect.DeepEqual` entre o resultado de `CloseoutReviewsOwed` e o loop antigo `EvaluateCloseoutReview`/`closedByReceipt` por escopo fora de operação. Variar os casos: membro transferido (redirect), membro externo não confiável, propriedade conflitante entre dois roadmaps ativos.
3. **Frescor dentro da operação.** Hook no `Files` que altera `spec.md` de um membro depois do primeiro `Resolve`. Esperado: `ErrGovernanceReadViewChanged` e nenhum resultado verde.
4. **Frescor entre operações.** Duas chamadas seguidas no mesmo `Store`, com a mudança de status de um membro entre elas. A segunda tem que ver o novo status, provando que não há snapshot herdado.
5. **Binding.** Store com outro `FederatedResolver` ou outro `Files` dentro do callback continua sendo recusado (`contextConflict`), e o caminho não-keyable continua sem snapshots.
6. **Ciclo (passo 2).** A `After` B, B `After` A, e também uma variante cruzando projetos. Os dois ficam não fechados, e a contagem de `closedByReceipt` fica igual com e sem memo. Numa cadeia linear de k=6, `getCloseoutState` roda ≤ k+1 vezes. Num caso em que X foi calculado com cadeia vazia e depois é pedido com uma cadeia que contém um escopo visitado por X, o memo **não** pode ser reutilizado.

Pontos em aberto: não verifiquei se `CaptureGovernanceRawHistoryGitIn` (rev-parse 2× por `Resolve`) já é memoizado pelo estado git do read view. Se não for, esse é o próximo custo depois da correção.
