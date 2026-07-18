# Rule: Kubernetes

## When to consult

Consult this guide em tarefas de manifests Kubernetes, Helm/Kustomize, configuração de deploy, escalabilidade e operação em cluster.

## Required patterns

- Todo workload must definir `resources.requests` e `resources.limits`.
- Probes (`liveness`, `readiness`) must refletir comportamento real da aplicação.
- Imagens must ser versionadas de forma imutável (tag fixa/digest), evitando `latest`.
- Configuração must separar segredo de configuração pública (Secret vs ConfigMap).
- Strategy de rollout must minimizar indisponibilidade (ex.: rolling update).
- Namespaces, labels e annotations must seguir padrão de rastreabilidade do projeto.

## Blocking anti-patterns

- Deploy without probes, without recursos ou with privilégios excessivos desnecessários.
- Uso de `:latest` em imagem de produção.
- Secrets hardcoded em manifest, chart ou values.
- Exposição externa without política mínima de security/restrição.
- Alteração de manifest without considerar backward compatibility de rollout.

## Minimum checks

- Validation estrutural de YAML/manifest without errors.
- `kubectl apply --dry-run=client` (ou equivalente) no escopo alterado.
- Renderização de templates (Helm/Kustomize) concluída without error.
- Verificação de políticas de security/contrato de plataforma aplicáveis.

## Precedência em conflito multi-domínio

- Em conflito with outras `rules`, apply a alternativa mais restritiva para security, contrato e operação.
- When houver choque entre velocidade e controle, priorize evidência verificável de `check` e mitigação explícita de risco.
- Registre no parecer de review a decisão de precedência e o racional objetivo.

## Rastreabilidade de recorrência

> Aplicar também: [.pose/rules/_base-recurrence.md](_base-recurrence.md)
