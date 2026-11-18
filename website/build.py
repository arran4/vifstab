"""Build a narrowly selected documentation site; never copy the repository root."""
import argparse
import html.parser
import json
import posixpath
import re
import shutil
import subprocess
import tempfile
from pathlib import Path
from urllib.parse import quote, unquote, urljoin, urlsplit

ROOT = Path(__file__).resolve().parents[1]
SITE = ROOT / "website"
PAGES = json.loads((SITE / "pages.json").read_text())
FILENAME = {entry[1]: entry[0] for entry in PAGES}
FILENAME["README.md"] = ""
DEFAULT_BASE = "https://arran4.github.io/vifstab/"
GITHUB = "https://github.com/arran4/vifstab/blob/main/"
LINK = re.compile(r"(?<!!)\[([^\]\n]+)\]\(([^)\s]+)\)")
FENCE = re.compile(r"^\s*(\x60{3,}|~{3,})")


def markdown(source, base):
    """Rewrite local Markdown links, validating actual repository targets."""
    body = (ROOT / source).read_text()
    lines = body.splitlines(keepends=True)
    if lines and lines[0].startswith("# "):
        lines = lines[1:]
    fenced = None
    result = []
    for line in lines:
        match = FENCE.match(line)
        if match:
            marker = match.group(1)[0]
            if fenced is None:
                fenced = marker
            elif fenced == marker:
                fenced = None
            result.append(line)
            continue
        if fenced:
            result.append(line)
            continue

        def rewrite(match):
            label, href = match.groups()
            parsed = urlsplit(href)
            if parsed.scheme or parsed.netloc or not parsed.path:
                return match.group(0)
            target = posixpath.normpath(posixpath.join(
                posixpath.dirname(source), unquote(parsed.path)))
            if target.startswith("../") or not (ROOT / target).is_file():
                raise ValueError(f"{source}: incompatible relative link {href}")
            if target in FILENAME:
                route = FILENAME[target]
                dest = urljoin(base, route + "/" if route else "")
            else:
                dest = GITHUB + quote(target)
            if parsed.fragment:
                dest += "#" + parsed.fragment
            return f"[{label}]({dest})"

        result.append(LINK.sub(rewrite, line))
    return "".join(result).lstrip()


class PageParser(html.parser.HTMLParser):
    def __init__(self):
        super().__init__()
        self.ids = set()
        self.links = []

    def handle_starttag(self, tag, attrs):
        attrs = dict(attrs)
        if "id" in attrs:
            self.ids.add(attrs["id"])
        for attribute in ("href", "src"):
            if attribute in attrs:
                self.links.append(attrs[attribute])


def validate(output, base):
    """Check every generated local link, anchor and output path."""
    allowed = {"index.html", "404.html", "sitemap.xml", "robots.txt", "css/site.css"}
    allowed.update(f"{p[0]}/index.html" for p in PAGES if p[0])
    actual = {p.relative_to(output).as_posix() for p in output.rglob("*") if p.is_file()}
    unexpected = actual - allowed
    required = allowed - {"robots.txt"}
    if unexpected or required - actual:
        raise ValueError(f"output inventory: unexpected={unexpected}, missing={required-actual}")
    parsed = {}
    for path in output.rglob("*.html"):
        parser = PageParser()
        parser.feed(path.read_text())
        parsed[path.relative_to(output).as_posix()] = parser
    origin = urlsplit(base)
    for name, parser in parsed.items():
        current = urljoin(base, name)
        for href in parser.links:
            target = urlsplit(urljoin(current, href))
            if (target.scheme, target.netloc) != (origin.scheme, origin.netloc):
                continue
            if not target.path.startswith(origin.path):
                raise ValueError(f"{name}: link outside project base: {href}")
            rel = unquote(target.path[len(origin.path):])
            if not rel or rel.endswith("/"):
                rel += "index.html"
            if rel not in actual:
                raise ValueError(f"{name}: broken link {href}")
            if target.fragment and rel in parsed and unquote(target.fragment) not in parsed[rel].ids:
                raise ValueError(f"{name}: missing anchor {href}")
    print(f"Verified {len(parsed)} HTML pages and {len(actual)} output files")


def main():
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument("--base-url", default=DEFAULT_BASE)
    parser.add_argument("--output", type=Path, required=True)
    args = parser.parse_args()
    base = args.base_url.rstrip("/") + "/"
    if urlsplit(base).scheme not in ("http", "https") or not urlsplit(base).netloc:
        parser.error("base URL must be absolute HTTP(S)")
    output = args.output.resolve()
    # Refuse to overwrite existing data; CI uses a fresh output directory.
    if output.exists() and (not output.is_dir() or any(output.iterdir())):
        parser.error("output must be a new or empty directory")
    with tempfile.TemporaryDirectory(prefix="vifstab-site-") as temp:
        stage = Path(temp)
        for name in ("layouts", "static"):
            shutil.copytree(SITE / name, stage / name)
        shutil.copy2(SITE / "hugo.toml", stage / "hugo.toml")
        (stage / "data").mkdir()
        (stage / "data" / "navigation.json").write_text(json.dumps(
            [{"path": route + "/" if route else "", "title": title}
             for route, _, title, _ in PAGES]))
        content = stage / "content"
        content.mkdir()
        for route, source, title, description in PAGES:
            filename = (content / f"{route}.md") if route else content / "_index.md"
            front = {"title": title, "description": description}
            filename.write_text(json.dumps(front) + "\n" + markdown(source, base))
        subprocess.run(["hugo", "--source", str(stage), "--destination", str(output),
                        "--baseURL", base, "--minify"], check=True)
        validate(output, base)


if __name__ == "__main__":
    main()
