# Workflow: Refactor

## Objetivo

Melhorar estrutura interna e manutenibilidade sem alterar comportamento funcional observado.

## Precondições

- Motivação técnica do refactor está documentada.
- Escopo está delimitado por módulo e risco.
- Critérios de não-regressão funcional foram definidos.
- Baseline de testes/checks está disponível.

## Checklist de execução

1. Definir objetivo técnico (legibilidade, acoplamento, duplicação, etc.).
2. Mapear limites de escopo e contratos que devem permanecer intactos.
3. Quebrar refactor em etapas pequenas, revisáveis e revertíveis.
4. Executar mudanças mecânicas com commits/diffs coesos; quando governado por uma spec, incluir o trailer `POSE-Spec: <slug>` na mensagem do commit antes de rodar `pose artifact-check` e `pose close`.
5. Garantir equivalência comportamental com testes automatizados.
6. Rodar checks determinísticos relevantes (`test`, `lint`, `typecheck`, `build`).
7. Medir ganhos práticos (complexidade, clareza, cobertura, manutenção).
8. Registrar riscos residuais e follow-ups não essenciais.
9. Se o Modo Contribuidor estiver ativo e o refactor expuser atritos do motor ou templates rígidos, registre feedback via `pose contribute stage --type enhancement`.

## Saídas obrigatórias

- Descrição do problema estrutural e da estratégia aplicada.
- Evidência de preservação de comportamento.
- Resultado de checks determinísticos executados.
- Lista de ganhos obtidos e pendências futuras.

## Critérios de pronto

- Comportamento funcional permaneceu equivalente.
- Refactor reduziu dívida técnica de forma verificável.
- Não houve expansão de escopo para mudanças não relacionadas.
- Checks relevantes aprovados.
