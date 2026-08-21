# Regra: Backend Python

## Quando consultar

Consulte este guia para endpoints de API, modelos de domínio, persistência, tarefas assíncronas em segundo plano e integrações de serviços em aplicações backend Python (FastAPI, Django, Flask, Celery, Litestar).

## Padrões obrigatórios

- Use type annotations explícitas em funções públicas, classes e fronteiras de schema (`typing`, dataclasses/modelos Pydantic).
- Isole I/O bloqueante (operações de arquivo, chamadas síncronas legadas de banco) dos event loops assíncronos usando threadpools ou drivers assíncronos nativos.
- Propague exceções com contexto de domínio claro e mapeie para status codes HTTP padronizados.
- Encapsule consultas a bancos de dados e fronteiras de transação em abstrações dedicadas de repositório ou unit-of-work.
- Use logs estruturados sem expor dados pessoais (PII) ou tokens sensíveis.

## Anti-padrões bloqueantes

- Capturar `Exception` ou `BaseException` às cegas com `pass` sem registrar log ou tratar o erro.
- Usar argumentos default mutáveis (ex: `def foo(bar=[])`).
- Executar queries SQL puras com interpolação ou formatação de strings em vez de queries parametrizadas.
- Invocar chamadas síncronas bloqueantes diretamente dentro de endpoints `async def`.
- Deixar sessões ou conexões de banco de dados abertas ao longo do ciclo de vida da requisição.

## Checagens mínimas

- Rodar `pytest` nos testes do escopo afetado.
- Rodar `ruff check .` ou `flake8` sem achados críticos.
- Rodar `mypy` ou `pyright` sem erros bloqueantes de tipagem.
- Rodar verificação de segurança com `bandit` ao alterar escopos sensíveis.

## Precedência em conflitos multi-domínio

- Aplicar a regra de segurança, contrato e operação mais restritiva em caso de conflito.
- Preferir evidências de validação verificáveis e mitigação explícita de risco quando velocidade entrar em conflito com controle.
- Registrar a decisão de precedência e a justificativa objetiva no review.

## Rastreabilidade de recorrência

> Aplique também: [.pose/rules/_base-recurrence.md](_base-recurrence.md)
