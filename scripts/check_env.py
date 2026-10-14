#!/usr/bin/env python3
"""M0 environment check.

What it does: reports whether the tools Spec-to-Service needs are present.
Why it exists: fail early (and clearly) if Go is missing, before any code is written.
What it does not do: install anything, or print API key values (only set/unset).
Exit code is 1 only if a REQUIRED item (Python 3.10+, Go 1.22+) is missing.
"""
import os
import re
import shutil
import subprocess
import sys

KEYS = ["NVIDIA_API_KEY", "GEMINI_API_KEY", "ANTHROPIC_API_KEY"]
OPTIONAL_TOOLS = ["docker", "hadolint", "kubeconform"]


def run(cmd):
    try:
        return subprocess.run(cmd, capture_output=True, text=True, timeout=15).stdout.strip()
    except Exception:
        return ""


def main():
    ok = True

    v = sys.version_info
    py_ok = v >= (3, 10)
    print(f"[{'ok' if py_ok else 'FAIL'}] python {v.major}.{v.minor}.{v.micro} (need 3.10+)")
    ok &= py_ok

    if shutil.which("go"):
        out = run(["go", "version"])
        m = re.search(r"go(\d+)\.(\d+)", out)
        go_ok = bool(m) and (int(m.group(1)), int(m.group(2))) >= (1, 22)
        print(f"[{'ok' if go_ok else 'FAIL'}] {out} (need 1.22+)")
        ok &= go_ok
    else:
        go_ok = False
        ok = False
        print("[FAIL] go not found (REQUIRED)")
        print("       Install: `brew install go` (macOS), or https://go.dev/dl/ ; then re-run.")

    for t in OPTIONAL_TOOLS:
        print(f"[{'ok' if shutil.which(t) else 'skip'}] {t} (optional; checks marked 'skipped' if missing)")

    for k in KEYS:
        print(f"[{'set' if os.environ.get(k) else 'unset'}] {k}")

    for mod in ["openai", "yaml", "jinja2", "pytest", "rich"]:
        try:
            __import__(mod)
            print(f"[ok] python module {mod}")
        except ImportError:
            print(f"[missing] python module {mod} (pip install)")

    sys.exit(0 if ok else 1)


if __name__ == "__main__":
    main()
