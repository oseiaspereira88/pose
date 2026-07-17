#!/usr/bin/env python3
"""Detector de recorrência sobre o histórico de relatórios POSE.

Lê todos os .jsonl em .pose/reports/history/ e identifica task_slugs que
ocorreram acima de `--threshold` vezes na janela `--window-days`.

Heurísticas:
- Foco em outcomes não-passes (fail, partial, unknown). Sucessos repetidos não
  caracterizam recorrência problemática.
- Agrupamento por task_slug + report_type.
- Inclui última ocorrência, contagem de fails vs partials, e workflow citado.

Saída para consumo por shell:
  recurrence.window_days=<N>
  recurrence.threshold=<T>
  recurrence.records_scanned=<N>
  recurrence.flagged_keys=<N>
  (linhas `[RECORRENTE] <key>: <stats>` em stderr quando encontradas)

Exit codes:
  0 — sem chaves recorrentes acima do threshold
  1 — ao menos uma chave flagged (consumido por `./pose recurrence-check --strict`)
  2 — erro de uso/IO
"""
from __future__ import annotations

import argparse
import datetime
import json
import pathlib
import sys


def parse_args(argv: list[str]) -> argparse.Namespace:
    parser = argparse.ArgumentParser(description="Detector de recorrência sobre history JSONL")
    parser.add_argument("--history-dir", required=True)
    parser.add_argument("--window-days", type=int, default=14)
    parser.add_argument("--threshold", type=int, default=3)
    parser.add_argument(
        "--include-pass",
        action="store_true",
        help="Inclui outcome=pass na contagem (default: ignora passes).",
    )
    return parser.parse_args(argv)


def parse_iso(value: str) -> datetime.datetime | None:
    if not value:
        return None
    try:
        return datetime.datetime.fromisoformat(value.replace("Z", "+00:00"))
    except ValueError:
        return None


def main(argv: list[str]) -> int:
    args = parse_args(argv)
    history_dir = pathlib.Path(args.history_dir)
    if not history_dir.is_dir():
        print(f"Erro: diretório de history ausente: {history_dir}", file=sys.stderr)
        return 2

    now = datetime.datetime.now(datetime.timezone.utc)
    window_start = now - datetime.timedelta(days=args.window_days)

    # key = (task_slug, report_type) → list of records
    buckets: dict[tuple[str, str], list[dict]] = {}
    scanned = 0

    for jsonl in sorted(history_dir.glob("*.jsonl")):
        try:
            for line_no, raw in enumerate(jsonl.read_text(encoding="utf-8").splitlines(), start=1):
                raw = raw.strip()
                if not raw:
                    continue
                try:
                    record = json.loads(raw)
                except json.JSONDecodeError:
                    print(
                        f"[AVISO] linha inválida em {jsonl.name}:{line_no} — ignorada",
                        file=sys.stderr,
                    )
                    continue
                scanned += 1

                generated_at = parse_iso(record.get("generated_at", ""))
                if generated_at is None or generated_at < window_start:
                    continue

                outcome = record.get("outcome", "unknown") or "unknown"
                if not args.include_pass and outcome == "pass":
                    continue

                key = (
                    record.get("task_slug", "<unknown>"),
                    record.get("report_type", "standard"),
                )
                buckets.setdefault(key, []).append(record)
        except OSError as exc:
            print(f"[AVISO] falha ao ler {jsonl}: {exc}", file=sys.stderr)
            continue

    flagged = 0
    for key in sorted(buckets):
        records = buckets[key]
        if len(records) < args.threshold:
            continue
        flagged += 1
        task_slug, report_type = key
        outcomes_count: dict[str, int] = {}
        latest_workflow = ""
        latest_at = ""
        for rec in records:
            outcome = rec.get("outcome", "unknown") or "unknown"
            outcomes_count[outcome] = outcomes_count.get(outcome, 0) + 1
            if rec.get("workflow"):
                latest_workflow = rec["workflow"]
            if rec.get("generated_at", "") > latest_at:
                latest_at = rec["generated_at"]
        outcomes_summary = ", ".join(
            f"{k}={v}" for k, v in sorted(outcomes_count.items())
        )
        msg = (
            f"[RECORRENTE] {task_slug} ({report_type}): {len(records)} runs em "
            f"{args.window_days}d; outcomes={outcomes_summary}; "
            f"último={latest_at or '?'}"
        )
        if latest_workflow:
            msg += f"; workflow={latest_workflow}"
        print(msg, file=sys.stderr)

    print(f"recurrence.window_days={args.window_days}")
    print(f"recurrence.threshold={args.threshold}")
    print(f"recurrence.records_scanned={scanned}")
    print(f"recurrence.flagged_keys={flagged}")

    return 1 if flagged > 0 else 0


if __name__ == "__main__":
    sys.exit(main(sys.argv[1:]))
