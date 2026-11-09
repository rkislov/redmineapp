#!/usr/bin/env python3
# Copyright 2026 Кислов Роман Сергеевич
# SPDX-License-Identifier: Apache-2.0
"""Collect the licenses of dependencies used by all supported desktop targets."""
import json
import os
import pathlib
import subprocess

root = pathlib.Path(__file__).resolve().parent.parent
modules = {}
for target in ("darwin", "linux", "windows"):
    env = dict(os.environ, GOOS=target, CGO_ENABLED="1")
    stream = subprocess.check_output(["go", "list", "-deps", "-json", "."], cwd=root, env=env, text=True)
    decoder = json.JSONDecoder()
    offset = 0
    while offset < len(stream):
        while offset < len(stream) and stream[offset].isspace():
            offset += 1
        if offset == len(stream):
            break
        obj, offset = decoder.raw_decode(stream, offset)
        module = obj.get("Module", {})
        if module.get("Dir") and not module.get("Main"):
            modules[module["Path"]] = module

parts = ["Third-party notices for redmineapp\n\nThe Apache-2.0 license of redmineapp does not replace the licenses below.\n"]
for name, module in sorted(modules.items()):
    directory = pathlib.Path(module["Dir"])
    licenses = []
    for path in directory.rglob("*"):
        if not path.is_file():
            continue
        rel = str(path.relative_to(directory)).lower()
        if path.name.upper().startswith(("LICENSE", "COPYING", "OFL")) and (path.parent == directory or "font" in rel or "theme" in rel):
            licenses.append(path)
    if not licenses:
        raise SystemExit("Missing license for " + name)
    parts.append("\n" + "=" * 78 + "\n" + name + " " + module.get("Version", "") + "\n")
    for path in sorted(licenses):
        parts.append("\n--- " + str(path.relative_to(directory)) + " ---\n" + path.read_text(errors="replace") + "\n")

parts.append("\n" + "=" * 78 + "\n" + (root / "packaging/pdfium-notices.txt").read_text())

goroot = pathlib.Path(subprocess.check_output(["go", "env", "GOROOT"], text=True).strip())
parts.append("\n" + "=" * 78 + "\nGo standard library and runtime\n\n" + (goroot / "LICENSE").read_text())
text = "\n".join(line.rstrip() for line in "".join(parts).splitlines()) + "\n"
(root / "THIRD_PARTY_NOTICES.txt").write_text(text)
print(f"Collected licenses for {len(modules)} modules and the Go runtime")
