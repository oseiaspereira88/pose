# Rule: Frontend React

## Quando consultar

Consulte este guia em tarefas de UI, componentes React, estado cliente, acessibilidade, formulários e integração frontend com APIs.

## Padrões obrigatórios

- Componentes devem ser pequenos, com responsabilidade única e props tipadas com clareza.
- Efeitos (`useEffect`) devem declarar dependências completas e ter cleanup quando aplicável.
- Estado derivado deve ser calculado, evitando duplicação desnecessária em `useState`.
- Fluxos assíncronos devem tratar estados de loading, erro e sucesso de forma explícita.
- Acessibilidade mínima: uso semântico de HTML, labels em campos e navegação por teclado preservada.
- Comunicação com backend deve ser encapsulada em camada de serviço/hook reutilizável.

## Anti-padrões bloqueadores

- Lógica de regra de negócio espalhada diretamente em componentes visuais.
- `useEffect` sem dependências corretas, causando stale data ou loops infinitos.
- Uso de `any` indiscriminado para contornar problemas de tipagem.
- Silenciar erros de API no cliente sem feedback observável para usuário/log.
- Quebrar acessibilidade básica (campos sem label, botões sem texto acessível).

## Checks mínimos

- `lint` do frontend sem erros.
- `typecheck` do frontend sem erros.
- `test` unitário/integrado dos fluxos alterados.
- `build` do frontend concluído com sucesso.

## Precedência em conflito multi-domínio

- Em conflito com outras `rules`, aplique a alternativa mais restritiva para segurança, contrato e operação.
- Quando houver choque entre velocidade e controle, priorize evidência verificável de `check` e mitigação explícita de risco.
- Registre no parecer de review a decisão de precedência e o racional objetivo.

## Rastreabilidade de recorrência

> Aplicar também: [.pose/rules/_base-recurrence.md](_base-recurrence.md)
