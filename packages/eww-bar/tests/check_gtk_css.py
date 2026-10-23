#!/usr/bin/env python3
"""Assert every pseudo-class in eww.scss is one GTK actually implements.

Usage: check_gtk_css.py <eww.scss>
"""
import re
import sys

# From GTK 3's CSS documentation. libgtk's symbol table is not evidence.
KNOWN = {
    # state
    "active", "hover", "selected", "disabled", "indeterminate",
    "checked", "focus", "focus-visible", "focus-within", "backdrop",
    "link", "visited",
    # structural
    "first-child", "last-child", "only-child", "nth-child", "nth-last-child",
    "not", "dir", "drop",
}


def selectors(css):
    """Yield the selector text preceding each rule block.

    Handles nesting, which this stylesheet has: @media blocks contain rules.
    A selector runs from the last structural character -- '{', '}' or ';' --
    up to the '{' that opens its block.
    """
    css = re.sub(r"/\*.*?\*/", "", css, flags=re.S)   # block comments
    css = re.sub(r"//[^\n]*", "", css)                 # line comments

    start = 0
    for i, ch in enumerate(css):
        if ch in "{};":
            if ch == "{":
                yield css[start:i]
            start = i + 1


def main(path):
    css = open(path, encoding="utf-8").read()
    bad = []

    for selector in selectors(css):
        selector = selector.strip()
        # @media, @mixin, @keyframes, @include: at-rules carry colons that are
        # not pseudo-classes -- (prefers-color-scheme: dark) above all.
        if not selector or selector.startswith("@"):
            continue
        for name in re.findall(r"(?<!:):([a-zA-Z-]+)", selector):
            if name not in KNOWN:
                bad.append((name, " ".join(selector.split())))

    if not bad:
        return 0

    print("eww.scss uses pseudo-classes GTK does not implement.", file=sys.stderr)
    print("GTK abandons the rest of the stylesheet at the first one, so this "
          "would strip the bar's appearance from that line down.\n", file=sys.stderr)
    for name, selector in bad:
        print(f"  :{name}\tin  {selector}", file=sys.stderr)
    print(f"\nKnown: {', '.join(':' + k for k in sorted(KNOWN))}", file=sys.stderr)
    return 1


if __name__ == "__main__":
    raise SystemExit(main(sys.argv[1]))
