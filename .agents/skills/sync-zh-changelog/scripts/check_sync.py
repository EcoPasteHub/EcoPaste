#!/usr/bin/env python3
"""Validate structural parity between English and Chinese release sections."""

from __future__ import annotations

import argparse
import re
import sys
from dataclasses import dataclass
from pathlib import Path


RELEASE_HEADING = re.compile(
    r"^## \[(?P<version>[^]]+)]\((?P<url>[^)]+)\) \((?P<date>[^)]+)\)$"
)
SECTION_HEADING = re.compile(r"^### (?P<title>.+)$")
LIST_ITEM = re.compile(r"^(?:[-*+] |\d+\. )")
MARKDOWN_LINK = re.compile(r"\[([^]]+)]\(([^)]+)\)")

HEADING_TRANSLATIONS = {
    "✨ Features": "✨ 新功能",
    "🐛 Bug Fixes": "🐛 问题修复",
    "⚡️ Performance": "⚡️ 性能优化",
    "⏪ Reverts": "⏪ 回退",
    "⚠️ Upgrade Notice": "⚠️ 升级说明",
}


@dataclass(frozen=True)
class Release:
    """Represent one parsed changelog release section."""

    version: str
    url: str
    date: str
    body: tuple[str, ...]


@dataclass(frozen=True)
class Section:
    """Represent a third-level heading and the content below it."""

    title: str
    content: tuple[str, ...]


def parse_releases(path: Path) -> list[Release]:
    """Parse release sections in source order from a changelog file."""
    if not path.is_file():
        raise ValueError(f"missing changelog: {path}")

    lines = path.read_text(encoding="utf-8").splitlines()
    releases: list[Release] = []
    current_match: re.Match[str] | None = None
    current_body: list[str] = []

    for line in lines:
        match = RELEASE_HEADING.match(line)
        if match:
            if current_match:
                releases.append(build_release(current_match, current_body))
            current_match = match
            current_body = []
            continue

        if current_match:
            current_body.append(line)

    if current_match:
        releases.append(build_release(current_match, current_body))

    if not releases:
        raise ValueError(f"no release sections found in {path}")

    return releases


def build_release(match: re.Match[str], body: list[str]) -> Release:
    """Build an immutable release from a heading match and body lines."""
    return Release(
        version=match.group("version"),
        url=match.group("url"),
        date=match.group("date"),
        body=tuple(body),
    )


def find_release(releases: list[Release], version: str) -> Release:
    """Return the unique release matching a version."""
    matches = [release for release in releases if release.version == version]
    if not matches:
        raise ValueError(f"release {version} is missing")
    if len(matches) > 1:
        raise ValueError(f"release {version} appears {len(matches)} times")

    return matches[0]


def parse_sections(release: Release) -> tuple[tuple[str, ...], list[Section]]:
    """Split release prose before the first heading from named sections."""
    preamble: list[str] = []
    sections: list[Section] = []
    current_title: str | None = None
    current_content: list[str] = []

    for line in release.body:
        match = SECTION_HEADING.match(line)
        if match:
            if current_title is not None:
                sections.append(Section(current_title, tuple(current_content)))
            current_title = match.group("title")
            current_content = []
            continue

        if current_title is None:
            preamble.append(line)
        else:
            current_content.append(line)

    if current_title is not None:
        sections.append(Section(current_title, tuple(current_content)))

    return tuple(preamble), sections


def content_units(
    lines: tuple[str, ...],
) -> list[tuple[str, tuple[tuple[str, str], ...]]]:
    """Return content kinds and Markdown links in source order."""
    units: list[tuple[str, tuple[tuple[str, str], ...]]] = []
    paragraph: list[str] = []

    def flush_paragraph() -> None:
        if not paragraph:
            return
        text = " ".join(part.strip() for part in paragraph)
        units.append(("paragraph", tuple(MARKDOWN_LINK.findall(text))))
        paragraph.clear()

    for line in lines:
        if not line.strip():
            flush_paragraph()
            continue

        if LIST_ITEM.match(line):
            flush_paragraph()
            units.append(("list", tuple(MARKDOWN_LINK.findall(line))))
            continue

        paragraph.append(line)

    flush_paragraph()
    return units


def compare_units(
    label: str, source: tuple[str, ...], target: tuple[str, ...]
) -> list[str]:
    """Compare content-unit kinds and Markdown links in source order."""
    source_units = content_units(source)
    target_units = content_units(target)
    errors: list[str] = []

    if len(source_units) != len(target_units):
        errors.append(
            f"{label}: content unit count differs "
            f"({len(source_units)} != {len(target_units)})"
        )
        return errors

    for index, (source_unit, target_unit) in enumerate(
        zip(source_units, target_units), start=1
    ):
        if source_unit[0] != target_unit[0]:
            errors.append(
                f"{label} item {index}: content kind differs "
                f"({source_unit[0]} != {target_unit[0]})"
            )
        if source_unit[1] != target_unit[1]:
            errors.append(f"{label} item {index}: Markdown links differ")

    return errors


def validate(source: Release, target: Release) -> list[str]:
    """Validate release metadata and translated structure."""
    errors: list[str] = []

    if source.url != target.url:
        errors.append(f"compare URL differs ({source.url} != {target.url})")
    if source.date != target.date:
        errors.append(f"release date differs ({source.date} != {target.date})")

    source_preamble, source_sections = parse_sections(source)
    target_preamble, target_sections = parse_sections(target)
    errors.extend(
        compare_units("release preamble", source_preamble, target_preamble)
    )

    if len(source_sections) != len(target_sections):
        errors.append(
            f"section count differs ({len(source_sections)} != {len(target_sections)})"
        )
        return errors

    for index, (source_section, target_section) in enumerate(
        zip(source_sections, target_sections), start=1
    ):
        expected_title = HEADING_TRANSLATIONS.get(source_section.title)
        if expected_title and target_section.title != expected_title:
            errors.append(
                f"section {index}: expected heading {expected_title!r}, "
                f"found {target_section.title!r}"
            )
        errors.extend(
            compare_units(
                f"section {index} ({source_section.title})",
                source_section.content,
                target_section.content,
            )
        )

    return errors


def parse_args() -> argparse.Namespace:
    """Parse command-line arguments."""
    parser = argparse.ArgumentParser(
        description="Validate EcoPaste English/Chinese changelog parity."
    )
    parser.add_argument(
        "--root",
        type=Path,
        default=Path.cwd(),
        help="repository root containing both changelog files",
    )
    parser.add_argument(
        "--version",
        help="release version to validate; defaults to the latest English release",
    )
    return parser.parse_args()


def main() -> int:
    """Run changelog validation and print a concise result."""
    args = parse_args()
    root = args.root.resolve()

    try:
        source_releases = parse_releases(root / "CHANGELOG.md")
        target_releases = parse_releases(root / "CHANGELOG.zh-CN.md")
        version = args.version or source_releases[0].version
        source = find_release(source_releases, version)
        target = find_release(target_releases, version)
        errors = validate(source, target)
        if args.version is None and target_releases[0].version != version:
            errors.insert(
                0, f"latest release {version} is not the first Chinese section"
            )
    except ValueError as error:
        print(f"error: {error}", file=sys.stderr)
        return 1

    if errors:
        for error in errors:
            print(f"error: {version}: {error}", file=sys.stderr)
        return 1

    print(f"CHANGELOG.zh-CN.md is synchronized for {version}.")
    return 0


if __name__ == "__main__":
    raise SystemExit(main())
