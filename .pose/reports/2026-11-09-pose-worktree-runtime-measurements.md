# Medição do runtime opcional por worktree (B35, fatia 3)

Medido em 2026-11-10 por `POSE_RUNTIME_MEASURE=<arquivo> go test ./internal/cli -run TestMeasureTheRuntimeOnTheRealRepository`. As amostras, os cenários e os limiares T1 a T6 estavam no teste e na spec `pose-worktree-runtime-qos-and-measurement` antes da primeira execução e **não foram alterados depois de nenhuma**.

Houve duas execuções, e as duas estão aqui. A primeira (330 s) passou em todos os limiares, mas a revisão independente achou três lacunas na medição: o status não imprimia as métricas das faixas (`runtime_metrics` saiu nulo), a saída do `surface-check` não era comparada com a standalone, e o pico residente era lido antes da fase concorrente. Corrigi o harness (só isso; sem tocar nos limiares) e rodei de novo (339 s). A segunda execução é a que qualifica; a primeira fica como confirmação independente dos mesmos números.

## Ambiente

| item | valor |
|---|---|
| CPUs / GOMAXPROCS | 16 / 16 |
| SO / arquitetura | linux amd64, kernel 7.2.10-1-cachyos |
| Go / Git | go1.27.3-X:nodwarf5 / git version 2.56.0 |
| repositório | `pose-dist`, 2871 commits, HEAD `1d4d257d0595` (a primeira execução: HEAD `7c2abbf6801e`, 2862 commits) |
| check pesado | `review-check spec:pose-governance-audit-skips-pure-git-reads` |
| consulta curta | `surface-check spec:pose-governance-audit-skips-pure-git-reads` |
| configuração do runtime | padrão: 1 slot pesado (fila 4), 2 slots curtos (fila 16), memo de 256 MiB |

## Cenários, segunda execução (ms)

| cenário | amostras | mediana | mínimo | máximo | mediana da 1ª execução |
|---|---|---|---|---|---|
| cold_standalone | 3 | 26836 | 26516 | 26904 | 26781 |
| fallback_no_runtime | 3 | 26938 | 26914 | 26965 | 25616 |
| resident_first | 1 | 26924 | 26924 | 26924 | 26304 |
| resident_warm | 5 | 11275 | 11216 | 11353 | 11049 |
| restart_then_run | 3 | 26992 | 26882 | 27019 | 25754 |
| short_idle | 10 | 9 | 7 | 11 | 8 |
| short_under_heavy | 10 | 8 | 7 | 9 | 8 |

## Limiares (segunda execução)

| limiar | resultado |
|---|---|
| T1 warm over cold | pass: 11275 ms / 26836 ms = 0.42 <= 0.60 |
| T2 restart inherits nothing | pass: 26992 ms / 26836 ms = 1.01 >= 0.80 |
| T3 short under heavy | pass: p95 9 ms <= 250 ms (idle p95 11 ms) |
| T4 runtime memory | pass: peak resident 535 MiB <= 1536 MiB |
| T5 output | pass: every hosted run, the review-check and the short query idle and under load, printed exactly what the standalone run printed (25610 and 89 bytes) |
| T6 fallback overhead | pass: 102 ms <= 500 ms |

## Leitura

- O runtime aquecido responde o `review-check` em cerca de 11 s contra cerca de 27 s standalone (0.42 da mediana cold; 0.41 na primeira execução). A primeira execução dentro do runtime é tão lenta quanto a standalone (o memo começa vazio), e um runtime reiniciado volta a ela (1.01 da cold): nada é herdado de um processo anterior.
- Uma consulta curta leva cerca de 8 ms com o runtime ocioso e cerca de 8 ms enquanto um `review-check` de cerca de 27 s roda na faixa pesada (p95 11 ms e 9 ms): as faixas isolam. A saída dela é idêntica à standalone nas duas condições.
- O pico residente do runtime, lido depois da fase concorrente, foi 535 MiB (RSS 518 MiB), com o memo limitado a 256 MiB.
- Tentar um runtime que não existe custou 102 ms em relação ao standalone nesta execução (-1165 ms na primeira: a diferença entre execuções é ruído, e a conclusão é que o custo é desprezível).

## Suportado, configurado e efetivo

- Suportado: Linux e macOS (socket Unix com credencial do par); Windows roda sempre standalone. A medição foi feita só em Linux.
- Configurado: os padrões acima; as flags `--heavy-slots`, `--short-slots`, `--heavy-queue`, `--short-queue` e `--memo-mib` os mudam.
- Efetivo nesta medição (métricas do status ao fim da fase de carga): faixa pesada com 1 slot e fila de 4, 7 comandos admitidos, 0 recusados, espera máxima 0 ms; faixa curta com 2 slots e fila de 16, 20 admitidos, 0 recusados, espera máxima 0 ms. Nenhum comando esperou nem foi recusado: a carga do cenário (um check pesado e consultas curtas em sequência) não esgota nenhuma fila; a recusa por fila cheia é exercida pelo teste com comando lento injetado, não por esta medição.

## Limites da medição

Um ambiente, um repositório, 3 a 10 amostras por cenário. Vale para o tamanho de histórico medido, não para outros. A carga pesada foi um `review-check` (o mais comum); `artifact-check` e `closeout-check` não foram medidos sob carga. As consultas curtas foram sequenciais, não simultâneas entre si.
