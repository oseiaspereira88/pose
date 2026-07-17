#!/usr/bin/env python3
"""Grafo de dependências entre specs POSE (pose-spec-dependencies).

O frontmatter POSE é deliberadamente flat (key: value por linha). `depends_on`
é uma lista inline separada por vírgulas, com refs tipadas:

    depends_on: outra-spec, milestone:<roadmap>/<id>, roadmap:<slug>
    priority: 2

Modos:
  --check   valida sintaxe das refs, existência de specs referenciadas,
            priority inteiro >= 0, auto-dependência e aciclicidade do grafo
            spec→spec. Refs `milestone:`/`roadmap:` recebem apenas validação
            sintática enquanto `.pose/roadmaps/` não existir (spec
            pose-roadmap-artifact liga a resolução real).
  --emit    imprime o grafo em JSON determinístico (insumo de
            .pose/indexes/spec-graph.json para pose-mcp/Conductor).

Saída de --check para consumo por shell (mesmo contrato dos demais gates):
  linhas `[ERRO] ...` e `[AVISO] ...` em stdout.

Exit codes: 0 ok · 1 erros no --check · 2 uso/IO.
"""
from __future__ import annotations

import argparse
import json
import pathlib
import re
import sys

SLUG_RE = re.compile(r"^[a-z0-9][a-z0-9._-]*$")
MILESTONE_REF_RE = re.compile(r"^milestone:([a-z0-9][a-z0-9._-]*)/([a-z0-9][a-z0-9._-]*)$")
ROADMAP_REF_RE = re.compile(r"^roadmap:([a-z0-9][a-z0-9._-]*)$")
FRONTMATTER_RE = re.compile(r"^---\s*\n(.*?)\n---\s*\n", re.DOTALL)


def parse_frontmatter(text: str) -> dict[str, str]:
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
        value = re.sub(r"\s+#.*$", "", value).strip()
        fields[key.strip()] = value
    return fields


def parse_depends_on(value: str) -> list[str]:
    """Lista inline: 'a, b' ou '[a, b]'. Vazio → []."""
    value = value.strip()
    if value.startswith("[") and value.endswith("]"):
        value = value[1:-1]
    return [item.strip() for item in value.split(",") if item.strip()]


def ref_kind(ref: str) -> str:
    """'spec' | 'milestone' | 'roadmap' | 'invalid'."""
    if MILESTONE_REF_RE.match(ref):
        return "milestone"
    if ROADMAP_REF_RE.match(ref):
        return "roadmap"
    if ":" in ref:
        return "invalid"  # prefixo de tipo desconhecido
    if SLUG_RE.match(ref):
        return "spec"
    return "invalid"


def load_specs(specs_dir: pathlib.Path) -> dict[str, dict]:
    """slug → {status, priority(raw), depends_on(list), path}."""
    specs: dict[str, dict] = {}
    if not specs_dir.is_dir():
        return specs
    candidates: list[tuple[str, pathlib.Path]] = []
    for entry in sorted(specs_dir.iterdir()):
        if entry.is_dir():
            spec_md = entry / "spec.md"
            if spec_md.is_file():
                candidates.append((entry.name, spec_md))
        elif entry.suffix == ".md" and entry.name.lower() != "readme.md":
            candidates.append((entry.stem, entry))
    for slug, path in candidates:
        try:
            fm = parse_frontmatter(path.read_text(encoding="utf-8"))
        except OSError:
            continue
        specs[slug] = {
            "status": fm.get("status", ""),
            "priority_raw": fm.get("priority", ""),
            "depends_on": parse_depends_on(fm.get("depends_on", "")),
            "path": str(path),
        }
    return specs


def parse_roadmap(path: pathlib.Path) -> dict:
    """Parseia o artefato de roadmap (pose-roadmap-artifact): frontmatter flat
    + seções `## Milestone: <id>` com bullets flat (- after/- target_*/- specs)."""
    text = path.read_text(encoding="utf-8")
    fm = parse_frontmatter(text)
    roadmap = {
        "slug": fm.get("slug", path.stem),
        "status": fm.get("status", ""),
        "depends_on": parse_depends_on(fm.get("depends_on", "")),
        "milestones": [],
        "path": str(path),
    }
    current = None
    for line in text.split("\n"):
        stripped = line.strip()
        if stripped.startswith("## Milestone:"):
            current = {
                "id": stripped.split(":", 1)[1].strip(),
                "after": [],
                "target_start": "",
                "target_due": "",
                "specs": [],
            }
            roadmap["milestones"].append(current)
            continue
        if stripped.startswith("## "):
            current = None
            continue
        if current is None or not stripped.startswith("- "):
            continue
        key, _, value = stripped[2:].partition(":")
        key = key.strip()
        value = re.sub(r"\s+#.*$", "", value).strip()
        if key == "after":
            current["after"] = parse_depends_on(value)
        elif key in ("target_start", "target_due"):
            current[key] = value
        elif key == "specs":
            current["specs"] = parse_depends_on(value)
    return roadmap


def load_roadmaps(root: pathlib.Path) -> dict[str, dict]:
    roadmaps: dict[str, dict] = {}
    roadmaps_dir = root / "roadmaps"
    if not roadmaps_dir.is_dir():
        return roadmaps
    for path in sorted(roadmaps_dir.glob("*.md")):
        if path.name.lower() == "readme.md":
            continue
        rm = parse_roadmap(path)
        roadmaps[rm["slug"]] = rm
    return roadmaps


VALID_ROADMAP_STATUS = ("draft", "active", "done", "abandoned")
DATE_RE = re.compile(r"^\d{4}-\d{2}-\d{2}$")


def check_roadmaps(specs: dict[str, dict], roadmaps: dict[str, dict]) -> int:
    """Valida roadmaps (R3/R4): membership única em ativos, milestones/DAG,
    datas e resolução de refs tipadas nas specs."""
    errors = 0
    spec_owner: dict[str, str] = {}
    milestone_ids: set[str] = set()
    for slug in sorted(roadmaps):
        rm = roadmaps[slug]
        if rm["status"] and rm["status"] not in VALID_ROADMAP_STATUS:
            print(f"[ERRO] roadmap {slug}: status inválido: '{rm['status']}' (use {'|'.join(VALID_ROADMAP_STATUS)})")
            errors += 1
        for dep in rm["depends_on"]:
            if dep == slug:
                print(f"[ERRO] roadmap {slug}: depends_on referencia o próprio roadmap")
                errors += 1
            elif dep not in roadmaps:
                print(f"[ERRO] roadmap {slug}: depends_on referencia roadmap inexistente: '{dep}'")
                errors += 1
        seen_ms: set[str] = set()
        for ms in rm["milestones"]:
            ms_id = ms["id"]
            if not ms_id or not SLUG_RE.match(ms_id):
                print(f"[ERRO] roadmap {slug}: milestone com id inválido: '{ms_id}'")
                errors += 1
                continue
            if ms_id in seen_ms:
                print(f"[ERRO] roadmap {slug}: milestone duplicado: '{ms_id}'")
                errors += 1
            seen_ms.add(ms_id)
            milestone_ids.add(f"{slug}/{ms_id}")
            if ms["target_start"] and not DATE_RE.match(ms["target_start"]):
                print(f"[ERRO] roadmap {slug}/{ms_id}: target_start inválido: '{ms['target_start']}' (YYYY-MM-DD)")
                errors += 1
            if ms["target_due"] and not DATE_RE.match(ms["target_due"]):
                print(f"[ERRO] roadmap {slug}/{ms_id}: target_due inválido: '{ms['target_due']}' (YYYY-MM-DD)")
                errors += 1
            if ms["target_start"] and ms["target_due"] and ms["target_start"] > ms["target_due"]:
                print(f"[ERRO] roadmap {slug}/{ms_id}: target_start > target_due")
                errors += 1
            for spec_ref in ms["specs"]:
                if spec_ref not in specs:
                    print(f"[ERRO] roadmap {slug}/{ms_id}: spec inexistente: '{spec_ref}'")
                    errors += 1
                elif rm["status"] == "active":
                    owner = spec_owner.get(spec_ref)
                    if owner and owner != slug:
                        print(f"[ERRO] spec '{spec_ref}' em dois roadmaps ativos: {owner} e {slug}")
                        errors += 1
                    spec_owner[spec_ref] = slug
        for ms in rm["milestones"]:
            for ref in ms["after"]:
                if ref.startswith("spec:"):
                    target = ref[len("spec:"):]
                    if target not in specs:
                        print(f"[ERRO] roadmap {slug}/{ms['id']}: after referencia spec inexistente: '{target}'")
                        errors += 1
                elif ref not in seen_ms:
                    print(f"[ERRO] roadmap {slug}/{ms['id']}: after referencia milestone inexistente: '{ref}'")
                    errors += 1
        # DAG entre milestones do roadmap
        ms_edges = {ms["id"]: [r for r in ms["after"] if not r.startswith("spec:")] for ms in rm["milestones"]}
        cycle = find_cycle(ms_edges)
        if cycle:
            print(f"[ERRO] roadmap {slug}: ciclo entre milestones: {' → '.join(cycle)}")
            errors += 1
    # Refs tipadas nas specs resolvem contra roadmaps existentes (R4). Sem
    # nenhum roadmap no repo, a validação segue apenas sintática (feature não
    # adotada — comportamento da Fase 1 preservado).
    if roadmaps:
        for slug in sorted(specs):
            for ref in specs[slug]["depends_on"]:
                if ref.startswith("roadmap:"):
                    target = ref[len("roadmap:"):]
                    if target not in roadmaps:
                        print(f"[ERRO] {slug}: depends_on referencia roadmap inexistente: '{ref}'")
                        errors += 1
                elif ref.startswith("milestone:"):
                    target = ref[len("milestone:"):]
                    if target not in milestone_ids:
                        print(f"[ERRO] {slug}: depends_on referencia milestone inexistente: '{ref}'")
                        errors += 1
    # DAG entre roadmaps
    rm_edges = {slug: [d for d in rm["depends_on"] if d in roadmaps] for slug, rm in roadmaps.items()}
    cycle = find_cycle(rm_edges)
    if cycle:
        print(f"[ERRO] ciclo de dependência entre roadmaps: {' → '.join(cycle)}")
        errors += 1
    return errors


def find_cycle(edges: dict[str, list[str]]) -> list[str] | None:
    """Retorna um ciclo (lista de slugs) no grafo spec→spec, ou None."""
    WHITE, GRAY, BLACK = 0, 1, 2
    color = {node: WHITE for node in edges}
    stack: list[str] = []

    def visit(node: str) -> list[str] | None:
        color[node] = GRAY
        stack.append(node)
        for dep in edges.get(node, []):
            if dep not in color:
                continue
            if color[dep] == GRAY:
                return stack[stack.index(dep):] + [dep]
            if color[dep] == WHITE:
                cycle = visit(dep)
                if cycle:
                    return cycle
        stack.pop()
        color[node] = BLACK
        return None

    for node in sorted(edges):
        if color[node] == WHITE:
            cycle = visit(node)
            if cycle:
                return cycle
    return None


def check(specs: dict[str, dict]) -> int:
    errors = 0
    spec_edges: dict[str, list[str]] = {slug: [] for slug in specs}
    for slug in sorted(specs):
        info = specs[slug]
        raw_priority = info["priority_raw"]
        if raw_priority:
            try:
                if int(raw_priority) < 0:
                    raise ValueError
            except ValueError:
                print(f"[ERRO] {slug}: priority deve ser inteiro >= 0 (encontrado: '{raw_priority}')")
                errors += 1
        seen: set[str] = set()
        for ref in info["depends_on"]:
            if ref in seen:
                print(f"[AVISO] {slug}: dependência duplicada em depends_on: '{ref}'")
                continue
            seen.add(ref)
            kind = ref_kind(ref)
            if kind == "invalid":
                print(
                    f"[ERRO] {slug}: ref inválida em depends_on: '{ref}' "
                    f"(use <spec-slug>, milestone:<roadmap>/<id> ou roadmap:<slug>)"
                )
                errors += 1
            elif kind == "spec":
                if ref == slug:
                    print(f"[ERRO] {slug}: depends_on referencia a própria spec")
                    errors += 1
                elif ref not in specs:
                    print(f"[ERRO] {slug}: depends_on referencia spec inexistente: '{ref}'")
                    errors += 1
                else:
                    spec_edges[slug].append(ref)
            # milestone:/roadmap: → só sintaxe até .pose/roadmaps/ existir.
    cycle = find_cycle(spec_edges)
    if cycle:
        print(f"[ERRO] ciclo de dependência entre specs: {' → '.join(cycle)}")
        errors += 1
    return errors


def emit(specs: dict[str, dict]) -> dict:
    nodes = {}
    edges = []
    for slug in sorted(specs):
        info = specs[slug]
        priority = None
        if info["priority_raw"]:
            try:
                priority = int(info["priority_raw"])
            except ValueError:
                priority = None
        nodes[slug] = {
            "status": info["status"],
            "priority": priority,
            "depends_on": info["depends_on"],
        }
        for ref in info["depends_on"]:
            kind = ref_kind(ref)
            if kind == "invalid":
                continue
            edges.append({"from": slug, "to": ref, "type": kind})
    return {"schemaVersion": 1, "specs": nodes, "edges": edges}


def main(argv: list[str]) -> int:
    parser = argparse.ArgumentParser(description="Grafo de dependências de specs POSE")
    parser.add_argument("--specs-dir", required=True, help="Caminho de .pose/specs")
    mode = parser.add_mutually_exclusive_group(required=True)
    mode.add_argument("--check", action="store_true", help="Valida refs, priority e aciclicidade (specs + roadmaps)")
    mode.add_argument("--emit", action="store_true", help="Imprime o grafo em JSON")
    mode.add_argument("--emit-roadmaps", action="store_true", help="Imprime os roadmaps parseados em JSON")
    args = parser.parse_args(argv)

    specs_dir = pathlib.Path(args.specs_dir)
    if not specs_dir.is_dir():
        print(f"[ERRO] diretório de specs inexistente: {specs_dir}", file=sys.stderr)
        return 2
    specs = load_specs(specs_dir)
    roadmaps = load_roadmaps(specs_dir.parent)

    if args.check:
        errors = check(specs) + check_roadmaps(specs, roadmaps)
        return 1 if errors else 0
    if args.emit_roadmaps:
        print(json.dumps({"schemaVersion": 1, "roadmaps": roadmaps}, indent=2, ensure_ascii=False))
        return 0
    print(json.dumps(emit(specs), indent=2, ensure_ascii=False))
    return 0


if __name__ == "__main__":
    sys.exit(main(sys.argv[1:]))
