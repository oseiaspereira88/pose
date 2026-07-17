#!/usr/bin/env python3
"""Agrega métricas estruturadas a partir de .pose/reports/history/*.jsonl.

Permite decisões objetivas (promover check optional → required, identificar
workflows instáveis, comparar contextos ci vs manual) baseadas em outcomes
acumulados em vez de scraping de markdown.

Subcomandos:
  outcomes [--by workflow|task|context] [--since-days N] [--json]
  workflows [--since-days N] [--json]       (atalho para outcomes --by workflow)
  tasks     [--since-days N] [--json]       (atalho para outcomes --by task)

Saída humana: tabela com colunas pass/fail/partial/unknown/total/rate.
Saída --json: lista de objetos com mesmas chaves.
"""
from __future__ import annotations

import argparse
import datetime
import json
import pathlib
import sys

GROUP_KEYS = {
    "workflow": "workflow",
    "task": "task_slug",
    "context": "context",
}
OUTCOMES = ("pass", "fail", "partial", "skipped", "unknown")


def parse_args(argv: list[str]) -> argparse.Namespace:
    parser = argparse.ArgumentParser(description="Estatísticas sobre history JSONL")
    parser.add_argument("--history-dir", required=True)
    parser.add_argument("--by", choices=sorted(GROUP_KEYS), default="workflow",
                        help="Chave de agrupamento (default: workflow)")
    parser.add_argument("--since-days", type=int, default=0,
                        help="Filtra registros gerados nos últimos N dias (0 = sem filtro)")
    parser.add_argument("--json", action="store_true")
    return parser.parse_args(argv)


def parse_iso(value: str) -> datetime.datetime | None:
    if not value:
        return None
    try:
        return datetime.datetime.fromisoformat(value.replace("Z", "+00:00"))
    except ValueError:
        return None


def aggregate(history_dir: pathlib.Path, group_field: str, since_days: int):
    cutoff = None
    if since_days > 0:
        cutoff = datetime.datetime.now(datetime.timezone.utc) - datetime.timedelta(days=since_days)

    buckets: dict[str, dict[str, int]] = {}
    scanned = 0
    skipped_by_window = 0
    skipped_invalid = 0

    for jsonl in sorted(history_dir.glob("*.jsonl")):
        try:
            content = jsonl.read_text(encoding="utf-8")
        except OSError as exc:
            print(f"[AVISO] falha ao ler {jsonl}: {exc}", file=sys.stderr)
            continue
        for raw in content.splitlines():
            raw = raw.strip()
            if not raw:
                continue
            try:
                record = json.loads(raw)
            except json.JSONDecodeError:
                skipped_invalid += 1
                continue
            scanned += 1

            if cutoff is not None:
                generated_at = parse_iso(record.get("generated_at", ""))
                if generated_at is None or generated_at < cutoff:
                    skipped_by_window += 1
                    continue

            key = record.get(group_field) or "_unset_"
            outcome = record.get("outcome", "unknown") or "unknown"
            if outcome not in OUTCOMES:
                outcome = "unknown"

            bucket = buckets.setdefault(key, {o: 0 for o in OUTCOMES})
            bucket[outcome] += 1

    rows = []
    for key in sorted(buckets):
        counts = buckets[key]
        total = sum(counts.values())
        graded_total = total - counts["unknown"] - counts["skipped"]
        pass_rate = (counts["pass"] / graded_total) if graded_total > 0 else None
        rows.append({
            "key": key,
            **counts,
            "total": total,
            "pass_rate": pass_rate,
        })

    return rows, scanned, skipped_by_window, skipped_invalid


def render_table(rows: list[dict], group_label: str) -> str:
    if not rows:
        return f"_Nenhum registro para agrupamento por {group_label}._"

    header_label = group_label.upper()
    key_width = max(len(header_label), max(len(r["key"]) for r in rows))
    headers = [header_label, "PASS", "FAIL", "PART", "SKIP", "UNK", "TOT", "RATE"]
    widths = [key_width, 5, 5, 5, 5, 5, 5, 7]

    def fmt_row(values):
        return " | ".join(str(v).ljust(w) for v, w in zip(values, widths))

    lines = [fmt_row(headers), fmt_row(["-" * w for w in widths])]
    for r in rows:
        rate = "n/a" if r["pass_rate"] is None else f"{r['pass_rate']*100:.0f}%"
        lines.append(fmt_row([
            r["key"], r["pass"], r["fail"], r["partial"],
            r["skipped"], r["unknown"], r["total"], rate,
        ]))
    return "\n".join(lines)


def main(argv: list[str]) -> int:
    args = parse_args(argv)
    history_dir = pathlib.Path(args.history_dir)
    if not history_dir.is_dir():
        print(f"Erro: history dir ausente: {history_dir}", file=sys.stderr)
        return 2

    group_field = GROUP_KEYS[args.by]
    rows, scanned, skipped_window, skipped_invalid = aggregate(
        history_dir, group_field, args.since_days
    )

    if args.json:
        payload = {
            "group_by": args.by,
            "since_days": args.since_days,
            "records_scanned": scanned,
            "records_skipped_by_window": skipped_window,
            "records_skipped_invalid": skipped_invalid,
            "rows": rows,
        }
        print(json.dumps(payload, indent=2, ensure_ascii=False))
        return 0

    print(f"# Stats por {args.by}"
          + (f" (últimos {args.since_days} dia(s))" if args.since_days else "")
          + "\n")
    print(render_table(rows, args.by))
    print()
    print(f"stats.records_scanned={scanned}")
    print(f"stats.records_skipped_by_window={skipped_window}")
    print(f"stats.records_skipped_invalid={skipped_invalid}")
    print(f"stats.groups={len(rows)}")
    return 0


if __name__ == "__main__":
    sys.exit(main(sys.argv[1:]))
