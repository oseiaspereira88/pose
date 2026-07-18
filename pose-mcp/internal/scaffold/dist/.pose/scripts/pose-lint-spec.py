#!/usr/bin/env python3
"""Lint de specs POSE.

Detecta seções vazias ou que contém apenas placeholders (HTML comments) em
`.pose/specs/<slug>/spec.md`. Permite identificar specs marcadas como prontas
mas com `Validation` ou `Final Report` ainda esqueléticos.

Por padrão verifica todas as seções obrigatórias e marca `Decisions` como
opcional. Use `--required-only` para checar só obrigatórias.

Gate de ciclo de vida (quando o frontmatter declara `status: done`):
  - `completed_at` deve estar preenchido;
  - todo follow-up listado em `Final Report > Follow-ups` deve ter disposição
    válida entre colchetes (`[open]`, `[spawned: <slug>]`, `[covered: <slug>]`,
    `[duplicate: <slug>]`, `[done]`, `[wont-do: <motivo>]`);
  - o alvo de `spawned`/`covered`/`duplicate` deve referenciar uma spec
    existente (e não a própria) — barra 'covered falso' por typo/slug morto.
Specs sem frontmatter/`status` (formato legado) não disparam o gate.

Saída para consumo por shell:
  spec.path=<path>
  spec.status=<status|unset>
  spec.sections.total=<N>
  spec.sections.filled=<N>
  spec.sections.skeleton=<N>
  spec.sections.empty=<N>
  spec.required.missing=<N>
  spec.followups.total=<N>
  spec.followups.open=<N>
  spec.lifecycle.failures=<N>
  (linhas `[ERRO] <slug>: ...` em stderr)

Exit codes:
  0 — todas as seções obrigatórias têm conteúdo e o gate de ciclo de vida passou
  1 — pelo menos 1 seção obrigatória vazia/esquelética ou falha de gate
  2 — erro de uso/IO
"""
from __future__ import annotations

import argparse
from collections import Counter
from datetime import datetime, timezone
import pathlib
import re
import sys

REQUIRED_SECTIONS = (
    "Intent",
    "Requirements",
    "Technical Plan",
    "Tasks",
    "Validation",
    "Final Report",
)
OPTIONAL_SECTIONS = ("Decisions",)

VALID_STATUS = (
    "draft",
    "in-progress",
    "done",
    "blocked",
    "superseded",
    "abandoned",
)

# Disposições de follow-up. As que exigem alvo (slug/motivo) listadas à parte.
VALID_DISPOSITIONS = ("open", "spawned", "covered", "duplicate", "done", "wont-do")
DISPOSITIONS_REQUIRING_TARGET = ("spawned", "covered", "duplicate", "wont-do")

# Heading do template: "## 1. Intent", "## 6. Validation", etc.
HEADING_RE = re.compile(r"^##\s+\d+\.\s+(.+?)\s*$")
SUBHEADING_RE = re.compile(r"^###\s+(.+?)\s*$")
PLACEHOLDER_LINE_RE = re.compile(r"^\s*<!--.*-->\s*$")
EMPTY_BULLET_RE = re.compile(r"^\s*-\s*$")
META_LINE_RE = re.compile(r"^\s*-\s*[A-Za-zÀ-ÿ ]+:\s*$")  # "- Data:", "- Comando:"
HTML_COMMENT_BLOCK_RE = re.compile(r"<!--.*?-->", re.DOTALL)
BULLET_RE = re.compile(r"^\s*-\s+(.*\S)\s*$")
DISPOSITION_RE = re.compile(r"^\[\s*([a-z-]+)\s*(?::\s*(.+?))?\s*\]\s*(.*)$")
FRONTMATTER_RE = re.compile(r"^---\s*\n(.*?)\n---\s*\n", re.DOTALL)


def parse_args(argv: list[str]) -> argparse.Namespace:
    parser = argparse.ArgumentParser(description="Lint de spec.md POSE")
    parser.add_argument("--spec", required=True, help="Caminho para spec.md")
    parser.add_argument(
        "--required-only",
        action="store_true",
        help="Checa apenas seções obrigatórias (default: também avisa em opcionais)",
    )
    parser.add_argument(
        "--ready-check",
        action="store_true",
        help="Definition of Ready (pose-definition-of-ready): gate de ENTRADA — "
        "Intent/Requirements/Technical Plan preenchidos, acceptance criteria com "
        "IDs (- R<N>:) e depends_on sintaticamente válido. Não exige "
        "Validation/Final Report (a spec ainda não executou).",
    )
    parser.add_argument("--ears", action="store_true",
                        help="Exige que acceptance criteria usem sintaxe EARS")
    return parser.parse_args(argv)


def parse_frontmatter(text: str) -> dict[str, str]:
    """Extrai frontmatter YAML simples (key: value) do topo do arquivo.

    Aceita apenas pares escalares de primeiro nível, ignorando comentários
    inline (`# ...`). Retorna {} quando não há frontmatter."""
    match = FRONTMATTER_RE.match(text)
    if not match:
        return {}
    fields: dict[str, str] = {}
    for line in match.group(1).splitlines():
        if not line.strip() or line.lstrip().startswith("#"):
            continue
        if ":" not in line:
            continue
        key, _, value = line.partition(":")
        # Remove comentário inline do valor ("done   # estado").
        value = re.sub(r"\s+#.*$", "", value).strip()
        fields[key.strip()] = value
    return fields


def strip_html_comments(text: str) -> str:
    return HTML_COMMENT_BLOCK_RE.sub("", text)


def is_content_line(line: str) -> bool:
    """Retorna True se a linha representa conteúdo de fato, não placeholder."""
    stripped = line.strip()
    if not stripped:
        return False
    if PLACEHOLDER_LINE_RE.match(line):
        return False
    if EMPTY_BULLET_RE.match(line):
        return False
    if SUBHEADING_RE.match(line):
        return False
    if META_LINE_RE.match(line):
        # "- Data:" (sem valor) — conta como placeholder.
        return False
    if stripped == "---":
        return False
    return True


def split_sections(text: str) -> dict[str, list[str]]:
    """Divide o documento por headings nível ## numerados. Retorna {section_name: [lines]}."""
    sections: dict[str, list[str]] = {}
    current_name: str | None = None
    current_lines: list[str] = []
    for line in text.splitlines():
        match = HEADING_RE.match(line)
        if match:
            if current_name is not None:
                sections[current_name] = current_lines
            current_name = match.group(1).strip()
            current_lines = []
        else:
            current_lines.append(line)
    if current_name is not None:
        sections[current_name] = current_lines
    return sections


def classify_section(lines: list[str]) -> str:
    """Retorna 'filled' | 'skeleton' | 'empty'."""
    content_count = sum(1 for line in lines if is_content_line(line))
    has_any_line = any(line.strip() for line in lines)
    if content_count > 0:
        return "filled"
    if has_any_line:
        return "skeleton"
    return "empty"


def extract_followups(final_report_lines: list[str]) -> list[str]:
    """Retorna os bullets sob o subheading `### Follow-ups` do Final Report."""
    bullets: list[str] = []
    in_followups = False
    for line in final_report_lines:
        sub = SUBHEADING_RE.match(line)
        if sub:
            in_followups = sub.group(1).strip().lower().startswith("follow-up")
            continue
        if not in_followups:
            continue
        bullet = BULLET_RE.match(line)
        if bullet:
            bullets.append(bullet.group(1).strip())
    return bullets


# Disposições cujo alvo é um slug de spec (validável). `wont-do` carrega um
# motivo em texto livre, não um slug — fica de fora da checagem de existência.
DISPOSITIONS_WITH_SLUG_TARGET = ("spawned", "covered", "duplicate")


def collect_spec_slugs(specs_dir: pathlib.Path) -> set[str]:
    """Slugs de specs existentes (dirs com spec.md em .pose/specs/)."""
    if not specs_dir.is_dir():
        return set()
    return {p.parent.name for p in specs_dir.glob("*/spec.md")}


def parse_depends_on(value: str) -> list[str]:
    """Lista inline do frontmatter: 'a, b' ou '[a, b]'. Vazio → [].

    Mantenha em sincronia com .pose/scripts/pose-spec-graph.py (fonte da
    validação estrutural do grafo — aqui só resolvemos status de specs irmãs)."""
    value = value.strip()
    if value.startswith("[") and value.endswith("]"):
        value = value[1:-1]
    return [item.strip() for item in value.split(",") if item.strip()]


def sibling_spec_status(specs_dir: pathlib.Path, slug: str) -> str | None:
    """Status do frontmatter de outra spec, ou None se não existe/não parseia."""
    spec_md = specs_dir / slug / "spec.md"
    if not spec_md.is_file():
        return None
    try:
        fm = parse_frontmatter(spec_md.read_text(encoding="utf-8"))
    except OSError:
        return None
    return fm.get("status", "").strip() or None


def lint_followup_disposition(
    content: str,
    known_slugs: set[str] | None = None,
    current_slug: str | None = None,
) -> tuple[str | None, str | None]:
    """Valida a disposição de um follow-up.

    Retorna (disposition, erro). `disposition` é None quando não há tag.
    `erro` é None quando válido. Quando `known_slugs` é passado, valida que o
    alvo de spawned/covered/duplicate referencia uma spec existente e não a
    própria spec (anti-drift: barra 'covered falso' por typo/slug inexistente)."""
    match = DISPOSITION_RE.match(content)
    if not match:
        return None, "sem disposição (esperado prefixo [open|spawned|covered|duplicate|done|wont-do])"
    disposition = match.group(1)
    target = (match.group(2) or "").strip()
    if disposition not in VALID_DISPOSITIONS:
        return disposition, f"disposição inválida: [{disposition}]"
    if disposition in DISPOSITIONS_REQUIRING_TARGET and not target:
        kind = "motivo" if disposition == "wont-do" else "slug"
        return disposition, f"disposição [{disposition}] exige {kind} (use [{disposition}: <{kind}>])"
    if known_slugs is not None and disposition in DISPOSITIONS_WITH_SLUG_TARGET:
        if current_slug is not None and target == current_slug:
            return disposition, f"disposição [{disposition}] aponta para a própria spec ({target})"
        if target not in known_slugs:
            return disposition, f"disposição [{disposition}: {target}] aponta para spec inexistente"
    return disposition, None


# --- Definition of Ready (pose-definition-of-ready) ---

READY_SECTIONS = ("Intent", "Requirements", "Technical Plan")
# spec-requirements-traceability: '- R<N>: texto' with an optional bracketed
# criticality before the separator ('- R3 [alta]: texto'). Existing specs
# using the plain form (no bracket) keep matching unchanged — the bracket
# group is non-capturing-optional, not a new requirement.
ACCEPTANCE_ID_RE = re.compile(r"^\s*-\s*R(\d+)\s*(?:\[(\w+)\])?\s*[:—-]")
# EARS forms: ubiquitous, event-driven, state-driven, optional-feature, and
# unwanted behavior.  The trailing behavior is deliberately free-form.
EARS_RE = re.compile(
    r"^(?:The\s+\S+(?:\s+\S+){0,8}\s+shall\s+.+|"
    r"When\s+.+,\s+the\s+\S+(?:\s+\S+){0,8}\s+shall\s+.+|"
    r"While\s+.+,\s+the\s+\S+(?:\s+\S+){0,8}\s+shall\s+.+|"
    r"Where\s+.+,\s+the\s+\S+(?:\s+\S+){0,8}\s+shall\s+.+|"
    r"If\s+.+,\s+then\s+the\s+\S+(?:\s+\S+){0,8}\s+shall\s+.+)$",
    re.IGNORECASE,
)
DEP_SLUG_RE = re.compile(r"^[a-z0-9][a-z0-9._-]*$")
DEP_MILESTONE_RE = re.compile(r"^milestone:[a-z0-9][a-z0-9._-]*/[a-z0-9][a-z0-9._-]*$")
DEP_ROADMAP_RE = re.compile(r"^roadmap:[a-z0-9][a-z0-9._-]*$")


def parse_requirement_ids(requirements_lines: list[str]) -> list[tuple[str, str | None]]:
    """Extracts (R-ID, criticality) pairs from Requirements bullets in
    document order — criticality is None when the bullet has no '[...]'.
    """
    ids = []
    for line in requirements_lines:
        m = ACCEPTANCE_ID_RE.match(line)
        if m:
            ids.append((f"R{m.group(1)}", m.group(2)))
    return ids


def check_requirement_ids(slug: str, sections: dict[str, list[str]]) -> int:
    """spec-requirements-traceability R1: R-IDs must be unique within a spec.
    IDs are stable once published (removing a criterion should mark it
    retired in prose, not free its ID for reuse) — duplicates within the same
    document are always a mistake, so this is a hard failure, not a warning.
    """
    failures = 0
    seen: dict[str, int] = {}
    for rid, _criticality in parse_requirement_ids(sections.get("Requirements", [])):
        seen[rid] = seen.get(rid, 0) + 1
    for rid, count in sorted(seen.items()):
        if count > 1:
            print(
                f"[ERRO] {slug}: R-ID duplicado: {rid} aparece {count} vezes em Requirements",
                file=sys.stderr,
            )
            failures += 1
    return failures


def check_ears(slug: str, sections: dict[str, list[str]]) -> int:
    failures = 0
    for line in sections.get("Requirements", []):
        match = ACCEPTANCE_ID_RE.match(line)
        if not match:
            continue
        criterion = line[match.end():].strip()
        if not EARS_RE.match(criterion):
            print(f"[ERRO] {slug}: {match.group(0).strip()} deve usar sintaxe EARS", file=sys.stderr)
            failures += 1
    return failures


def check_canonical_heading_uniqueness(slug: str, text: str) -> int:
    """Rejeita headings canônicos repetidos que seriam sobrescritos no parser.

    Seções numeradas são chaves únicas do contrato editorial. `Follow-ups`
    também é único porque alimenta o backlog vivo e duas ocorrências fariam
    itens desaparecerem silenciosamente de `extract_followups`.
    """
    failures = 0
    section_names = [
        match.group(1).strip().casefold()
        for line in text.splitlines()
        if (match := HEADING_RE.match(line))
    ]
    for name, count in sorted(Counter(section_names).items()):
        if count > 1:
            print(
                f"[ERRO] {slug}: heading canônico duplicado: {name} aparece {count} vezes",
                file=sys.stderr,
            )
            failures += 1

    followups_count = sum(
        1
        for line in text.splitlines()
        if (match := SUBHEADING_RE.match(line))
        and match.group(1).strip().casefold().startswith("follow-up")
    )
    if followups_count > 1:
        print(
            f"[ERRO] {slug}: heading canônico duplicado: Follow-ups aparece {followups_count} vezes",
            file=sys.stderr,
        )
        failures += 1
    return failures


def check_lifecycle_dates(slug: str, frontmatter: dict[str, str]) -> int:
    """Valida timestamps ISO e impede conclusão anterior à criação.

    O acervo legado usa tanto `YYYY-MM-DD` quanto timestamps RFC3339 e alguns
    frontmatters antigos mantêm aspas. Todos representam instantes ISO válidos.
    """
    failures = 0
    parsed: dict[str, datetime] = {}
    for field in ("created_at", "completed_at"):
        value = frontmatter.get(field, "").strip().strip("\"'")
        if not value:
            continue
        try:
            normalized = value[:-1] + "+00:00" if value.endswith("Z") else value
            instant = datetime.fromisoformat(normalized)
            if instant.tzinfo is None:
                instant = instant.replace(tzinfo=timezone.utc)
            parsed[field] = instant
        except ValueError:
            print(
                f"[ERRO] {slug}: {field} deve usar ISO 8601: '{value}'",
                file=sys.stderr,
            )
            failures += 1
    if (
        "created_at" in parsed
        and "completed_at" in parsed
        and parsed["completed_at"] < parsed["created_at"]
    ):
        print(
            f"[ERRO] {slug}: completed_at ({parsed['completed_at']}) anterior a "
            f"created_at ({parsed['created_at']})",
            file=sys.stderr,
        )
        failures += 1
    return failures


def ready_check(slug: str, frontmatter: dict[str, str], sections: dict[str, list[str]]) -> int:
    """Gate de entrada: retorna o número de violações de DoR (0 = ready)."""
    failures = 0
    for name in READY_SECTIONS:
        if name not in sections or classify_section(sections[name]) != "filled":
            print(f"[ERRO] {slug}: DoR: seção {name} ausente/vazia/esquelética", file=sys.stderr)
            failures += 1
    criteria = [
        line
        for line in sections.get("Requirements", [])
        if ACCEPTANCE_ID_RE.match(line)
    ]
    if not criteria:
        print(
            f"[ERRO] {slug}: DoR: nenhum acceptance criterion com ID estável "
            f"(use bullets '- R<N>: ...' em Requirements)",
            file=sys.stderr,
        )
        failures += 1
    for ref in parse_depends_on(frontmatter.get("depends_on", "")):
        if DEP_SLUG_RE.match(ref) or DEP_MILESTONE_RE.match(ref) or DEP_ROADMAP_RE.match(ref):
            continue
        print(f"[ERRO] {slug}: DoR: ref inválida em depends_on: '{ref}'", file=sys.stderr)
        failures += 1
    print(f"spec.ready={'true' if failures == 0 else 'false'}")
    print(f"spec.ready.failures={failures}")
    return failures


def main(argv: list[str]) -> int:
    args = parse_args(argv)
    spec_path = pathlib.Path(args.spec)
    if not spec_path.is_file():
        print(f"Erro: spec ausente: {spec_path}", file=sys.stderr)
        return 2

    raw_text = spec_path.read_text(encoding="utf-8")
    frontmatter = parse_frontmatter(raw_text)
    text = strip_html_comments(raw_text)
    sections = split_sections(text)
    slug = spec_path.parent.name

    if args.ready_check:
        return 1 if ready_check(slug, frontmatter, sections) else 0

    targets = list(REQUIRED_SECTIONS)
    if not args.required_only:
        targets += list(OPTIONAL_SECTIONS)

    total = 0
    filled = 0
    skeleton = 0
    empty = 0
    required_missing = 0

    for section_name in targets:
        is_required = section_name in REQUIRED_SECTIONS
        if section_name not in sections:
            if is_required:
                print(f"[ERRO] {slug}: seção obrigatória ausente: {section_name}", file=sys.stderr)
                required_missing += 1
            else:
                print(f"[AVISO] {slug}: seção opcional ausente: {section_name}", file=sys.stderr)
            continue
        total += 1
        status = classify_section(sections[section_name])
        if status == "filled":
            filled += 1
        elif status == "skeleton":
            skeleton += 1
            level = "ERRO" if is_required else "AVISO"
            print(
                f"[{level}] {slug}: {section_name}: esqueleto "
                f"(apenas placeholders/comentários)",
                file=sys.stderr,
            )
            if is_required:
                required_missing += 1
        else:
            empty += 1
            level = "ERRO" if is_required else "AVISO"
            print(
                f"[{level}] {slug}: {section_name}: vazia",
                file=sys.stderr,
            )
            if is_required:
                required_missing += 1

    # --- Ciclo de vida (frontmatter) ---
    spec_status = frontmatter.get("status", "").strip() or "unset"
    lifecycle_failures = 0

    if spec_status != "unset" and spec_status not in VALID_STATUS:
        print(
            f"[ERRO] {slug}: status inválido no frontmatter: '{spec_status}' "
            f"(use {'|'.join(VALID_STATUS)})",
            file=sys.stderr,
        )
        lifecycle_failures += 1

    lifecycle_failures += check_lifecycle_dates(slug, frontmatter)
    lifecycle_failures += check_canonical_heading_uniqueness(slug, text)

    # Dependências não satisfeitas (pose-spec-dependencies): aviso, nunca gate —
    # o enforcement de execução é do Conductor/Harness, não do lint.
    if spec_status == "in-progress":
        specs_dir = spec_path.parent.parent
        for dep in parse_depends_on(frontmatter.get("depends_on", "")):
            if ":" in dep:
                continue  # refs milestone:/roadmap: são opacas para o lint
            dep_status = sibling_spec_status(specs_dir, dep)
            if dep_status is not None and dep_status != "done":
                print(
                    f"[AVISO] {slug}: in-progress com dependência não satisfeita: "
                    f"'{dep}' (status: {dep_status})",
                    file=sys.stderr,
                )

    requirement_id_failures = check_requirement_ids(slug, sections)
    ears_failures = check_ears(slug, sections) if args.ears else 0

    followups = extract_followups(sections.get("Final Report", []))
    followups_open = 0
    known_slugs = collect_spec_slugs(spec_path.parent.parent)

    if spec_status == "done":
        if not frontmatter.get("completed_at", "").strip():
            print(
                f"[ERRO] {slug}: status: done exige 'completed_at' preenchido no frontmatter",
                file=sys.stderr,
            )
            lifecycle_failures += 1
        for content in followups:
            disposition, err = lint_followup_disposition(content, known_slugs, slug)
            if err is not None:
                snippet = content[:60] + ("…" if len(content) > 60 else "")
                print(f"[ERRO] {slug}: follow-up sem disposição válida: {err} → \"{snippet}\"", file=sys.stderr)
                lifecycle_failures += 1
            elif disposition == "open":
                followups_open += 1
    else:
        # Em specs não-done, conta open apenas para visibilidade (não é gate).
        for content in followups:
            disposition, _ = lint_followup_disposition(content)
            if disposition == "open" or disposition is None:
                followups_open += 1

    print(f"spec.path={spec_path}")
    print(f"spec.status={spec_status}")
    print(f"spec.sections.total={total}")
    print(f"spec.sections.filled={filled}")
    print(f"spec.sections.skeleton={skeleton}")
    print(f"spec.sections.empty={empty}")
    print(f"spec.required.missing={required_missing}")
    print(f"spec.followups.total={len(followups)}")
    print(f"spec.followups.open={followups_open}")
    print(f"spec.lifecycle.failures={lifecycle_failures}")
    print(f"spec.requirements.ids={len(parse_requirement_ids(sections.get('Requirements', [])))}")
    print(f"spec.requirements.duplicate_failures={requirement_id_failures}")
    print(f"spec.ears.failures={ears_failures}")

    return 1 if (required_missing > 0 or lifecycle_failures > 0 or requirement_id_failures > 0 or ears_failures > 0) else 0


if __name__ == "__main__":
    sys.exit(main(sys.argv[1:]))
