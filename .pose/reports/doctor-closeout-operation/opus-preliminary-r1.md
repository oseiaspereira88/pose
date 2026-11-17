# Revisão preliminar de F4/F5/F6 (somente leitura, sem atestação)

Isto não aprova nada nem fecha nenhum item. Também não rodei nenhum comando: o Bash foi negado neste modo. Por isso **não li os diffs de fonte** e não consegui separar com certeza as mudanças de F5 do estudo de archive. Analisei o estado atual dos arquivos. Os arquivos em `/var/tmp/.../h8review6` e `/tmp/pose-active-delivery/f6-review-surface.log` ficam fora do diretório permitido e não foram lidos. Do log de F5 vi apenas `PASS` / `ok …/internal/pose 69.618s` (`.pose/review-private/active-delivery-tmp/f5-tests-race.log:85-86`); não confirmei pelo log que o `-race` estava ativo.

Não há bloqueador do código atual. Os bloqueios que existem são de evidência: medição do F5 e o R8 do executor.

---

## F5: snapshots por operação (R1–R5) e memo (R6)

### MEDIUM-1: `traceDoctorCloseoutSources` provavelmente anula o ganho no Harne8 e não tem controle negativo
`pose-mcp/internal/pose/doctor_closeout_memo.go:43-58`, com propagação em `:175`.

- **Cenário:** um scope terminal com qualquer dependência que resolve para outro projeto (ou não resolve, ou tem redirect canônico) recebe `cut`. Esse corte sobe para todos os ancestrais pelo `parent.merge`. Os dois tipos de scope que o doctor avalia se comportam assim:
  - **Scopes abertos:** `false` nunca entra no cache, e scopes sem recibo saem cedo (`closeout_review_policy.go:801-802`).
  - **Scopes fechados:** são justamente os de custo alto. No Harne8 eles carregam `xref:`; o comentário em `federated_acceptance.go:716` registra 64 provas, 39 distintas, num único closeout.
  
  Resultado provável: quase nenhum hit no caminho real do Harne8.
- **A proteção é necessária?**
  - Para agregados remotos ela é redundante, porque `AggregateDependencyClosed` já corta em `:345`.
  - Para specs remotas existe um argumento de equivalência, mas ele não está provado:
    - `federatedSourceProof` já memoiza por operação, sem relógio na chave (`federated_acceptance.go:720-728`).
    - A store de origem vem de `Roots.StoreFor` sem `atReviewTime`, então usa `time.Now` no primeiro cálculo (`review_trust_time.go:23-27`).
    - Portanto o memo agregado não aumentaria a defasagem além da que já existe *dentro da operação*.
  - Faltam no inventário as outras leituras remotas de `getCloseoutState`: `ValidateGraph`, gitlink/`verifyFederatedArtifactAtRevision`, redirect canônico local→remoto e status via snapshot.
- **Teste:** `doctor_closeout_memo_test.go:238` só exercita um `xref:` de *roadmap*, que já é cortado por `:345`. Nenhum teste reprovaria se o helper fosse removido.
- **Correção mínima:** manter o helper. Para evidência futura:
  1. Contar hits e motivos de corte (helper / remoto agregado / limite / erro) num clone do Harne8.
  2. Se o helper for o gargalo, restringir o corte só a `identity.Kind=="spec" && Resolved`. Isso exige testes de mutação em cada input remoto (status, artefato na revisão, bundle, expiração cruzando o instante, redirect) e fixar o relógio da store remota em `m.at`.
  3. Incluir um controle negativo em que a remoção do helper reprove.
  
  Não recomendo retirar a proteção antes disso.

### MEDIUM-2: relógio fixo também no caminho de fallback, sem registro na spec
`doctor_closeout_memo.go:100` executa `atReviewTime` **antes** da checagem de elegibilidade (`:102-108`).

- **Cenário:** com `Authorize != nil`, binding recusado ou view não keyável, o memo fica desligado, mas a expiração e a confiança locais passam a ser avaliadas no instante de início da query. Numa operação já medida acima de 600 s, uma review que expira no meio é julgada válida. O lado remoto continua em `time.Now`, o que cria uma assimetria.
- **Conflito com a spec:** R5 pede que o fallback não keyável mantenha o comportamento conservador. R6 e D1 não registram essa mudança de semântica.
- **Correção mínima:** aplicar `atReviewTime` só no ramo do memo, ou registrar uma Decision com limite e um teste de expiração cruzando o instante.

### LOW-1: o hit depende de `localScopeClosed` ser o único detector de ciclo
O detector está em `closeout_review_policy.go:393`. Há chamadores diretos de `closedByReceipt` sem checagem de cadeia: `roadmaps.go:358,403`, `roadmaps.go:65`, `obligation_projection.go:683`. A verificação de hit (`doctor_closeout_memo.go:148-158`) compara a cadeia com `visited`, então isso não cria nova falha de corretude. Mas um ciclo por esses caminhos continua sendo um problema anterior a esta entrega. Sugestão: um teste com ciclo através de `after:`.

### O que confere no código lido
- **R1:** erro da query ou da auditoria impede publicar a lista (`:893-898`).
- **R2/R5:** os snapshots são alocados antes do binding (`review_operation_view.go:150-155`); o caminho não keyável zera `specs` (`:160-167`).
- **Memo:** `false` e erro não entram no cache (`:175-176`). A view finalizada e `m.closed` caem no verificador original. A entrada exige a mesma view, o mesmo binding, o mesmo relógio e nenhum dos modos `indexMaterial`/`planWithoutCloseoutGate`. Os limites de 256, 2048 e 32768 forçam corte ou fallback. Nenhum lock é segurado durante a recursão.
- O cruzamento remoto de agregados é cortado mesmo quando dá erro.

**O que pode ser alegado:** testes unitários passaram. F5 não está resolvido, e não há medição de latência nem de hit-rate do memo final.

---

## F4: `recon_milestone_implementation_ready`

### MEDIUM-3: o filtro de "no review attempt exists" é amplo e não exige que o blocker exista
`tests/e2e/pose-multirepo/reconciliation.sh:214`

- **Cenário A:** o filtro descarta qualquer blocker que contenha esse texto, inclusive um de outro sujeito, como um membro ou executor sem tentativa de review.
- **Cenário B:** se `integrity` (predecessores `[]`) virar terminal por regressão de policy e os `blockers` ficarem vazios, a função passa. O R8 diz "não fechar automaticamente", e isso não fica afirmado no nível do milestone.
- **Ligação com integração:** em `:162/:189` o script diz que "integration still block", mas `:230` só verifica `field-pilot`. A pendência de integração depende só desse blocker de review que foi filtrado.
- **Correção mínima:** passar `$milestone` para a função e exigir exatamente o blocker do próprio scope, algo como `[.blockers[]|select(test("no review attempt exists"))] == ["<texto do milestone:pose-abm-foundation/$milestone>"]`, mais `.terminal == false`, se esse campo existir no JSON.

### LOW-2: a prova de bundle pode ser vazia por construção
`reconciliation.sh:217-220`: se nenhuma dependência tiver `identity.kind=="spec"` e `project_id=="proj.pose-dist"` (identidade ausente ou mudança de esquema), `[] | all` dá verdadeiro. Hoje isso é mitigado pela igualdade exata em `:212`, porque uma dependência não resolvida gera blocker federado. **Correção:** selecionar pelo `ref` em `$implementations` e exigir `length == ($implementations|length)` antes de `all`.

### O que confere
- `sort` sem `unique` detecta implementação duplicada; a igualdade exata detecta falta e blocker extra (`:212,:214-216`).
- `ready` fica ligado à lista de predecessores.
- A falha de `closeout-check` sem JSON faz o `jq` falhar.
- Não vi enfraquecimento das checagens de unknown ou cobertura no roadmap (`:228-231`).

**Evidência futura:** o R8 continua `deferred-integration`, com follow-up aberto para o entrypoint do executor (`.pose/specs/2026-09-21-harne8-abm-spec-authority-reconciliation.md:220,238`). F4 não cobre esse ponto, e a jornada completa ainda está rodando.

---

## F6: timeout de 60→180 s em `spec_review_real_test.go:27`

### LOW-3 (risco residual): a evidência não cobre os defaults de produção
- **Como o tempo se distribui:**
  - O `Timeout` vale **por chamada MCP** (`mcpclient.go:58` via `Connect`).
  - São 6 chamadas sequenciais (`review.go:86-89,128`).
  - Em 162,80 s, isso dá média acima de 27 s, logo ao menos uma chamada passou do default de 20 s (`review.go:73`). A produção usa esse default (`cmd/conductor/main.go:480`).
- **Bypass do servidor HTTP:** o teste chama `ServeHTTP` com `httptest.NewRecorder` (`:30`), o que contorna o `WriteTimeout: 60s` (`main.go:589`).
- **Em produção, com o motor frio:** as partes provavelmente viriam como `"pose-mcp unavailable"` (`review.go:108`) ou a escrita seria cortada.

**O que pode ser alegado:** o check real opt-in termina com closeout terminal, todas as partes disponíveis e 404 para spec inexistente, dentro de um orçamento de teste. Todas as asserções foram mantidas (`:51-62,67`). O contrato público não foi expandido.

**O que não pode ser alegado:** que a superfície de review do conductor funcione a frio com os defaults de produção.

**Correção mínima:** registrar isso como known gap ou follow-up (latência por parte) sem mexer nos defaults nesta entrega. Opcionalmente, logar a duração de cada parte no teste.

---

## Resumo

| Item | Achado | Tipo |
|---|---|---|
| F5 | MEDIUM-1: helper provavelmente anula o ganho no Harne8 e não tem controle negativo | Evidência futura (manter a proteção) |
| F5 | MEDIUM-2: relógio fixo também no fallback, fora da spec | Risco; correção pequena |
| F5 | LOW-1 | Risco anterior à entrega |
| F4 | MEDIUM-3: filtro amplo e sem asserção de não-terminal | Corrigir antes do commit |
| F4 | LOW-2 | Endurecimento |
| F6 | LOW-3 | Limitar o que se alega + known gap |

F5 continua aberto até doctor e checks completos terminarem dentro do orçamento.
