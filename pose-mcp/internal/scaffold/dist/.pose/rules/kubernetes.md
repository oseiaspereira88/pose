# Rule: Kubernetes

## When to consult

Consult this guide em tarefas de manifests Kubernetes, Helm/Kustomize, configuração de deploy, escalabilidade e operação em cluster.

## Required patterns

- Todo workload deve definir `resources.requests` e `resources.limits`.
- Probes (`liveness`, `readiness`) devem refletir comportamento real da aplicação.
- Imagens devem ser versionadas de forma imutável (tag fixa/digest), evitando `latest`.
- Configuração deve separar segredo de configuração pública (Secret vs ConfigMap).
- Estratégia de rollout deve minimizar indisponibilidade (ex.: rolling update).
- Namespaces, labels e annotations devem seguir padrão de rastreabilidade do projeto.

## Blocking anti-patterns

- Deploy sem probes, sem recursos ou com privilégios excessivos desnecessários.
- Uso de `:latest` em imagem de produção.
- Secrets hardcoded em manifest, chart ou values.
- Exposição externa sem política mínima de security/restrição.
- Alteração de manifest sem considerar backward compatibility de rollout.

## Minimum checks

- Validação estrutural de YAML/manifest sem errors.
- `kubectl apply --dry-run=client` (ou equivalente) no escopo alterado.
- Renderização de templates (Helm/Kustomize) concluída sem error.
- Verificação de políticas de security/contrato de plataforma aplicáveis.

## Precedência em conflito multi-domínio

- Em conflito com outras `rules`, aplique a alternativa mais restritiva para security, contrato e operação.
- Quando houver choque entre velocidade e controle, priorize evidência verificável de `check` e mitigação explícita de risco.
- Registre no parecer de review a decisão de precedência e o racional objetivo.

## Rastreabilidade de recorrência

> Aplicar também: [.pose/rules/_base-recurrence.md](_base-recurrence.md)
