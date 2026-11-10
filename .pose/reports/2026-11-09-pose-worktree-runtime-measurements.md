# Medição do runtime opcional por worktree (B35, fatia 3)

Medido em 2026-11-10 por `POSE_RUNTIME_MEASURE=<arquivo> go test ./internal/cli -run TestMeasureTheRuntimeOnTheRealRepository` (330 s). As amostras, os cenários e os limiares T1 a T6 estavam no teste e na spec `pose-worktree-runtime-qos-and-measurement` antes da primeira execução; nenhum foi ajustado depois dela. Esta foi a única execução.

## Ambiente

| item | valor |
|---|---|
| CPUs / GOMAXPROCS | 16 / 16 |
| SO / arquitetura | linux amd64, kernel 7.2.10-1-cachyos |
| Go / Git | go1.27.3-X:nodwarf5 / git version 2.56.0 |
| repositório | `pose-dist`, 2862 commits, HEAD `7c2abbf6801e` |
| check pesado | `review-check spec:pose-governance-audit-skips-pure-git-reads` |
| consulta curta | `surface-check spec:pose-governance-audit-skips-pure-git-reads` |
| configuração do runtime | padrão: 1 slot pesado (fila 4), 2 slots curtos (fila 16), memo de 256 MiB |

## Cenários (ms)

| cenário | amostras | mediana | mínimo | máximo |
|---|---|---|---|---|
| cold_standalone | 3 | 26781 | 26466 | 27103 |
| fallback_no_runtime | 3 | 25616 | 25595 | 26315 |
| resident_first | 1 | 26304 | 26304 | 26304 |
| resident_warm | 5 | 11049 | 11012 | 11089 |
| restart_then_run | 3 | 25754 | 25741 | 25770 |
| short_idle | 10 | 8 | 7 | 9 |
| short_under_heavy | 10 | 8 | 7 | 10 |

## Limiares

| limiar | resultado |
|---|---|
| T1 warm over cold | pass: 11049 ms / 26781 ms = 0.41 <= 0.60 |
| T2 restart inherits nothing | pass: 25754 ms / 26781 ms = 0.96 >= 0.80 |
| T3 short under heavy | pass: p95 10 ms <= 250 ms (idle p95 9 ms) |
| T4 runtime memory | pass: peak resident 537 MiB <= 1536 MiB |
| T5 output | pass: every hosted run printed exactly what the standalone run printed (25561 bytes) |
| T6 fallback overhead | pass: -1165 ms <= 500 ms |

## Leitura

- O runtime aquecido responde o `review-check` em cerca de 11 s contra cerca de 27 s standalone (0.41 da mediana cold). A primeira execução dentro do runtime é tão lenta quanto a standalone (o memo começa vazio), e um runtime reiniciado volta a ela (0.96 da cold): nada é herdado de um processo anterior.
- Uma consulta curta leva cerca de 8 ms com o runtime ocioso e cerca de 8 ms enquanto um `review-check` de 25 s roda na faixa pesada (p95 9 ms e 10 ms): as faixas isolam.
- O pico residente do runtime depois das execuções foi 537 MiB (RSS 503 MiB), com o memo limitado a 256 MiB.
- Tentar um runtime que não existe custou -1165 ms em relação ao standalone: dentro do ruído da medição (o valor é negativo porque as amostras variam em mais de um segundo).

## Suportado, configurado e efetivo

- Suportado: Linux e macOS (socket Unix com credencial do par); Windows roda sempre standalone. A medição foi feita só em Linux.
- Configurado: os padrões acima; as flags `--heavy-slots`, `--short-slots`, `--heavy-queue`, `--short-queue` e `--memo-mib` os mudam.
- Efetivo nesta medição: 1 comando pesado por vez e até 2 curtos; as métricas do status contaram ? admitidos pesados, ? curtos e ? recusados pesados.

## Limites da medição

Um ambiente, um repositório, 3 a 10 amostras por cenário. Vale para o tamanho de histórico medido, não para outros. A carga pesada foi um `review-check` (o mais comum); outros comandos pesados (`artifact-check`, `closeout-check`) não foram medidos sob carga.
