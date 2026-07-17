#!/usr/bin/env python3
"""Sugere trilha canônica POSE para um tipo de tarefa.

Lê .pose/indexes/task-map.json e renderiza em texto humano (default) ou JSON.
Quando nenhum tipo é informado, lista todos os disponíveis.

Domínio pode ser:
  - explícito via --domain <d>
  - inferido via --path <p> contra .pose/indexes/repo-map.json
"""
from __future__ import annotations

import argparse
import json
import os
import pathlib
import sys

# Heurísticas de inferência de domínio. Ordem importa: primeiro match vence.
PATH_DOMAIN_HINTS = (
    ("k8s/", "k8s"),
    ("/charts/", "k8s"),
    ("/helm/", "k8s"),
)
LANGUAGE_DOMAIN_MAP = {
    "javascript": "frontend",
    "typescript": "frontend",
    "go": "backend-go",
}


def parse_args(argv: list[str]) -> argparse.Namespace:
    parser = argparse.ArgumentParser(description="Trilha canônica por tipo de tarefa")
    parser.add_argument("--task-map", required=True)
    parser.add_argument("--repo-map", default="", help="Caminho do repo-map.json (necessário para --path)")
    parser.add_argument("--repo-root", default="", help="Raiz do repo (para normalizar --path)")
    parser.add_argument("--task-type", default="", help="Tipo de tarefa (ex.: feature). Vazio = lista todos.")
    parser.add_argument("--json", action="store_true", help="Saída em JSON")
    parser.add_argument("--domain", default="", help="Filtra rules_by_domain (ex.: frontend, backend-go, k8s)")
    parser.add_argument("--path", default="", help="Caminho dentro do repo; tenta inferir o domínio")
    return parser.parse_args(argv)


def infer_domain(path: str, repo_root: str, repo_map_path: str) -> tuple[str, str]:
    """Retorna (domain, source). source ∈ {hint-path, repo-map, indef}."""
    if not path:
        return "", "indef"

    # Heurística direta no path (k8s/helm).
    norm = path.replace("\\", "/")
    if repo_root and norm.startswith(repo_root):
        norm = norm[len(repo_root):].lstrip("/")
    for hint, domain in PATH_DOMAIN_HINTS:
        if hint.strip("/") in norm.split("/") or hint in f"/{norm}/":
            return domain, "hint-path"

    # Lookup em repo-map.json: encontra o módulo cuja `path` é prefixo de norm.
    if not repo_map_path:
        return "", "indef"
    try:
        data = json.loads(pathlib.Path(repo_map_path).read_text(encoding="utf-8"))
    except (OSError, json.JSONDecodeError):
        return "", "indef"

    candidates = []
    for kind in ("apps", "services", "packages"):
        for item in data.get(kind, []):
            mod_path = item.get("path", "").strip("/")
            if not mod_path:
                continue
            if norm == mod_path or norm.startswith(mod_path + "/"):
                candidates.append((len(mod_path), item))
    if not candidates:
        return "", "indef"
    # Match mais específico (path mais longo) vence.
    candidates.sort(key=lambda c: c[0], reverse=True)
    item = candidates[0][1]

    declared_domain = (item.get("domain") or "").strip()
    if declared_domain and declared_domain != "unknown":
        # module-metadata pode usar 'frontend', 'backend', etc.
        mapped = {
            "backend": "backend-go",  # convenção do projeto: backend principal é Go
        }.get(declared_domain, declared_domain)
        return mapped, "repo-map"

    lang = (item.get("language") or "").lower()
    if lang in LANGUAGE_DOMAIN_MAP:
        return LANGUAGE_DOMAIN_MAP[lang], "repo-map"

    return "", "indef"


def render_human(name: str, task: dict, domain: str, domain_source: str, path_hint: str) -> str:
    lines = [f"# Trilha POSE — {name}", ""]
    desc = task.get("description", "")
    if desc:
        lines.append(desc)
        lines.append("")

    if path_hint:
        if domain:
            lines.append(f"- Path:      {path_hint}  →  domínio inferido: {domain} ({domain_source})")
        else:
            lines.append(f"- Path:      {path_hint}  →  domínio não inferido")

    lines.append(f"- Workflow:  {task.get('workflow') or '_n/a_'}")
    lines.append(f"- Skill:     {task.get('skill') or '_n/a_'}")

    base_rules = task.get("rules", [])
    rule_set = list(base_rules)
    rules_by_domain = task.get("rules_by_domain") or {}
    if domain:
        domain_rules = rules_by_domain.get(domain, [])
        if not domain_rules and domain not in rules_by_domain:
            available = sorted(rules_by_domain.keys())
            lines.append(
                f"- AVISO: domain '{domain}' não declarado em rules_by_domain "
                f"(disponíveis: {', '.join(available) or 'nenhum'})"
            )
        rule_set.extend(domain_rules)
    rule_paths = ", ".join(f".pose/rules/{r}.md" for r in rule_set) or "_nenhuma_"
    lines.append(f"- Rules:     {rule_paths}")

    if not domain and rules_by_domain:
        lines.append(
            "  (use --domain <"
            + "|".join(sorted(rules_by_domain.keys()))
            + "> ou --path <p> para rules adicionais)"
        )

    lines.append(f"- Spec:      {task.get('requires_spec')!r}")
    lines.append(f"- ADR:       {task.get('requires_adr')!r}")
    lines.append(f"- Knowledge: consume={task.get('knowledge_consume')}, produce={task.get('knowledge_produce')!r}")
    lines.append(f"- Validação: {task.get('validation') or '_n/a_'}")
    return "\n".join(lines)


def main(argv: list[str]) -> int:
    args = parse_args(argv)
    path = pathlib.Path(args.task_map)
    if not path.is_file():
        print(f"Erro: task-map ausente: {path}", file=sys.stderr)
        return 2
    try:
        data = json.loads(path.read_text(encoding="utf-8"))
    except (OSError, json.JSONDecodeError) as exc:
        print(f"Erro: falha ao parsear task-map: {exc}", file=sys.stderr)
        return 2

    tasks = data.get("tasks", {})
    if not isinstance(tasks, dict):
        print("Erro: task-map.tasks deve ser objeto", file=sys.stderr)
        return 2

    if not args.task_type:
        if args.json:
            print(json.dumps(sorted(tasks), indent=2, ensure_ascii=False))
            return 0
        print("Tipos de tarefa disponíveis:")
        for name in sorted(tasks):
            desc = tasks[name].get("description", "")
            print(f"  - {name}: {desc}")
        return 0

    task = tasks.get(args.task_type)
    if task is None:
        available = ", ".join(sorted(tasks))
        print(
            f"Erro: tipo de tarefa desconhecido: '{args.task_type}' "
            f"(disponíveis: {available})",
            file=sys.stderr,
        )
        return 2

    # Resolução de domínio: --domain explícito vence sobre --path inferido.
    domain = args.domain
    domain_source = "explicit" if domain else ""
    if not domain and args.path:
        domain, domain_source = infer_domain(args.path, args.repo_root, args.repo_map)

    if args.json:
        payload = {"name": args.task_type, **task}
        if domain:
            payload["domain_effective"] = domain
            payload["domain_source"] = domain_source
            extra = (task.get("rules_by_domain") or {}).get(domain, [])
            payload["rules_effective"] = list(task.get("rules", [])) + extra
        if args.path:
            payload["path_input"] = args.path
        print(json.dumps(payload, indent=2, ensure_ascii=False))
        return 0

    print(render_human(args.task_type, task, domain, domain_source, args.path))
    return 0


if __name__ == "__main__":
    sys.exit(main(sys.argv[1:]))
