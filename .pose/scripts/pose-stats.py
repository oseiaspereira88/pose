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
import html
import json
import os
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
    parser.add_argument("--html", action="store_true",
                        help="Gera relatório HTML auto-contido")
    parser.add_argument("--out", help="Arquivo de saída para --html")
    parser.add_argument("--specs-dir", help="Diretório de specs para métricas de lead time")
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


def spec_metrics(specs_dir: pathlib.Path | None) -> dict[str, object]:
    """Return best-effort lead-time and open-follow-up metrics from specs."""
    if specs_dir is None or not specs_dir.is_dir():
        return {"completed": 0, "lead_time_days": None, "open_followups": 0}
    lead_times: list[int] = []
    open_followups = 0
    for spec in specs_dir.glob("*/spec.md"):
        try:
            text = spec.read_text(encoding="utf-8")
        except OSError:
            continue
        open_followups += text.count("- [open]")
        values = {
            key.strip(): value.strip()
            for line in text.splitlines()
            if ":" in line
            for key, value in [line.split(":", 1)]
            if key.strip() in {"created_at", "completed_at"}
        }
        created = parse_iso(values.get("created_at", ""))
        completed = parse_iso(values.get("completed_at", ""))
        if created and completed and completed >= created:
            lead_times.append((completed - created).days)
    return {
        "completed": len(lead_times),
        "lead_time_days": round(sum(lead_times) / len(lead_times), 1) if lead_times else None,
        "open_followups": open_followups,
    }


def render_html(rows: list[dict], task_rows: list[dict], scanned: int, skipped_invalid: int, metrics: dict[str, object]) -> str:
    """Render a CSP-safe, standalone report; all dynamic values are escaped."""
    table_rows = "".join(
        "<tr><td>{}</td><td>{}</td><td>{}</td><td>{}</td><td>{}</td></tr>".format(
            html.escape(str(row["key"])), row["pass"], row["fail"], row["partial"], row["total"])
        for row in rows
    ) or "<tr><td colspan=\"5\">No history records found.</td></tr>"
    recurring_rows = "".join(
        "<tr><td>{}</td><td>{}</td><td>{}</td></tr>".format(
            html.escape(str(row["key"])), row["total"], row["fail"] + row["partial"])
        for row in task_rows if row["total"] >= 2
    ) or "<tr><td colspan=\"3\">No recurrence candidates found.</td></tr>"
    lead = metrics["lead_time_days"]
    lead_text = "unavailable" if lead is None else f"{lead} days"
    return f"""<!doctype html>
<html lang=\"en\"><head><meta charset=\"utf-8\"><meta name=\"viewport\" content=\"width=device-width, initial-scale=1\">
<meta http-equiv=\"Content-Security-Policy\" content=\"default-src 'none'; style-src 'unsafe-inline'\">
<title>POSE local insights</title><style>body{{font-family:system-ui;margin:2rem;max-width:960px}}table{{border-collapse:collapse;width:100%}}th,td{{border:1px solid #ccc;padding:.45rem;text-align:left}}.cards{{display:flex;gap:1rem;flex-wrap:wrap}}.card{{border:1px solid #ccc;padding:1rem;min-width:10rem}}</style></head>
<body><h1>POSE local insights</h1><p>Offline report generated from repository-local data.</p>
<div class=\"cards\"><div class=\"card\"><b>History records</b><br>{scanned}</div><div class=\"card\"><b>Invalid records skipped</b><br>{skipped_invalid}</div><div class=\"card\"><b>Open follow-ups</b><br>{metrics['open_followups']}</div><div class=\"card\"><b>Average completed-spec lead time</b><br>{lead_text}</div></div>
<h2>Outcomes by workflow</h2><table><thead><tr><th>Workflow</th><th>Pass</th><th>Fail</th><th>Partial</th><th>Total</th></tr></thead><tbody>{table_rows}</tbody></table>
<h2>Recurrence candidates</h2><table><thead><tr><th>Task</th><th>Occurrences</th><th>Fail or partial</th></tr></thead><tbody>{recurring_rows}</tbody></table>
</body></html>"""


def write_atomic(path: pathlib.Path, content: str) -> None:
    path.parent.mkdir(parents=True, exist_ok=True)
    temporary = path.with_name(f".{path.name}.tmp")
    temporary.write_text(content, encoding="utf-8")
    os.replace(temporary, path)


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

    if args.html:
        output = pathlib.Path(args.out) if args.out else history_dir.parent / "pose-stats.html"
        metrics = spec_metrics(pathlib.Path(args.specs_dir) if args.specs_dir else None)
        task_rows, _, _, _ = aggregate(history_dir, GROUP_KEYS["task"], args.since_days)
        write_atomic(output, render_html(rows, task_rows, scanned, skipped_invalid, metrics))
        print(f"stats.html={output}")
        return 0

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
