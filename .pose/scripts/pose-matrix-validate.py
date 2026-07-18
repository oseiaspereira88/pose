#!/usr/bin/env python3
"""Valida o schema de .pose/indexes/validation-matrix.json.

Pega typos como `severty` em vez de `severity` que silenciosamente downgradam
o comportamento de `./pose validate` (campos desconhecidos seriam ignorados
e a severidade cairia para o default 'required').

Saída para consumo por shell:
  matrix.errors=<N>
  matrix.warnings=<N>
  matrix.stacks=<N>
  matrix.module_overrides=<N>
  (linhas `[ERRO] <path>: <motivo>` ou `[AVISO] <path>: <motivo>` em stderr)

Exit codes:
  0 — schema OK (avisos não bloqueiam)
  1 — pelo menos 1 erro de schema
  2 — erro de uso/IO
"""
from __future__ import annotations

import argparse
import json
import pathlib
import sys

ALLOWED_MODES = {"strict", "tolerant"}
ALLOWED_SEVERITIES = {"required", "optional"}
ALLOWED_CHECK_KEYS = {"name", "command", "program", "args", "env", "severity", "when"}
ALLOWED_WHEN_KEYS = {"fileExists", "fileNotExists"}
ALLOWED_OVERRIDE_KEYS = {"stack", "mode", "checks", "replaceDefaultChecks"}
ALLOWED_TOP_KEYS = {"defaults", "stacks", "moduleOverrides"}
ALLOWED_STACKS = {"node", "go", "rust", "java", "contract"}


def parse_args(argv: list[str]) -> argparse.Namespace:
    parser = argparse.ArgumentParser(description="Schema validator para validation-matrix.json")
    parser.add_argument("--matrix-path", required=True)
    return parser.parse_args(argv)


def validate(matrix: dict) -> tuple[list[str], list[str], int, int]:
    errors: list[str] = []
    warnings: list[str] = []

    # Top-level chaves desconhecidas geralmente são typos — flag como erro.
    for key in matrix:
        if key not in ALLOWED_TOP_KEYS:
            errors.append(
                f"root: chave desconhecida '{key}' "
                f"(esperado: {sorted(ALLOWED_TOP_KEYS)})"
            )

    defaults = matrix.get("defaults", {})
    if not isinstance(defaults, dict):
        errors.append("defaults: deve ser objeto")
    else:
        mode = defaults.get("mode")
        if mode is not None and mode not in ALLOWED_MODES:
            errors.append(
                f"defaults.mode: '{mode}' inválido "
                f"(use: {sorted(ALLOWED_MODES)})"
            )

    stacks = matrix.get("stacks", {})
    if not isinstance(stacks, dict):
        errors.append("stacks: deve ser objeto")
        stacks = {}
    else:
        for stack_name, stack_def in stacks.items():
            if stack_name not in ALLOWED_STACKS:
                warnings.append(
                    f"stacks.{stack_name}: stack fora do conjunto conhecido "
                    f"{sorted(ALLOWED_STACKS)} — confirmar intencional"
                )
            if not isinstance(stack_def, dict):
                errors.append(f"stacks.{stack_name}: deve ser objeto")
                continue
            checks = stack_def.get("checks", [])
            if not isinstance(checks, list):
                errors.append(f"stacks.{stack_name}.checks: deve ser lista")
                continue
            for idx, rule in enumerate(checks):
                _validate_check(f"stacks.{stack_name}.checks[{idx}]", rule, errors, warnings)

    overrides = matrix.get("moduleOverrides", {})
    if not isinstance(overrides, dict):
        errors.append("moduleOverrides: deve ser objeto")
        overrides = {}
    else:
        for mod_path, override in overrides.items():
            if not isinstance(override, dict):
                errors.append(f"moduleOverrides.{mod_path}: deve ser objeto")
                continue
            for key in override:
                if key not in ALLOWED_OVERRIDE_KEYS:
                    errors.append(
                        f"moduleOverrides.{mod_path}: chave desconhecida '{key}' "
                        f"(esperado: {sorted(ALLOWED_OVERRIDE_KEYS)})"
                    )
            mode = override.get("mode")
            if mode is not None and mode not in ALLOWED_MODES:
                errors.append(
                    f"moduleOverrides.{mod_path}.mode: '{mode}' inválido"
                )
            replace_defaults = override.get("replaceDefaultChecks")
            if replace_defaults is not None and not isinstance(replace_defaults, bool):
                errors.append(
                    f"moduleOverrides.{mod_path}.replaceDefaultChecks: deve ser boolean"
                )
            stack = override.get("stack")
            if stack is not None and stack not in ALLOWED_STACKS:
                warnings.append(
                    f"moduleOverrides.{mod_path}.stack: '{stack}' fora do conjunto conhecido"
                )
            checks = override.get("checks", [])
            if not isinstance(checks, list):
                errors.append(f"moduleOverrides.{mod_path}.checks: deve ser lista")
                continue
            for idx, rule in enumerate(checks):
                _validate_check(
                    f"moduleOverrides.{mod_path}.checks[{idx}]",
                    rule, errors, warnings,
                )

    return errors, warnings, len(stacks), len(overrides)


def _validate_check(prefix: str, rule, errors: list[str], warnings: list[str]) -> None:
    if not isinstance(rule, dict):
        errors.append(f"{prefix}: deve ser objeto")
        return

    for key in rule:
        if key not in ALLOWED_CHECK_KEYS:
            # Maior valor: pegar typos como `severty`, `cmd`, etc.
            errors.append(
                f"{prefix}: chave desconhecida '{key}' "
                f"(esperado: {sorted(ALLOWED_CHECK_KEYS)})"
            )

    command = rule.get("command")
    program = rule.get("program")
    args = rule.get("args")
    if (command is None) == (program is None):
        errors.append(f"{prefix}: exige exatamente um de command legado ou program estruturado")
    if command is not None and (not isinstance(command, str) or not command.strip()):
        errors.append(f"{prefix}.command: deve ser string não-vazia")
    if program is not None and (not isinstance(program, str) or not program.strip()):
        errors.append(f"{prefix}.program: deve ser string não-vazia")
    if args is not None and (not isinstance(args, list) or not all(isinstance(arg, str) for arg in args)):
        errors.append(f"{prefix}.args: deve ser lista de strings")
    if command is not None and args:
        errors.append(f"{prefix}.args: só é aceito com program estruturado")
    env = rule.get("env")
    if env is not None and (not isinstance(env, dict) or not all(isinstance(key, str) and isinstance(value, str) for key, value in env.items())):
        errors.append(f"{prefix}.env: deve ser objeto string:string")
    elif isinstance(env, dict):
        for key in env:
            if not key.strip() or "=" in key:
                errors.append(f"{prefix}.env: chave inválida: {key!r}")

    severity = rule.get("severity")
    if severity is None:
        # severity ausente = default 'required'; aceitar, mas avisar.
        warnings.append(
            f"{prefix}.severity: ausente — assumirá 'required' (default)"
        )
    elif severity not in ALLOWED_SEVERITIES:
        errors.append(
            f"{prefix}.severity: '{severity}' inválido "
            f"(use: {sorted(ALLOWED_SEVERITIES)})"
        )

    when = rule.get("when")
    if when is not None:
        if not isinstance(when, dict):
            errors.append(f"{prefix}.when: deve ser objeto")
        else:
            for key in when:
                if key not in ALLOWED_WHEN_KEYS:
                    errors.append(
                        f"{prefix}.when: chave desconhecida '{key}' "
                        f"(esperado: {sorted(ALLOWED_WHEN_KEYS)})"
                    )


def main(argv: list[str]) -> int:
    args = parse_args(argv)
    matrix_path = pathlib.Path(args.matrix_path)
    if not matrix_path.is_file():
        print(f"Erro: matriz ausente: {matrix_path}", file=sys.stderr)
        return 2

    try:
        matrix = json.loads(matrix_path.read_text(encoding="utf-8"))
    except (OSError, json.JSONDecodeError) as exc:
        print(f"Erro: falha ao parsear {matrix_path}: {exc}", file=sys.stderr)
        return 2

    if not isinstance(matrix, dict):
        print(f"Erro: raiz da matriz deve ser objeto JSON", file=sys.stderr)
        return 2

    errors, warnings, n_stacks, n_overrides = validate(matrix)

    for err in errors:
        print(f"[ERRO] {err}", file=sys.stderr)
    for warn in warnings:
        print(f"[AVISO] {warn}", file=sys.stderr)

    print(f"matrix.errors={len(errors)}")
    print(f"matrix.warnings={len(warnings)}")
    print(f"matrix.stacks={n_stacks}")
    print(f"matrix.module_overrides={n_overrides}")

    return 1 if errors else 0


if __name__ == "__main__":
    sys.exit(main(sys.argv[1:]))
