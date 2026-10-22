#!/usr/bin/env python3
"""Assert every pseudo-class in eww.scss is one GTK actually implements.

This exists because of a specific outage. `calendar:week` was added to the
stylesheet on the strength of the string appearing in libgtk's symbol table.
It is not a pseudo-class: :week and :day were GTK3 *regions*, deprecated in
3.14 and since removed. GTK does not skip a rule it cannot parse -- it
abandons the rest of the stylesheet -- so one bad selector discarded every
rule after it and the whole bar lost its appearance.

Nothing caught it. dart-sass compiles `calendar:week` happily, because it is
valid CSS syntax; only GTK knows which pseudo-classes exist. The first place
the error could surface was the running bar, after a rebuild.

Running the stylesheet through GTK's own parser would be the faithful check,
and it is not worth what it costs: it needs a GTK typelib environment inside
the sandbox, and nixpkgs' pygobject wrapper fights over GI_TYPELIB_PATH. An
allowlist catches the entire class of bug for twenty lines and no dependency.

Usage: check_gtk_css.py <eww.scss>
"""
import re
import sys

# GTK 3 CSS pseudo-classes. From the GTK CSS documentation, not from grepping
# the library -- that is the mistake this script exists to prevent.
#
# Add to this list only with evidence from GTK's documentation, and say where
# the evidence came from in the commit message.
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
