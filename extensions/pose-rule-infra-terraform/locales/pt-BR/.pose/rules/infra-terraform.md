# Regra: Infraestrutura Terraform

## Quando consultar

Consulte este guia para módulos Terraform, configurações OpenTofu, definições de recursos em nuvem (AWS, GCP, Azure), gerenciamento de state e configuração de providers.

## Padrões obrigatórios

- Trave as versões dos providers e do Terraform explicitamente nos blocos `versions.tf` / `required_providers`.
- Armazene arquivos de state em backends remotos seguros (S3 com DynamoDB locking, GCS, Terraform Cloud) com criptografia habilitada.
- Siga princípios de menor privilégio em roles IAM, policies e security groups.
- Modularize componentes de infraestrutura com variáveis de entrada claras, descrições e contratos explícitos de output.
- Use regras de validação em variáveis e marque segredos como `sensitive = true`.

## Anti-padrões bloqueantes

- Commitar credenciais em texto claro, chaves de acesso ou tokens de API em arquivos `.tf` ou repositórios git.
- Usar permissões com curinga `*` em ações IAM ou ARNs de recursos sem justificativa formal documentada.
- Manter arquivos `terraform.tfstate` locais não criptografados em branches compartilhadas ou ambientes produtivos.
- Alterar recursos em nuvem manualmente (drift) sem reconciliar o state do Terraform.
- Definir regras de ingress abertas (`0.0.0.0/0`) em portas administrativas ou de banco de dados sensíveis (SSH, RDP, Postgres, MySQL).

## Checagens mínimas

- Rodar `terraform fmt -check` no diretório de infraestrutura.
- Rodar `terraform validate` nos módulos inicializados.
- Rodar linters de segurança como `tflint`, `checkov` ou `tfsec` sem violações de severidade alta ou crítica.

## Precedência em conflitos multi-domínio

- Aplicar a regra de segurança, contrato e operação mais restritiva em caso de conflito.
- Preferir evidências de validação verificáveis e mitigação explícita de risco quando velocidade entrar em conflito com controle.
- Registrar a decisão de precedência e a justificativa objetiva no review.

## Rastreabilidade de recorrência

> Aplique também: [.pose/rules/_base-recurrence.md](_base-recurrence.md)
