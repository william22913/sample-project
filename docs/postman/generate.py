#!/usr/bin/env python3
"""Regenerate the Postman collection from the OpenAPI spec.

The collection is a BUILD ARTIFACT, not a hand-maintained file. Edit
docs/swagger/swagger.yaml and re-run this; collection.json is overwritten
wholesale. Do not hand-edit collection.json -- the next run discards it.

The spec is the source of truth, which is why this script exists rather than a
saved generation command: the converter emits no tests, so the baseline
assertions below are added after conversion, and a bare `openapi2postmanv2`
invocation would silently drop them on every regeneration.

Usage:
    python docs/postman/generate.py

Requires: openapi2postmanv2 on PATH  (npm i -g openapi-to-postmanv2)
"""
import json
import pathlib
import shutil
import subprocess
import sys

HERE = pathlib.Path(__file__).resolve().parent
REPO = HERE.parent.parent
SPEC = REPO / "docs" / "swagger" / "swagger.yaml"
OUT = HERE / "collection.json"

# folderStrategy=Tags groups requests into folders by each operation's `tags:`
# entry. It only works because every operation in the spec carries one; an
# untagged spec converts to a single flat folder regardless of this flag.
#
# parametersResolution=Example makes parameters and bodies take their values
# from the spec's `example:` fields rather than from the bare schema, so the
# generated requests carry a plausible uuid and a real filter string. Note the
# option name: this is the v2 interface. The v1 name
# (requestParametersResolution) is rejected as an invalid option by v6.x.
CONVERT_OPTIONS = "folderStrategy=Tags,parametersResolution=Example"

# Baseline assertion only: loose enough not to flake on a laptop, tight enough
# to catch a query that lost its index or a handler that started looping.
MAX_RESPONSE_MS = 2000

# Structural checks, applied at collection level so every request inherits them.
# Business-level assertions deliberately do NOT live here: they cannot be
# re-derived from the spec on regeneration, so they would rot silently. Anything
# that asserts a specific row, code or value belongs in a hand-written scenarios
# collection, kept separate from this generated baseline.
BASELINE_TESTS = [
    "pm.test('response is not a server error', () => "
    "pm.expect(pm.response.code).to.be.below(400));",
    f"pm.test('response time is under {MAX_RESPONSE_MS}ms', () => "
    f"pm.expect(pm.response.responseTime).to.be.below({MAX_RESPONSE_MS}));",
]


def convert(exe: str) -> None:
    subprocess.run(
        [
            exe,
            "-s", str(SPEC),
            "-o", str(OUT),
            "-p",
            "-O", CONVERT_OPTIONS,
        ],
        check=True,
    )


def normalize() -> None:
    """Strip the identifiers the converter assigns at random.

    openapi2postmanv2 stamps a fresh uuid on every item, on every saved
    response, and a _postman_id on the collection. They carry no information -
    Postman assigns its own on import, and Newman ignores them - so leaving them
    in made every regeneration a ~50-line diff that buried the contract changes
    the diff exists to surface.

    Only the structural ids are dropped, by position, not by a recursive sweep:
    an `id` inside a saved response body is a string of JSON and is left alone.

    The example VALUES are handled the other way round - fixed in the spec (see
    the note on components.parameters.TeacherID) so they are stable at the
    source, not scrubbed afterwards.
    """
    collection = json.loads(OUT.read_text(encoding="utf-8"))
    collection.get("info", {}).pop("_postman_id", None)
    collection.pop("_postman_id", None)
    for folder in collection.get("item", []):
        folder.pop("id", None)
        for request in folder.get("item", []):
            request.pop("id", None)
            for response in request.get("response", []):
                response.pop("id", None)
    OUT.write_text(json.dumps(collection, indent=2) + "\n", encoding="utf-8")


def enrich() -> None:
    collection = json.loads(OUT.read_text(encoding="utf-8"))
    collection["event"] = [
        {
            "listen": "test",
            "script": {"type": "text/javascript", "exec": BASELINE_TESTS},
        }
    ]
    OUT.write_text(json.dumps(collection, indent=2) + "\n", encoding="utf-8")


def report() -> None:
    collection = json.loads(OUT.read_text(encoding="utf-8"))
    folders = [i["name"] for i in collection["item"] if "item" in i]
    requests = sum(len(i["item"]) for i in collection["item"] if "item" in i)
    print(
        f"wrote {OUT.relative_to(REPO)}: "
        f"{len(folders)} folder(s) {folders}, {requests} request(s)"
    )


def main() -> int:
    if not SPEC.exists():
        print(f"spec not found: {SPEC}", file=sys.stderr)
        return 1

    # shutil.which rather than passing the bare name to subprocess: on Windows
    # the CLI installs as openapi2postmanv2.cmd, which subprocess does not
    # resolve (a shell would, which is why the command works by hand and this
    # script did not). which() honours PATHEXT and returns the real path.
    exe = shutil.which("openapi2postmanv2")
    if exe is None:
        print(
            "openapi2postmanv2 is not on PATH -- npm i -g openapi-to-postmanv2",
            file=sys.stderr,
        )
        return 1

    try:
        convert(exe)
    except subprocess.CalledProcessError as exc:
        print(f"conversion failed (exit {exc.returncode})", file=sys.stderr)
        return 1

    normalize()
    enrich()
    report()
    return 0


if __name__ == "__main__":
    raise SystemExit(main())
