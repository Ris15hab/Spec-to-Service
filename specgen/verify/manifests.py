"""Check Dockerfile and k8s manifests.

Always: parse the YAML and check the fields we rely on (kind, name, probes on /health).
Optionally: hadolint, kubeconform. A missing tool is reported as `skipped_tools`, never
as a pass for that tool.
Why: manifests come from templates so failures here mean a tool bug, not model error.
Deliberately does not: `docker build` or deploy anything.
"""
import shutil
import time
from pathlib import Path

import yaml

from specgen.verify.common import failed, passed, run_cmd


def check_manifests(project):
    t0 = time.monotonic()
    project = Path(project)
    problems, skipped = [], []
    kinds = set()
    for f in sorted((project / "k8s").glob("*.yaml")):
        try:
            for doc in yaml.safe_load_all(f.read_text()):
                kinds.add(doc.get("kind"))
                if not doc.get("apiVersion") or not doc.get("metadata", {}).get("name"):
                    problems.append(f"{f.name}: missing apiVersion or metadata.name")
                if doc.get("kind") == "Deployment":
                    c = doc["spec"]["template"]["spec"]["containers"][0]
                    for probe in ("livenessProbe", "readinessProbe"):
                        if c.get(probe, {}).get("httpGet", {}).get("path") != "/health":
                            problems.append(f"{f.name}: {probe} must hit /health")
        except Exception as e:  # malformed YAML or unexpected shape
            problems.append(f"{f.name}: {e}")
    if not {"Deployment", "Service"} <= kinds:
        problems.append(f"expected Deployment and Service, found {sorted(k for k in kinds if k)}")
    if not (project / "Dockerfile").exists():
        problems.append("Dockerfile missing")
    if shutil.which("hadolint"):
        rc, out, _, _ = run_cmd(["hadolint", "--failure-threshold", "warning", "Dockerfile"], project, 30)
        if rc != 0:
            problems.append("hadolint:\n" + out)
    else:
        skipped.append("hadolint")
    if shutil.which("kubeconform"):
        rc, out, _, _ = run_cmd(["kubeconform", "-strict", "k8s"], project, 30)
        if rc != 0:
            problems.append("kubeconform:\n" + out)
    else:
        skipped.append("kubeconform")
    dur = time.monotonic() - t0
    if problems:
        return failed("manifests", dur, "manifest_error", "\n".join(problems), skipped_tools=skipped)
    return passed("manifests", dur, skipped_tools=skipped)
