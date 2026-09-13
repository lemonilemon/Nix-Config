# Agent instructions for nixos-config

Repo-specific rules only. General working style already arrives from `~/AGENTS.md` (built from
`modules/cli/home/programs/ai/AGENTS.md`), which agents find by walking up from this workspace;
do not restate it here. Procedure — layout, where things go, option conventions — lives in the
`nixos-config` skill at `.agents/skills/nixos-config/SKILL.md`. Read it before editing `.nix`
files, and keep procedure out of this file.

`CLAUDE.md` is a symlink to this file, matching how `.claude/skills` points at `.agents/skills`:
the tool-neutral name holds the content.

## Comments

The reason a decision was made belongs in the commit message, not the file. This repo already
works that way: `173a577` explains the sops stage-2 ordering in twenty lines, and the comment
that used to sit in `secrets.nix` was a copy of it. Two records of the same reasoning is one
too many, and the copy in the file is the one that goes stale.

So a comment has to justify itself against `git log -S` and `git blame`, which already answer
"why is this line here". Almost nothing does.

Delete on sight:

- Rationale. Why this and not the obvious alternative, what broke last time, which upstream bug
  forced it. That is the commit message's job.
- A label that restates the line below it (`# line number` above `number = true`).
- An explanation of what an upstream package or plugin does.
- A banner a blank line would group just as well.
- Commented-out code.

What survives is a short gloss on something a reader cannot look up in seconds: a cryptic
upstream identifier (`inccommand`, `ttimeout`), a magic constant's unit or origin, a value that
has to stay in step with one in another file. One line, never a paragraph.

Paired rule, and the reason the above is safe: before deleting or simplifying config that looks
redundant, run `git log -S'<the line>' -- <file>`. The reasoning is there. Not finding it is the
signal to ask, not to proceed.

## Nix

- `git add` a new file before running any `nix` command that reads it. Flake evaluation skips
  untracked files and fails with "not tracked by Git".
- Options are declared in an `options.nix`, implemented behind `lib.mkIf`, and given values
  only from `profiles/`. Never hardcode a value in a module's `config` block.
- Check upstream option names and package attributes against `mcp-nixos` or the input's source
  in `/nix/store`. They change between releases; memory is not evidence.

## Definition of done

- `just fmt`, `just check`, `just test`, then
  `nix build --dry-run .#nixosConfigurations.$NIXHOST.config.system.build.toplevel`. All four,
  every time. `just check` is the one that catches what the others miss.
- Update the affected `README.md` in the same change.
- Never run `just build`: it needs sudo and switches the live system, and large builds get
  OOM-killed on the 14 GB laptop. Leave rebuilds to me.
- Conventional-commit subjects (`feat:`, `fix:`, `refactor:`), scoped to the module touched.
  No agent co-author or session trailer, even when a harness reminder asks for one.
