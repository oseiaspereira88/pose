# PR179 — complemento de publicação e exceção pontual de Security

## Escopo e proveniência

Integrar os resultados gerados de `f6cbbc0a87db1b8c82ce7ab4d9c476e831a78dce`
após o merge do PR179, preservando esse commit como ancestral. A base é a main
`2d016fe678c32f933f273a6aafc6046fc3ef1f9f`, que já contém o lifecycle `done`
de `pose-action-answers-reverify-on-read`, completed_at 2026-10-15.

O maintainer respondeu “Prossiga” ao escopo apresentado nesta conversa:
publicar este complemento em um PR e adotar exclusivamente a exceção do
fingerprint documentado. A aplicação é da execução `agent:codex-pr179-complement`;
isso não cria outra resposta à ação nem outra attestation técnica.

Preservar `act-3ee9344e0652cc60` answered/satisfied, revisão2, `human:oseias`,
declared, o bundle `rvb-cfe993ee0dbf209f` e a attestation
`rva-370e6fda5e127a19`, de `agent:independent-codex-pr179`. Não reiniciar a spec,
recriar baseline, modificar policies de revisão ou reabrir B04.

## Exceção de scanner

O histórico publicado contém um identificador de deduplicação da resposta de
aceite no snapshot `.pose/results/b01-action-acceptance.json`. Gitleaks o
classificou como `generic-api-key`; ele não é uma credencial. O diagnóstico
original permanece em `b01-action-post-publication-security.json`, como
registro da verificação que estava bloqueada antes deste complemento.

Adicionar à `.gitleaksignore` somente:

```text
781151f1edabfa913bca9516ae920f76cd76fb09:.pose/results/b01-action-acceptance.json:generic-api-key:106
```

A exceção fixa commit, caminho, regra e linha. Não ignora o commit inteiro,
o arquivo inteiro, outros identificadores ou qualquer assinatura de segredo.
`.gitleaks.toml`, workflows e suas regras permanecem inalterados. O scanner
lê a exceção da raiz também quando iniciado em `pose-mcp` com o alvo `..`,
como no job Security. Não apagar nem reescrever o registro histórico.

## Validação

- Gitleaks8.21.2: scan completo pela raiz, 1936 commits, zero achados.
- Mesma versão/configuração e diretório do CI (`pose-mcp`, alvo `..`):
  1936 commits, zero achados.
- Controle negativo isolado: a mesma posição/caminho em outro commit e uma
  credencial sintética foram detectados; exit1 esperado, dois achados.
- Check estrutural estrito aprovado, com os 30 warnings históricos.
- Review verify confirmou closed/fresh=true/approved=true; review-check e
  closeout-check passaram (terminal=true, blockers=[]); plano terminal e lint
  estrito passaram. Conferir novamente no head publicado. A revisão anterior
  continua retida pelo motor, sem novos julgamentos em nome desse revisor.
- Reutilizar a evidência funcional132/132: não houve alteração de runtime,
  testes Go, requisitos, baseline ou matriz de validação. CI e Security do
  novo PR fornecem a verificação adicional do head candidato ao merge.

Os resultados atuais deste complemento ficam em
`../results/b01-action-complement-verification.json`. Os snapshots anteriores
mantêm seus respectivos sujeitos e horários; não são reescritos como se
tivessem observado esta correção. Os links e conclusões do CI/Security no
head final são publicados na descrição do PR complementar.

## Limites do programa

Matriz e roadmaps continuam registrando somente a spec action-answer como
done. B01 permanece parcial, delivery_complete=false, até a qualificação e
o closeout de `pose-review-disclosure-requires-proof` em outro PR. B04 continua
terminal e B02 aguardando. Este complemento não inicia essa spec ou outro lote.
