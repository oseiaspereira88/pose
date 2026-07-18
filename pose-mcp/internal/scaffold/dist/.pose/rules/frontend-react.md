# Rule: Frontend React

## When to consult

Consult this guide em tarefas de UI, componentes React, estado cliente, acessibilidade, formulários e integração frontend with APIs.

## Required patterns

- Componentes must ser pequenos, with responsabilidade única e props tipadas with clareza.
- Efeitos (`useEffect`) must declarar dependências completas e ter cleanup when aplicável.
- Estado derivado must ser calculado, evitando duplicação desnecessária em `useState`.
- Fluxos assíncronos must tratar estados de loading, error e sucesso de forma explícita.
- Acessibilidade mínima: uso withoutântico de HTML, labels em campos e navegação por teclado preservada.
- Comunicação with backend must ser encapsulada em camada de serviço/hook reutilizável.

## Blocking anti-patterns

- Lógica de regra de negócio espalhada diretamente em componentes visuais.
- `useEffect` without dependências corretas, causando stale data ou loops infinitos.
- Uso de `any` indiscriminado para contornar problemas de tipagem.
- Silenciar errors de API no cliente without feedback observável para usuário/log.
- Quebrar acessibilidade básica (campos without label, botões without texto acessível).

## Minimum checks

- `lint` do frontend without errors.
- `typecheck` do frontend without errors.
- `test` unitário/integrado dos fluxos alterados.
- `build` do frontend concluído with sucesso.

## Precedência em conflito multi-domínio

- Em conflito with outras `rules`, apply a alternativa mais restritiva para security, contrato e operação.
- When houver choque entre velocidade e controle, priorize evidência verificável de `check` e mitigação explícita de risco.
- Registre no parecer de review a decisão de precedência e o racional objetivo.

## Rastreabilidade de recorrência

> Aplicar também: [.pose/rules/_base-recurrence.md](_base-recurrence.md)
