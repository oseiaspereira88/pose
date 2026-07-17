#!/usr/bin/env python3
"""Compara o registro corrente de um relatório POSE contra o último entry do JSONL.

Lê todos os campos do registro como flags `--key value` (em vez de heredoc com
substituição shell, evitando aspas dobradas frágeis) e imprime ao stdout em
formato `chave=valor` (uma por linha) para consumo via bash.

Saída:
    status=first-run|stable|changed
    prev=<iso-timestamp ou vazio>
    count=<sequência inteira>
    stable_hash=<sha256>
    change=<descrição>   (zero ou mais linhas)
"""
from __future__ import annotations

import argparse
import hashlib
import json
import os
import sys


STABLE_KEYS = (
    "task_slug", "spec", "report_type", "workflow",
    "rules", "validation_profile", "context",
)


def parse_args(argv: list[str]) -> argparse.Namespace:
    parser = argparse.ArgumentParser(description="Comparação histórica de relatórios POSE")
    parser.add_argument("--history-file", required=True)
    parser.add_argument("--task", default="")
    parser.add_argument("--task-slug", default="")
    parser.add_argument("--spec", default="")
    parser.add_argument("--report-type", default="standard")
    parser.add_argument("--workflow", default="")
    parser.add_argument("--rules", default="")
    parser.add_argument("--validation-profile", default="")
    parser.add_argument("--risk", default="")
    parser.add_argument("--context", default="")
    return parser.parse_args(argv)


def main(argv: list[str]) -> int:
    args = parse_args(argv)

    record = {
        "task": args.task,
        "task_slug": args.task_slug,
        "spec": args.spec,
        "report_type": args.report_type,
        "workflow": args.workflow,
        "rules": args.rules,
        "validation_profile": args.validation_profile,
        "risk": args.risk,
        "context": args.context,
    }
    stable_payload = {k: record.get(k, "") for k in STABLE_KEYS}
    record["stable_hash"] = hashlib.sha256(
        json.dumps(stable_payload, sort_keys=True).encode()
    ).hexdigest()

    previous = None
    if os.path.exists(args.history_file):
        with open(args.history_file, encoding="utf-8") as f:
            lines = [line.strip() for line in f if line.strip()]
        if lines:
            try:
                previous = json.loads(lines[-1])
            except json.JSONDecodeError:
                previous = None

    changes: list[str] = []
    if previous:
        for key in STABLE_KEYS:
            prev_value = previous.get(key, "")
            curr_value = record.get(key, "")
            if prev_value != curr_value:
                changes.append(f"{key}: '{prev_value}' -> '{curr_value}'")
        if previous.get("stable_hash") == record["stable_hash"]:
            print("status=stable")
        elif changes:
            print("status=changed")
            for change in changes:
                print(f"change={change}")
        else:
            print("status=changed")
    else:
        print("status=first-run")

    print("prev=" + (previous.get("generated_at", "") if previous else ""))
    print(
        "count="
        + (str((previous.get("sequence", 0) + 1) if previous else 1))
    )
    print("stable_hash=" + record["stable_hash"])
    return 0


if __name__ == "__main__":
    sys.exit(main(sys.argv[1:]))
