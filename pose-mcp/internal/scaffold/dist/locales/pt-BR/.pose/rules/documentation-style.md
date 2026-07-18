# Rule: Documentation Style

## Quando consultar

Consulte este guia em tarefas de documentação de processo, regras, specs, workflows e instruções operacionais.

## Convenções obrigatórias

- Escreva no tom imperativo, com verbos de ação no início das instruções.
- Use bullets curtos com uma única ideia por item.
- Evite duplicar seções com o mesmo propósito em arquivos diferentes.
- Prefira links/referências para a fonte única em vez de copiar conteúdo.
- Use nomenclatura consistente: `check`, `spec` e `workflow`.
- Explicite o escopo da instrução para reduzir ambiguidades.

## Exemplos: bom vs ruim

### Redundância

- **Bom:** "Atualize critérios de review em `.pose/workflows/review.md` e referencie esse workflow no AGENTS raiz."
- **Ruim:** "Repita os critérios de review no AGENTS, no workflow e em cada spec relacionada."

### Referência ambígua

- **Bom:** "Rode o `check` de lint descrito no `workflow` de review."
- **Ruim:** "Rode aquela validação padrão antes de subir."

## Checklist rápido de aderência editorial

- Linguagem está imperativa e direta.
- Bullets estão curtos e sem sobreposição.
- Não há duplicação de seção entre arquivos.
- Termos `check`, `spec` e `workflow` foram usados de forma consistente.
- Referências apontam para arquivo/caminho explícito.

## Precedência em conflito multi-domínio

- Em conflito com outras `rules`, aplique a alternativa mais restritiva para segurança, contrato e operação.
- Quando houver choque entre velocidade e controle, priorize evidência verificável de `check` e mitigação explícita de risco.
- Registre no parecer de review a decisão de precedência e o racional objetivo.

## Rastreabilidade de recorrência

> Aplicar também: [.pose/rules/_base-recurrence.md](_base-recurrence.md)
