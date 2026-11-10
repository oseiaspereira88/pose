# Medição do runtime opcional por worktree (B35)

Medido em 2026-11-10 por `POSE_RUNTIME_MEASURE=<arquivo> go test ./internal/cli -run TestMeasureTheRuntimeOnTheRealRepository`. As amostras, os cenários e os limiares T1 a T6 estavam no teste e na spec `pose-worktree-runtime-qos-and-measurement` antes da primeira execução e **não foram alterados depois de nenhuma execução**. O limiar T7 (dois checks pesados simultâneos, no máximo 2,5 vezes uma execução warm) foi fixado no teste no commit `e91c7fff`, antes da execução que o mede, e está no R8 da spec.

Houve quatro execuções, e as quatro estão no JSON ao lado:

1. 1ª (330 s): T1 a T6 passaram. A revisão da fatia 3 achou lacunas no harness (métricas das faixas não impressas, saída curta não comparada, pico lido antes da carga).
2. 2ª (339 s): harness corrigido; T1 a T6 passaram.
3. 3ª (343 s): a revisão do milestone apontou que faltavam CPU, RAM por cenário e dois jobs pesados simultâneos; o harness ganhou T7 e as figuras de CPU e RSS por cenário; T1 a T7 passaram.
4. 4ª (345 s): a segunda rodada do milestone apontou que a CPU das consultas curtas ociosas era descartada, que o clock tick era fixo e que a memória por cenário era o RSS ao fim e não o pico; o harness lê `getconf CLK_TCK`, zera e lê o pico por cenário (`clear_refs`) e guarda a CPU ociosa; T1 a T7 passaram. **Esta é a que qualifica o milestone.**

As quatro execuções deram os mesmos veredictos e números próximos (warm entre 0,40 e 0,42 da cold, dois pesados em 2,0 vezes), o que é a melhor evidência de que o resultado não é acaso de uma execução.

## Ambiente

| item | valor |
|---|---|
| CPUs / GOMAXPROCS | 16 / 16 |
| SO / arquitetura | linux amd64, kernel 7.2.10-1-cachyos |
| Go / Git | go1.27.3-X:nodwarf5 / git version 2.56.0 |
| repositório | `pose-dist`, 2889 commits, HEAD `57e20957ce49` |
| check pesado | `review-check spec:pose-governance-audit-skips-pure-git-reads` |
| consulta curta | `surface-check spec:pose-governance-audit-skips-pure-git-reads` |
| configuração do runtime | padrão: 1 slot pesado (fila 4), 2 slots curtos (fila 16), memo de 256 MiB |

## Cenários, 4ª execução (ms)

| cenário | amostras | mediana | mínimo | máximo | medianas da 1ª / 2ª / 3ª |
|---|---|---|---|---|---|
| cold_standalone | 3 | 25706 | 25643 | 26596 | 26781 / 26836 / 25684 |
| fallback_no_runtime | 3 | 25600 | 25577 | 25619 | 25616 / 26938 / 25536 |
| resident_first | 1 | 25677 | 25677 | 25677 | 26304 / 26924 / 25507 |
| resident_warm | 5 | 10662 | 10619 | 10667 | 11049 / 11275 / 10573 |
| restart_then_run | 3 | 25649 | 25633 | 25719 | 25754 / 26992 / 25558 |
| short_idle | 10 | 8 | 7 | 9 | 8 / 9 / 8 |
| short_under_heavy | 10 | 8 | 7 | 10 | 8 / 8 / 7 |
| two_heavy_wall | 1 | 21266 | 21266 | 21266 | - / - / 21197 |

## Limiares (4ª execução)

| limiar | resultado |
|---|---|
| T1 warm over cold | pass: 10662 ms / 25706 ms = 0.41 <= 0.60 |
| T2 restart inherits nothing | pass: 25649 ms / 25706 ms = 1.00 >= 0.80 |
| T3 short under heavy | pass: p95 10 ms <= 250 ms (idle p95 9 ms) |
| T4 runtime memory | pass: peak resident 550 MiB <= 1536 MiB |
| T5 output | pass: every hosted run, the review-check and the short query idle and under load, printed exactly what the standalone run printed (25610 and 89 bytes) |
| T6 fallback overhead | pass: -106 ms <= 500 ms |
| T7 two simultaneous heavy checks | pass: both through the runtime with the standalone output; wall 21266 ms <= 2.5 x 10662 ms |

## CPU e memória do runtime por cenário (4ª execução)

| cenário | CPU do processo do runtime (ms) | pico do RSS no cenário (MiB) | RSS ao fim (MiB) |
|---|---|---|---|
| primeira execução e 5 warm | 61200 | 550 | 498 |
| 10 consultas curtas, runtime ocioso | 0 | 498 | 498 |
| um check pesado com 10 consultas curtas ao lado | 10250 | 537 | 501 |
| dois checks pesados simultâneos | 20530 | 548 | 496 |

Pico do processo ao longo de toda a medição: 550 MiB. O tempo de CPU é o do processo do runtime (usuário mais sistema, de `/proc`, com o clock tick lido de `getconf CLK_TCK`); não inclui os processos `git` que ele inicia, e o das consultas curtas (arredondado em tiques de 10 ms) é 0 ms: elas custam menos que um tique cada. Dois checks pesados levaram 2.0 vezes uma execução warm: com um slot pesado o segundo espera o primeiro (espera máxima registrada 10619 ms), e a CPU do par é cerca do dobro da de um check, como a execução em série implica.

## Leitura

- O runtime aquecido responde o `review-check` em cerca de 10,7 s contra cerca de 25,7 s standalone (0.41 da cold). A primeira execução dentro do runtime é tão lenta quanto a standalone (o memo começa vazio), e um runtime reiniciado volta a ela (1.00 da cold): nada é herdado.
- Uma consulta curta leva cerca de 8 ms ociosa e cerca de 8 ms enquanto um check de 25 s roda na faixa pesada (p95 9 e 10 ms), com saída idêntica à standalone.
- Dois checks pesados simultâneos: os dois passam pelo runtime com a saída idêntica; o segundo espera a vez e o par leva cerca de 2 vezes um check warm.
- O fallback sem runtime custou -106 ms (ruído; o custo é desprezível).

## Suportado, configurado e efetivo

- Suportado: Linux e macOS; Windows roda sempre standalone. A medição foi só em Linux.
- Configurado: padrões acima; `--heavy-slots`, `--short-slots`, `--heavy-queue`, `--short-queue`, `--memo-mib` os mudam.
- Efetivo (métricas do status ao fim da fase de carga): faixa pesada com 1 slot e fila de 4, 9 admitidos, 0 recusados, espera máxima 10619 ms (o segundo dos dois checks simultâneos); faixa curta com 2 slots e fila de 16, 20 admitidos, 0 recusados, espera 0. A recusa por fila cheia é exercida pelo teste com comando lento injetado, não por esta medição.

## Limites da medição

Um ambiente, um repositório, 3 a 10 amostras por cenário; vale para o histórico medido. A carga pesada foi `review-check`; `artifact-check` e `closeout-check` não foram medidos sob carga. Não há cota nem afinidade de CPU no runtime: ele limita o que admite e guarda, e a CPU medida é a que o processo gastou, não um limite imposto. As consultas curtas foram sequenciais entre si.
