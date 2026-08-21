# Regra: Infraestrutura Docker

## Quando consultar

Consulte este guia para Dockerfiles, Containerfiles, definições de Docker Compose, builds de imagens e especificações de runtime de contêineres.

## Padrões obrigatórios

- Use builds multi-stage para gerar imagens finais enxutas sem ferramentas de compilação ou dependências de desenvolvimento.
- Exija execução como non-root criando e alternando para um `USER` dedicado com privilégios mínimos de UID/GID.
- Trave imagens base em tags ou digests específicos e imutáveis, evitando `:latest`.
- Otimize o cache de camadas: copie manifestos de dependência (`package.json`, `go.mod`, `Cargo.toml`, `requirements.txt`) antes de copiar o código-fonte completo.
- Defina checagens de saúde explícitas (`HEALTHCHECK`) e configure sinais de parada adequados (`STOPSIGNAL`).
- Use `.dockerignore` para excluir artefatos locais, `.git`, `node_modules`, caches e arquivos de segredos do contexto de build.

## Anti-padrões bloqueantes

- Rodar contêineres como `root` em imagens de produção sem justificativa de segurança formal.
- Passar segredos, tokens ou senhas via instruções `ARG` ou `ENV` que persistem nas camadas da imagem.
- Usar tags de imagem base não fixadas ou mutáveis como `:latest` em Dockerfiles de produção.
- Instalar pacotes de sistema sem limpar caches no mesmo comando `RUN` (ex: faltar `rm -rf /var/lib/apt/lists/*`).
- Montar o socket do Docker do host (`/var/run/docker.sock`) dentro de contêineres de aplicação comuns.

## Checagens mínimas

- Rodar `hadolint Dockerfile` sem violações de nível de erro.
- Rodar `docker build --check .` ou scanner de vulnerabilidades (Trivy / Grype).
- Garantir que a imagem de runtime seja construída e inicializada com sucesso.

## Precedência em conflitos multi-domínio

- Aplicar a regra de segurança, contrato e operação mais restritiva em caso de conflito.
- Preferir evidências de validação verificáveis e mitigação explícita de risco quando velocidade entrar em conflito com controle.
- Registrar a decisão de precedência e a justificativa objetiva no review.

## Rastreabilidade de recorrência

> Aplique também: [.pose/rules/_base-recurrence.md](_base-recurrence.md)
