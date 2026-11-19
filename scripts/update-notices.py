#!/usr/bin/env python3
"""Preserve dependency and generator-template notices for binary distribution.

Uses only Go module provenance/checksums and license files, never system inventory.
Run after dependency/tool changes; --check verifies the committed artifact.
"""
import argparse
import json
import pathlib
import subprocess


def go(*args):
    return subprocess.check_output(["go", *args], text=True)


def objects(text):
    decoder = json.JSONDecoder()
    position = 0
    while position < len(text):
        while position < len(text) and text[position].isspace():
            position += 1
        if position == len(text):
            break
        value, position = decoder.raw_decode(text, position)
        yield value


def render():
    subprocess.run(["go", "mod", "download", "all"], check=True)
    modules = [module for module in objects(go("list", "-m", "-json", "all"))
               if not module.get("Main")]
    # Generated CLI templates contribute source even without a runtime module.
    modules.append(json.loads(go("mod", "download", "-json",
                                 "github.com/arran4/go-subcommand@v0.0.32")))
    output = ["# Third-party notices\n\n"
              "License texts from the Go module graph (including build/test modules), "
              "the pinned CLI generator templates and the Go standard library. "
              "Some listed modules are not linked into the binary. Module versions "
              "and integrity are recorded by go.mod/go.sum and the pinned generation "
              "directive. Regenerate with scripts/update-notices.py after updates.\n\n"]
    sources = [("Go standard library", pathlib.Path(go("env", "GOROOT").strip()), False)]
    sources += [(module["Path"], pathlib.Path(module["Dir"]), True)
                for module in sorted(modules, key=lambda item: item["Path"])]
    for name, directory, recursive in sources:
        candidates = directory.rglob("*") if recursive else [directory / "LICENSE"]
        licenses = sorted(path for path in candidates if path.is_file() and
                          path.name.upper().startswith(
                              ("LICENSE", "LICENCE", "COPYING", "NOTICE", "COPYRIGHT")))
        if not licenses:
            raise RuntimeError(f"No license/notice files found for {name}; manual provenance review required")
        output.append(f"## {name}\n\n")
        for path in licenses:
            text = path.read_text(encoding="utf-8")
            output.append(f"### {path.relative_to(directory).as_posix()}\n\n```text\n{text.rstrip()}\n```\n\n")
    return "".join(output)


def main():
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument("--check", action="store_true")
    args = parser.parse_args()
    target = pathlib.Path("THIRD_PARTY_NOTICES.md")
    rendered = render()
    if args.check:
        if not target.exists() or target.read_text(encoding="utf-8") != rendered:
            raise SystemExit("THIRD_PARTY_NOTICES.md needs regeneration and review")
    else:
        target.write_text(rendered, encoding="utf-8")


if __name__ == "__main__":
    main()
