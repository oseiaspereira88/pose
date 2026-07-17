#!/usr/bin/env python3
"""Descoberta de módulos e expansão da matriz de validação POSE.

Lê `.pose/indexes/validation-matrix.json` e emite, para cada módulo elegível,
uma linha pipe-separada consumida por `pose-validate.sh`:

    <abs_module>|<stack>|<mode>|<severity>|<command>

Argumentos posicionais:
    root          Raiz do repositório.
    matrix_path   Caminho do validation-matrix.json.
    mode_arg      "strict", "tolerant" ou "" (usa default da matriz).
    stack_filter  Filtro opcional por stack (node|go|rust|java) ou "".
    module_filter Caminho absoluto do módulo único a executar ou "".
"""
from __future__ import annotations

import json
import os
import sys


def main() -> int:
    if len(sys.argv) < 6:
        print("erro interno: argumentos insuficientes para pose-validate-discover.py", file=sys.stderr)
        return 2

    root, matrix_path, mode_arg, stack_filter, module_filter = sys.argv[1:6]

    with open(matrix_path, encoding="utf-8") as f:
        matrix = json.load(f)

    mode_default = matrix.get("defaults", {}).get("mode", "strict")
    mode = mode_arg or mode_default
    stack_filter = stack_filter.strip().lower()
    module_filter = os.path.normpath(module_filter) if module_filter else ""

    prune = {
        ".git", "node_modules", "vendor", ".venv", ".pnpm-store",
        "dist", "build", ".next", "target", "coverage",
    }
    stack_markers = {
        "node": ["package.json"],
        "go": ["go.mod"],
        "rust": ["Cargo.toml"],
        "java": [
            "pom.xml", "build.gradle", "build.gradle.kts",
            "settings.gradle", "settings.gradle.kts",
        ],
    }

    def discover() -> list[tuple[str, str]]:
        found: list[tuple[str, str]] = []
        for dirpath, dirnames, filenames in os.walk(root):
            dirnames[:] = [d for d in dirnames if d not in prune]
            for stack, markers in stack_markers.items():
                if any(mark in filenames for mark in markers):
                    found.append((os.path.normpath(dirpath), stack))
                    break
        for rel, override in matrix.get("moduleOverrides", {}).items():
            module = os.path.normpath(os.path.join(root, rel))
            if os.path.isdir(module):
                found.append((module, override.get("stack", "contract")))
        return sorted(set(found))

    def check_when(module: str, rule: dict) -> bool:
        when = rule.get("when")
        if not when:
            return True
        file_exists = when.get("fileExists")
        if file_exists and not os.path.exists(os.path.join(module, file_exists)):
            return False
        file_not_exists = when.get("fileNotExists")
        if file_not_exists and os.path.exists(os.path.join(module, file_not_exists)):
            return False
        return True

    for module, detected_stack in discover():
        rel = os.path.relpath(module, root).replace("\\", "/")
        override = matrix.get("moduleOverrides", {}).get(rel, {})
        stack = override.get("stack", detected_stack)
        if stack_filter and stack != stack_filter:
            continue
        if module_filter and os.path.normpath(module) != module_filter:
            continue
        mod_mode = override.get("mode", mode)
        if override.get("replaceDefaultChecks", False):
            checks = []
        else:
            checks = list(matrix.get("stacks", {}).get(stack, {}).get("checks", []))
        checks += list(override.get("checks", []))
        for rule in checks:
            if not check_when(module, rule):
                continue
            cmd = rule.get("command", "").strip()
            severity = rule.get("severity", "required")
            if cmd:
                print(f"{module}|{stack}|{mod_mode}|{severity}|{cmd}")

    return 0


if __name__ == "__main__":
    sys.exit(main())
