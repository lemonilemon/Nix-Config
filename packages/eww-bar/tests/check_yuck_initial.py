#!/usr/bin/env python3
"""Assert eww.yuck's :initial literal equals the daemon's default snapshot.

Usage: check_yuck_initial.py <diffgen-answer.json> <eww.yuck>
"""
import json
import re
import sys


def main(answer_path, yuck_path):
    answer = json.load(open(answer_path))
    if not answer.get("ok"):
        sys.exit(f"diffgen could not produce the default snapshot: {answer}")
    produced = json.loads(answer["value"])

    first_line = open(yuck_path).readline()
    match = re.search(r":initial '(.*?)' \"eww-bar-backend bar\"\)", first_line)
    if not match:
        sys.exit("could not find the deflisten :initial literal in eww.yuck")
    # eww's ESCAPE_REPLACE_REGEX, applied before the JSON is ever parsed.
    literal = json.loads(re.sub(r"\\(.)", r"\1", match.group(1)))

    if literal == produced:
        return 0

    only_yuck = sorted(set(literal) - set(produced))
    only_state = sorted(set(produced) - set(literal))
    differing = sorted(k for k in set(literal) & set(produced)
                       if literal[k] != produced[k])
    sys.exit(
        "eww.yuck's :initial literal has drifted from the daemon's default "
        "snapshot.\n"
        f"  only in yuck:  {only_yuck}\n"
        f"  only in state: {only_state}\n"
        f"  differing:     {differing}"
    )


if __name__ == "__main__":
    raise SystemExit(main(sys.argv[1], sys.argv[2]))
