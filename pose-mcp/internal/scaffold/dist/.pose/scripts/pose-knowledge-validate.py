#!/usr/bin/env python3
"""Valida frontmatter dos artefatos em .pose/knowledge/ conforme a rule
[.pose/rules/knowledge-governance.md].

Verifica em cada `*.md` (excluindo README.md, .gitkeep e archive/):
  - frontmatter YAML delimitado por --- nas primeiras linhas
  - `type` ∈ {handoff, note, decision-log}
  - `owner` presente e não placeholder (<owner>)
  - `sensitivity` ∈ {public-internal, restricted}
  - `created_at`, `last_reviewed_at`, `expires_at` em formato ISO YYYY-MM-DD
  - `expires_at` distância de `created_at` ≤ 90 dias

Saída para consumo por shell:
  knowledge.schema.errors=<N>
  knowledge.schema.warnings=<N>
  knowledge.schema.checked=<N>
  (e linhas `[ERRO] <path>: <motivo>` ou `[AVISO] <path>: <motivo>` em stderr)

Exit codes:
  0 — sem erros (avisos não bloqueiam)
  1 — pelo menos 1 erro de schema
  2 — erro de uso/IO
"""
from __future__ import annotations

import argparse
import datetime
import pathlib
import re
import sys

ALLOWED_TYPES = {"handoff", "note", "decision-log"}
ALLOWED_SENSITIVITY = {"public-internal", "restricted"}
REQUIRED_FIELDS = (
    "type", "slug", "owner", "sensitivity",
    "created_at", "last_reviewed_at", "expires_at",
)
PLACEHOLDER_RE = re.compile(r"^<[^>]+>$")
DATE_RE = re.compile(r"^\d{4}-\d{2}-\d{2}$")
TTL_MAX_DAYS = 90


def parse_args(argv: list[str]) -> argparse.Namespace:
    parser = argparse.ArgumentParser(description="Validação de frontmatter para .pose/knowledge/")
    parser.add_argument("--knowledge-dir", required=True, help="Caminho absoluto para .pose/knowledge/")
    return parser.parse_args(argv)


def read_frontmatter(path: pathlib.Path) -> tuple[dict[str, str], str | None]:
    """Retorna (campos, erro). Campos são pares chave→valor extraídos do YAML
    superficial entre --- e ---. Não interpreta listas/mapas aninhados.
    """
    try:
        content = path.read_text(encoding="utf-8")
    except OSError as exc:
        return {}, f"falha ao ler arquivo: {exc}"

    lines = content.splitlines()
    if not lines or lines[0].strip() != "---":
        return {}, "frontmatter ausente (esperado --- na primeira linha)"

    end_idx = None
    for idx, line in enumerate(lines[1:], start=1):
        if line.strip() == "---":
            end_idx = idx
            break
    if end_idx is None:
        return {}, "frontmatter não fechado (faltou --- de encerramento)"

    fields: dict[str, str] = {}
    for raw in lines[1:end_idx]:
        stripped = raw.rstrip()
        if not stripped or stripped.startswith("#"):
            continue
        # Aceita apenas chaves no primeiro nível ("key: value"). Linhas
        # indentadas (subcampos de source_refs etc.) são ignoradas aqui.
        if raw.startswith((" ", "\t")):
            continue
        if ":" not in stripped:
            continue
        key, _, value = stripped.partition(":")
        key = key.strip()
        value = value.strip().strip('"').strip("'")
        fields[key] = value
    return fields, None


def validate_file(path: pathlib.Path) -> tuple[list[str], list[str]]:
    """Retorna (erros, avisos) para o arquivo."""
    errors: list[str] = []
    warnings: list[str] = []

    fields, error = read_frontmatter(path)
    if error:
        errors.append(error)
        return errors, warnings

    for required in REQUIRED_FIELDS:
        value = fields.get(required, "")
        if not value:
            errors.append(f"campo obrigatório ausente: {required}")
            continue
        if PLACEHOLDER_RE.match(value):
            errors.append(f"campo {required} ainda contém placeholder: {value}")

    if fields.get("type") and fields["type"] not in ALLOWED_TYPES and not PLACEHOLDER_RE.match(fields["type"]):
        errors.append(
            f"type inválido: {fields['type']} (use: {', '.join(sorted(ALLOWED_TYPES))})"
        )

    if (
        fields.get("sensitivity")
        and fields["sensitivity"] not in ALLOWED_SENSITIVITY
        and not PLACEHOLDER_RE.match(fields["sensitivity"])
    ):
        errors.append(
            f"sensitivity inválida: {fields['sensitivity']} (use: {', '.join(sorted(ALLOWED_SENSITIVITY))})"
        )

    dates: dict[str, datetime.date | None] = {}
    for date_field in ("created_at", "last_reviewed_at", "expires_at"):
        value = fields.get(date_field, "")
        if not value or PLACEHOLDER_RE.match(value):
            dates[date_field] = None
            continue
        if not DATE_RE.match(value):
            errors.append(f"{date_field} fora do formato ISO YYYY-MM-DD: {value}")
            dates[date_field] = None
            continue
        try:
            dates[date_field] = datetime.date.fromisoformat(value)
        except ValueError:
            errors.append(f"{date_field} não é data válida: {value}")
            dates[date_field] = None

    created = dates.get("created_at")
    expires = dates.get("expires_at")
    reviewed = dates.get("last_reviewed_at")

    if created and expires and expires < created:
        errors.append(
            f"expires_at ({expires.isoformat()}) anterior a created_at ({created.isoformat()})"
        )

    if created and expires:
        ttl = (expires - created).days
        if ttl > TTL_MAX_DAYS:
            errors.append(
                f"TTL {ttl}d excede limite de {TTL_MAX_DAYS}d definido na rule"
            )

    if reviewed and created and reviewed < created:
        warnings.append(
            f"last_reviewed_at ({reviewed.isoformat()}) anterior a created_at ({created.isoformat()})"
        )

    return errors, warnings


def iter_knowledge_files(root: pathlib.Path):
    """Itera apenas artefatos elegíveis (md de primeiro nível, excluindo
    README, .gitkeep e arquivos em archive/)."""
    for entry in sorted(root.iterdir()):
        if not entry.is_file():
            continue
        if entry.suffix.lower() != ".md":
            continue
        if entry.name.lower() == "readme.md":
            continue
        yield entry


def main(argv: list[str]) -> int:
    args = parse_args(argv)
    knowledge_dir = pathlib.Path(args.knowledge_dir)
    if not knowledge_dir.is_dir():
        print(f"Erro: diretório de knowledge ausente: {knowledge_dir}", file=sys.stderr)
        return 2

    total_errors = 0
    total_warnings = 0
    total_checked = 0

    for path in iter_knowledge_files(knowledge_dir):
        total_checked += 1
        errors, warnings = validate_file(path)
        rel = path.name
        for err in errors:
            print(f"[ERRO] {rel}: {err}", file=sys.stderr)
            total_errors += 1
        for warn in warnings:
            print(f"[AVISO] {rel}: {warn}", file=sys.stderr)
            total_warnings += 1

    print(f"knowledge.schema.errors={total_errors}")
    print(f"knowledge.schema.warnings={total_warnings}")
    print(f"knowledge.schema.checked={total_checked}")

    return 1 if total_errors > 0 else 0


if __name__ == "__main__":
    sys.exit(main(sys.argv[1:]))
