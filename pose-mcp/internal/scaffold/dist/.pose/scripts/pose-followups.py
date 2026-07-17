#!/usr/bin/env python3
"""Agregador de follow-ups POSE.

Varre `.pose/specs/*/spec.md`, extrai os follow-ups da seção
`Final Report > Follow-ups` e suas disposições, e produz uma visão única do
backlog. É uma ferramenta de descoberta (sempre exit 0); o gate de obrigação
vive em `./pose lint-spec`.

Disposições reconhecidas (espelham o template e o lint-spec):
  [open]                  ainda não triado (backlog vivo)
  [spawned: <slug>]       virou/alimentou uma nova spec
  [covered: <slug>]       já coberto por outra spec existente
  [duplicate: <slug>]     mesmo follow-up já triado em outra spec
  [done]                  resolvido direto, sem spec separada
  [wont-do: <motivo>]     descartado conscientemente

Modos:
  --open           (default) lista apenas follow-ups [open] ou sem disposição
  --all            lista todos os follow-ups com spec + disposição
  --json           saída machine-readable
  --similarity N   limiar 0..100 de similaridade léxica (default 60)

Reporta candidatos a near-duplicate: follow-ups de specs diferentes com
similaridade léxica (Jaccard de tokens + SequenceMatcher, ambos stdlib) acima
do limiar. São CANDIDATOS determinísticos para triagem — o julgamento semântico
(mesma intenção? outra spec cobre?) e a confirmação ficam na skill
pose-spec-closeout (camada LLM/agente), não neste script.
"""
from __future__ import annotations

import argparse
import difflib
import json
import pathlib
import re
import sys

# Stopwords PT-BR + ruído técnico comum, removidas do conjunto de tokens para
# o Jaccard ganhar sinal. Mantém-se a string normalizada inteira no SequenceMatcher.
STOPWORDS = frozenset(
    """
    a o as os um uma uns umas de da do das dos no na nos nas em para por com
    sem sob sobre e ou que se ao aos à às quando onde como qual quais ja já
    nao não mais menos entre ate até cada todo toda todos todas via fica ficam
    """.split()
)

DEFAULT_SIMILARITY = 60  # 0..100; abaixo disso não é candidato a near-duplicate.

SUBHEADING_RE = re.compile(r"^###\s+(.+?)\s*$")
HEADING_RE = re.compile(r"^##\s+\d+\.\s+(.+?)\s*$")
BULLET_RE = re.compile(r"^\s*-\s+(.*\S)\s*$")
DISPOSITION_RE = re.compile(r"^\[\s*([a-z-]+)\s*(?::\s*(.+?))?\s*\]\s*(.*)$")
HTML_COMMENT_BLOCK_RE = re.compile(r"<!--.*?-->", re.DOTALL)
FRONTMATTER_RE = re.compile(r"^---\s*\n(.*?)\n---\s*\n", re.DOTALL)


def parse_args(argv: list[str]) -> argparse.Namespace:
    parser = argparse.ArgumentParser(description="Agregador de follow-ups POSE")
    parser.add_argument("--specs-dir", required=True, help="Caminho de .pose/specs")
    group = parser.add_mutually_exclusive_group()
    group.add_argument("--open", action="store_true", help="Lista só follow-ups abertos (default)")
    group.add_argument("--all", action="store_true", help="Lista todos os follow-ups")
    parser.add_argument("--json", action="store_true", help="Saída JSON")
    parser.add_argument(
        "--similarity",
        type=int,
        default=DEFAULT_SIMILARITY,
        metavar="0..100",
        help=(
            "Limiar de similaridade léxica para candidatos a near-duplicate "
            f"(default {DEFAULT_SIMILARITY}). 100 = só texto idêntico."
        ),
    )
    return parser.parse_args(argv)


def parse_status(text: str) -> str:
    match = FRONTMATTER_RE.match(text)
    if not match:
        return "unset"
    for line in match.group(1).splitlines():
        if line.strip().startswith("status:"):
            value = line.split(":", 1)[1]
            value = re.sub(r"\s+#.*$", "", value).strip()
            return value or "unset"
    return "unset"


def extract_followups(text: str) -> list[tuple[str, str, str]]:
    """Retorna [(disposition, target, content)] dos bullets de Follow-ups."""
    body = HTML_COMMENT_BLOCK_RE.sub("", text)
    in_final_report = False
    in_followups = False
    out: list[tuple[str, str, str]] = []
    for line in body.splitlines():
        heading = HEADING_RE.match(line)
        if heading:
            in_final_report = heading.group(1).strip().lower().startswith("final report")
            in_followups = False
            continue
        if not in_final_report:
            continue
        sub = SUBHEADING_RE.match(line)
        if sub:
            in_followups = sub.group(1).strip().lower().startswith("follow-up")
            continue
        if not in_followups:
            continue
        bullet = BULLET_RE.match(line)
        if not bullet:
            continue
        content = bullet.group(1).strip()
        disp_match = DISPOSITION_RE.match(content)
        if disp_match:
            disposition = disp_match.group(1)
            target = (disp_match.group(2) or "").strip()
            text_only = disp_match.group(3).strip()
        else:
            disposition = ""
            target = ""
            text_only = content
        out.append((disposition, target, text_only))
    return out


def normalize(text: str) -> str:
    """String normalizada: minúsculas, sem marcação/pontuação, espaços colapsados."""
    lowered = text.lower().replace("`", " ")
    cleaned = re.sub(r"[^0-9a-zà-ÿ]+", " ", lowered)
    return re.sub(r"\s+", " ", cleaned).strip()


def content_tokens(norm: str) -> frozenset[str]:
    """Tokens de conteúdo (sem stopwords, len > 2) para o Jaccard."""
    return frozenset(t for t in norm.split() if len(t) > 2 and t not in STOPWORDS)


def similarity(a_norm: str, b_norm: str) -> float:
    """Similaridade léxica determinística em 0..1.

    Combina Jaccard de tokens de conteúdo (captura reordenação/paráfrase leve)
    com SequenceMatcher (captura sobreposição de sequência). Pega o maior dos
    dois — um follow-up reescrito com as mesmas palavras-chave pontua alto mesmo
    com ordem diferente; uma cópia verbatim pontua 1.0."""
    ta, tb = content_tokens(a_norm), content_tokens(b_norm)
    if ta and tb:
        jaccard = len(ta & tb) / len(ta | tb)
    else:
        jaccard = 0.0
    seq = difflib.SequenceMatcher(None, a_norm, b_norm).ratio()
    return max(jaccard, seq)


def cluster_near_duplicates(records: list[dict], threshold: float) -> list[list[dict]]:
    """Agrupa follow-ups de specs diferentes com similaridade >= threshold (union-find)."""
    n = len(records)
    parent = list(range(n))

    def find(i: int) -> int:
        while parent[i] != i:
            parent[i] = parent[parent[i]]
            i = parent[i]
        return i

    def union(i: int, j: int) -> None:
        parent[find(i)] = find(j)

    norms = [normalize(r["text"]) for r in records]
    for i in range(n):
        for j in range(i + 1, n):
            if records[i]["spec"] == records[j]["spec"]:
                continue
            if similarity(norms[i], norms[j]) >= threshold:
                union(i, j)

    groups: dict[int, list[dict]] = {}
    for idx, rec in enumerate(records):
        groups.setdefault(find(idx), []).append(rec)
    return [
        group
        for group in groups.values()
        if len({r["spec"] for r in group}) > 1
    ]


def main(argv: list[str]) -> int:
    args = parse_args(argv)
    specs_dir = pathlib.Path(args.specs_dir)
    if not specs_dir.is_dir():
        print(f"Erro: diretório de specs ausente: {specs_dir}", file=sys.stderr)
        return 2

    show_all = args.all
    records: list[dict] = []
    for spec_md in sorted(specs_dir.glob("*/spec.md")):
        slug = spec_md.parent.name
        text = spec_md.read_text(encoding="utf-8")
        status = parse_status(text)
        for disposition, target, content in extract_followups(text):
            if not content:
                continue  # bullet placeholder vazio
            records.append(
                {
                    "spec": slug,
                    "spec_status": status,
                    "raw_disposition": disposition,
                    "target": target,
                    "text": content,
                }
            )

    open_records = [r for r in records if r["raw_disposition"] in ("", "open")]

    # Candidatos a near-duplicate: follow-ups de specs diferentes com
    # similaridade léxica >= threshold. São CANDIDATOS — o julgamento semântico
    # ("é a mesma intenção? outra spec já cobre?") e a confirmação ficam na
    # skill pose-spec-closeout, não aqui.
    threshold = max(0.0, min(1.0, args.similarity / 100.0))
    collisions = cluster_near_duplicates(records, threshold)

    selected = records if show_all else open_records

    if args.json:
        payload = {
            "total": len(records),
            "open": len(open_records),
            "specs": len({r["spec"] for r in records}),
            "similarity_threshold": args.similarity,
            "items": selected,
            "near_duplicate_candidates": [
                {
                    "members": [
                        {"spec": r["spec"], "text": r["text"], "disposition": r["raw_disposition"] or "open"}
                        for r in group
                    ],
                    "specs": sorted({r["spec"] for r in group}),
                }
                for group in collisions
            ],
        }
        print(json.dumps(payload, ensure_ascii=False, indent=2))
        return 0

    label = "todos os follow-ups" if show_all else "follow-ups abertos"
    print(f"# POSE follow-ups — {label}")
    print(f"# total={len(records)} open={len(open_records)} specs={len({r['spec'] for r in records})}")
    print()
    if not selected:
        print("(nenhum)")
    for r in selected:
        disp = r["raw_disposition"] or "open"
        tag = f"[{disp}: {r['target']}]" if r["target"] else f"[{disp}]"
        print(f"- {r['spec']} {tag}")
        print(f"    {r['text']}")
    if collisions:
        print()
        print(
            f"## Candidatos a near-duplicate ({len(collisions)}) "
            f"— similaridade léxica >= {args.similarity}/100"
        )
        print("## Confirme na triagem (skill pose-spec-closeout): é a mesma intenção? outra spec já cobre?")
        for n, group in enumerate(collisions, 1):
            specs = ", ".join(sorted({r["spec"] for r in group}))
            print(f"\n[{n}] specs: {specs}")
            for r in group:
                disp = r["raw_disposition"] or "open"
                print(f"    - ({r['spec']} [{disp}]) {r['text'][:90]}")
    return 0


if __name__ == "__main__":
    sys.exit(main(sys.argv[1:]))
