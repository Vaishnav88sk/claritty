import re
import subprocess
import sys


SECTIONS = {
    "breaking": "⚠️ Breaking Changes",
    "feat": "🚀 Features",
    "fix": "🐛 Fixes",
    "tasks": "🧰 Tasks",
    "docs": "📝 Documentation",
}
CONVENTIONAL_SUBJECT = re.compile(r"([a-z]+)(?:\(([^)]+)\))?(!)?:\s+(.+)")
START = "<!-- claritty-changelog:start -->"
END = "<!-- claritty-changelog:end -->"


def main(tag, repository_url):
    tag_ref = f"refs/tags/{tag}"
    previous = subprocess.run(
        [
            "git", "describe", "--tags", "--first-parent", "--abbrev=0",
            "--match", "v*", "--exclude", tag, tag_ref,
        ],
        capture_output=True,
        text=True,
    ).stdout.strip()
    revision = f"refs/tags/{previous}..{tag_ref}" if previous else tag_ref
    commits = subprocess.check_output(
        ["git", "log", "--no-merges", "--format=%H%x1f%s%x1f%aN", revision, "--"],
        text=True,
    )

    entries = {section: [] for section in SECTIONS}
    contributors = set()
    for line in commits.splitlines():
        commit, subject, author = line.split("\x1f", 2)
        section = "tasks"
        match = CONVENTIONAL_SUBJECT.fullmatch(subject)
        if match:
            kind, scope, breaking, subject = match.groups()
            if breaking:
                section = "breaking"
            elif kind in SECTIONS:
                section = kind
            if scope:
                subject = f"**{scope}** — {subject}"
        entries[section].append(f"- [`{commit[:7]}`]({repository_url}/commit/{commit}) {subject}")
        contributors.add(author)

    existing = re.sub(f"{START}.*?{END}", "", sys.stdin.read(), flags=re.DOTALL).strip()
    if existing:
        print(existing, "", sep="\n")
    print(START, "## Changelog", "", sep="\n")
    for section, heading in SECTIONS.items():
        if entries[section]:
            print(f"### {heading}", "", *entries[section], "", sep="\n")
    if contributors:
        print("### Contributors", "", *(f"- {name}" for name in sorted(contributors)), "", sep="\n")
    if previous:
        print(f"[Full changelog for {previous} → {tag}]({repository_url}/compare/{previous}...{tag})", "", sep="\n")
    print(END)


if __name__ == "__main__":
    main(*sys.argv[1:])
