"""Parse an LLM answer into {filename: content} from delimited blocks.

Format:  === FILE: handlers.go ===  ...  === END FILE ===
Why: delimiters survive code that contains markdown fences or backticks, unlike
JSON-wrapped code. Tolerates a surrounding ``` fence per file, stray prose
between blocks, and a missing END marker (block then ends at the next FILE marker).
Deliberately does not: guess files from bare code fences. No blocks = parse_error.
"""
import re

TEMPLATE_OWNED = {"main.go", "metrics.go", "go.mod", "go.sum", "Dockerfile", "Makefile"}
_START = re.compile(r"^=== FILE: *(.+?) *===\s*$", re.M)
_END = re.compile(r"^=== END FILE ===\s*$", re.M)


class ParseError(ValueError):
    pass


def _strip_fence(body: str) -> str:
    lines = body.strip("\n").split("\n")
    if len(lines) >= 2 and lines[0].lstrip().startswith("```") and lines[-1].strip() == "```":
        lines = lines[1:-1]
    return "\n".join(lines) + "\n"


def _valid_name(name: str) -> bool:
    return bool(re.fullmatch(r"[A-Za-z0-9_.\-]+", name)) and (name.endswith(".go") or name == "README.md")


def parse_files(text: str) -> dict:
    starts = list(_START.finditer(text))
    if not starts:
        raise ParseError("no '=== FILE: name ===' blocks found")
    files = {}
    for i, m in enumerate(starts):
        name = m.group(1).strip().strip("`")
        limit = starts[i + 1].start() if i + 1 < len(starts) else len(text)
        chunk = text[m.end():limit]
        end = _END.search(chunk)
        body = chunk[:end.start()] if end else chunk
        if name in TEMPLATE_OWNED:
            raise ParseError(f"file '{name}' is owned by the template and must not be written")
        if not _valid_name(name):
            raise ParseError(f"unexpected file name '{name}' (only flat *.go files and README.md)")
        files[name] = _strip_fence(body)
    return files


def check_required(files: dict) -> None:
    """A complete first answer needs handlers, at least one test file, and README."""
    missing = []
    if "handlers.go" not in files:
        missing.append("handlers.go")
    if not any(n.endswith("_test.go") for n in files):
        missing.append("a *_test.go file")
    if "README.md" not in files:
        missing.append("README.md")
    if missing:
        raise ParseError("missing required files: " + ", ".join(missing))
