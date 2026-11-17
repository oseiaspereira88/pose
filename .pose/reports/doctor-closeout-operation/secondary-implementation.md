Implementado nos cinco paths autorizados:

- Novo [doctor_closeout_memo.go](/home/go/IdeaProjects/harne8/pose-dist/pose-mcp/internal/pose/doctor_closeout_memo.go): memo por consulta, somente de aprovações, sensível a ancestors/visited/cut. Limites: **2.048 entradas, 256 visited por resultado e 32.768 visited totais**.
- Novo [doctor_closeout_memo_test.go](/home/go/IdeaProjects/harne8/pose-dist/pose-mcp/internal/pose/doctor_closeout_memo_test.go): equivalência, contagem real, ciclos, ancestors, lifetime, limites, erros, remoto e binding.
- [spec.go](/home/go/IdeaProjects/harne8/pose-dist/pose-mcp/internal/pose/spec.go): apenas dois campos Store adicionados; hook existente preservado.
- [closeout_review_policy.go](/home/go/IdeaProjects/harne8/pose-dist/pose-mcp/internal/pose/closeout_review_policy.go): integração, preservando wrapper e fallback legacy.
- [governance_read_view.go](/home/go/IdeaProjects/harne8/pose-dist/pose-mcp/internal/pose/governance_read_view.go): reset dos novos campos.

O clock local fica fixo no início da consulta: expiração é avaliada nesse instante. A auditoria final continua obrigatória. Cruzamentos remotos, cortes, erros e autorização dinâmica impedem reuso. Novas views e material isolado descartam o memo.

**Validação:** corrigindo a contagem informada antes, **cinco testes da versão intermediária passaram**, em 4,382 s. A tentativa posterior falhou no linker por `disk quota exceeded`; a versão final precisa ser executada:

```bash
cd pose-dist/pose-mcp
GOTOOLCHAIN=local GOPROXY=off GOSUMDB=off \
GOCACHE=/tmp/harne8-go-cache \
go test ./internal/pose -run '^TestDoctorCloseoutMemo' -count=1 -timeout=35s
```

`gofmt` e `git diff --check` passaram. As fixtures usam reviews/receipts reais, mas topologia sintética e índice legado; não comprovam desempenho do doctor nem resolução de F5.

Sem commits, push, doctor, index/start/close ou alterações em spec/matrix/docs. Review Opus 5.5 permanece pendente.
