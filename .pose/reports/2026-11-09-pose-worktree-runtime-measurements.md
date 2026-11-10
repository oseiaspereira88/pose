# Medição do runtime opcional por worktree (B35)

Medido em 2026-11-10 por `POSE_RUNTIME_MEASURE=<arquivo> go test ./internal/cli -run TestMeasureTheRuntimeOnTheRealRepository`. As amostras, os cenários e os limiares T1 a T6 estavam no teste e na spec `pose-worktree-runtime-qos-and-measurement` antes da primeira execução e **não foram alterados depois de nenhuma execução**. O limiar T7 (dois checks pesados simultâneos) foi acrescentado, no teste, num commit anterior à terceira execução, depois que a revisão do milestone apontou que a medição não tinha carga pesada simultânea nem CPU.

Houve três execuções, e as três estão aqui:

1. 1ª (330 s): passou em T1 a T6. A revisão da fatia 3 achou lacunas no harness (métricas das faixas não impressas, saída curta não comparada, pico lido antes da carga).
2. 2ª (339 s): harness corrigido, T1 a T6 passaram, métricas das faixas registradas.
3. 3ª (343 s): a revisão do milestone achou que faltavam CPU, RAM por cenário e dois jobs pesados simultâneos; o harness ganhou T7 e as figuras de CPU/RSS por cenário. **Esta é a que qualifica o milestone.** T1 a T7 passaram.

## Ambiente

| item | valor |
|---|---|
| CPUs / GOMAXPROCS | 16 / 16 |
| SO / arquitetura | linux amd64, kernel 7.2.10-1-cachyos |
| Go / Git | go1.27.3-X:nodwarf5 / git version 2.56.0 |
| repositório | `pose-dist`, 2883 commits, HEAD `e91c7fff898b` |
| check pesado | `review-check spec:pose-governance-audit-skips-pure-git-reads` |
| consulta curta | `surface-check spec:pose-governance-audit-skips-pure-git-reads` |
| configuração do runtime | padrão: 1 slot pesado (fila 4), 2 slots curtos (fila 16), memo de 256 MiB |

## Cenários, 3ª execução (ms)

| cenário | amostras | mediana | mínimo | máximo | mediana da 1ª / 2ª |
|---|---|---|---|---|---|
| cold_standalone | 3 | 25684 | 25548 | 25692 | 26781 / 26836 |
| fallback_no_runtime | 3 | 25536 | 25511 | 25602 | 25616 / 26938 |
| resident_first | 1 | 25507 | 25507 | 25507 | 26304 / 26924 |
| resident_warm | 5 | 10573 | 10537 | 10593 | 11049 / 11275 |
| restart_then_run | 3 | 25558 | 25556 | 25635 | 25754 / 26992 |
| short_idle | 10 | 8 | 7 | 9 | 8 / 9 |
| short_under_heavy | 10 | 7 | 7 | 8 | 8 / 8 |
| two_heavy_wall | 1 | 21197 | 21197 | 21197 | - / - |

## Limiares (3ª execução)

| limiar | resultado |
|---|---|
| T1 warm over cold | pass: 10573 ms / 25684 ms = 0.41 <= 0.60 |
| T2 restart inherits nothing | pass: 25558 ms / 25684 ms = 1.00 >= 0.80 |
| T3 short under heavy | pass: p95 8 ms <= 250 ms (idle p95 9 ms) |
| T4 runtime memory | pass: peak resident 550 MiB <= 1536 MiB |
| T5 output | pass: every hosted run, the review-check and the short query idle and under load, printed exactly what the standalone run printed (25610 and 89 bytes) |
| T6 fallback overhead | pass: -148 ms <= 500 ms |
| T7 two simultaneous heavy checks | pass: both through the runtime with the standalone output; wall 21197 ms <= 2.5 x 10573 ms |

## CPU e memória do runtime por cenário (3ª execução)

| cenário | CPU do processo do runtime (ms) | RSS ao fim (MiB) |
|---|---|---|
| primeira execução e 5 warm | 60570 | 499 |
| um check pesado com 10 consultas curtas ao lado | 10100 | 490 |
| dois checks pesados simultâneos | 20360 | 507 |

Pico residente do processo ao longo da medição: 550 MiB. O tempo de CPU é o do processo do runtime (usuário mais sistema, lido de `/proc`); não inclui os processos `git` que ele inicia. Dois checks pesados levaram 2.0 vezes uma execução warm: com um slot pesado o segundo espera o primeiro (espera máxima registrada: 10580 ms), e a CPU total do par é cerca do dobro da de um check, como a execução em série implica.

## Leitura

- O runtime aquecido responde o `review-check` em cerca de 10,6 s contra cerca de 25,7 s standalone (0.41 da cold). A primeira execução dentro do runtime é tão lenta quanto a standalone (o memo começa vazio), e um runtime reiniciado volta a ela (1.00 da cold): nada é herdado.
- Uma consulta curta leva cerca de 8 ms ociosa e cerca de 7 ms enquanto um check de 25 s roda na faixa pesada (p95 9 ms e 8 ms), com saída idêntica à standalone.
- Dois checks pesados simultâneos: os dois passam pelo runtime com a saída idêntica; o segundo espera a vez, e o par leva cerca de 2× um check warm.
- O fallback sem runtime custou -148 ms (ruído; o custo é desprezível).

## Suportado, configurado e efetivo

- Suportado: Linux e macOS; Windows roda sempre standalone. A medição foi só em Linux.
- Configurado: padrões acima; `--heavy-slots`, `--short-slots`, `--heavy-queue`, `--short-queue`, `--memo-mib` os mudam.
- Efetivo (métricas do status ao fim da fase de carga): faixa pesada com 1 slot e fila de 4, 9 admitidos, 0 recusados, espera máxima 10580 ms (o segundo dos dois checks simultâneos); faixa curta com 2 slots e fila de 16, 20 admitidos, 0 recusados, espera 0. A recusa por fila cheia é exercida pelo teste com comando lento injetado, não por esta medição.

## Limites da medição

Um ambiente, um repositório, 3 a 10 amostras por cenário; vale para o histórico medido. A carga pesada foi `review-check`; `artifact-check` e `closeout-check` não foram medidos sob carga. Não há cota nem afinidade de CPU no runtime: ele limita o que admite e guarda, e a CPU medida é a que o processo gastou, não um limite imposto. As consultas curtas foram sequenciais entre si.
