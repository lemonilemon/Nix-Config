# Eww bar backend: remediate in Python, then port to Go

Fix the backend's live defects and shrink its surface in Python, then port the
remainder to Go for a statically-typed, compiled daemon with no interpreter
startup and no dependency file that can go stale.

Ships as **two independent changesets**. Changeset 1 stands on its own and is
worth landing whether or not Changeset 2 ever happens. Changeset 2 is gated on
Changeset 1 and on a re-evaluation checkpoint, because Changeset 1 removes most
of the evidence that currently motivates a rewrite.

## Problem

`modules/desktop/home/hyprland/eww/scripts/eww_bar_backend/` is 3231 lines of
Python across 12 modules, with 1304 lines of tests, and it is growing: 39 commits
touched `scripts/` between 2026-07-04 and 2026-07-25. The concern driving this
work is that a port gets more expensive as the code grows, and that a nested-dict
state model with no schema anywhere in the pipeline degrades silently when it
drifts.

Both halves of that concern are correct. But investigation found that the
codebase also carries defects and dead weight that inflate the port and that
nothing about the language choice fixes.

## Research findings

Measured on this host against the live daemon, not projected.

### There is no performance problem at the daemon layer

| Metric | Value |
|---|---|
| Daemon RSS | 33 MB (the eww daemon it feeds is 108 MB) |
| CPU | 2.57 s over 843 s elapsed = 0.30% of one core |
| Thread states | All 16 blocked in kernel: 7 `hrtimer_nanosleep`, 4 `anon_pipe_read`, 1 `poll`, 1 `unix_stream_read`, 1 `skb_wait`, 2 `futex` |
| Journal (14 days) | 1191 lines, zero Python tracebacks |

The only user-visible defect is `eww-barctl` latency, invoked from 34 call sites
in `eww.yuck`, including `:onscroll` at `eww.yuck:93`.

### Most of that latency is an import-graph bug, not a language cost

`scripts/backend:10` does `from eww_bar_backend import *`, so the CLI client
imports the entire daemon, including `collectors` → `urllib.request` →
`http.client` → `email.parser`.

| Client | Median ping round-trip |
|---|---|
| Shipped `eww-barctl` | 76.35 ms |
| After import-graph split (prototyped, 110 tests green) | 24.19 ms |
| Compiled client (Go, measured against the same live daemon) | 1.10 ms |
| Compiled client (Rust, measured) | 0.72 ms |

The Python split recovers 68% of it. The residual 24 ms is CPython's ~13.6 ms
interpreter floor plus `json`/`site`. So a rewrite buys **23 ms per click** over
a change that takes an afternoon.

### The port surface is ~600 lines larger than assumed

`collectors.py:168-866` is the AI-usage subsystem: **699 lines**, 47% of
`collectors.py` and 23% of the package. An earlier estimate of "~106 lines" was a
grep of references, not the subsystem.

It is also the worst possible candidate for static typing. Its job is defensively
parsing six independently-versioned third-party schemas (ccusage, openusage-cli,
dunstctl, `hyprctl -j`, `pactl -f json`, api.anthropic.com), probing up to 8 key
spellings per field. Both static languages are *measurably worse* than Python
here: serde's `alias` hard-errors when two spellings are present at once, where
`collectors.py:168-180` returns the first match; `deny_unknown_fields` is unusable
against upstreams that will gain fields; `Vec<T>` fails a whole list on one bad
element where `collectors.py:44-55` filters per-element and `test_watchers.py:78-81`
explicitly asserts graceful degradation. Go's answer is `map[string]any`, which is
what Python already does.

This subsystem is also the only HTTP, the only credential handling, and the entire
cause of the import-graph latency.

### Go over Rust, for this code

Rust's advantage is real and narrow: partial struct literals are `E0063`, closed-set
enums are `E0308`, and `Vec<T>` has no nil/empty distinction. Both of this repo's
confirmed live defects are that species, and Go catches neither (`BtClass(powered)`
compiles; `Battery{Text:"x"}` marshals seven empty fields).

Go wins everywhere else that matters here:

| Dimension | Go | Rust |
|---|---|---|
| Third-party deps | Zero achievable; every import maps to stdlib | 11 crates floor (serde), +26 for the one HTTPS GET |
| Nix hash upkeep | `vendorHash = null` creates *no fetch derivation* (`module.nix:78-82`) | `cargoLock.lockFile` bypasses `cargoHash`; flat but non-zero |
| Derivation rebuild | 9.7-10.6 s | 18.2-21.1 s (Python baseline 3.5-4.4 s) |
| Test injection | ~15 `func`→`var` edits, no call-site churn | ~15-method trait through ~40 functions |
| Concurrency | 16 kernel-blocked threads transcribe to goroutines 1:1 | Forces a tokio-vs-blocking-API decision |
| Native audio | `jfreymuth/pulse` verified: v32 negotiated, sink enumeration, 21 events, volume writes | `pulseaudio` 0.3.1 has no `set_sink_volume`/`set_sink_mute`/`subscribe` |
| Native D-Bus | godbus 67 lines | zbus 71 lines (near-exact tie; neither links libdbus) |

Go's cost is four silent-failure modes, all opt-out, all one-line fixes once known:
byte-slicing `truncate_text` across 13 Nerd-Font call sites, `math.Round` vs
Python's banker's rounding, `encoding/json` HTML-escaping and map-key sorting, and
`reflect.DeepEqual(nil, [])` flipping `"sinks":null` ↔ `[]` under nine live
`(for ...)` loops.

Note both toolchains are already in the system closure, and dropping `python3`
from `runtimePackages` frees **zero bytes** (blueman, kitty, pipewire, wireplumber,
hyprland and pulsemixer all pull the same store path).

### Confirmed live defects

Each verified directly against the shipped code, not inferred.

| # | Defect | Location |
|---|---|---|
| 1 | `parse_controller` returns bluetoothctl's raw `Powered:` value, so `class` is `"yes"`/`"no"`. Executing the shipped function gives `class='no'` when powered off, so `eww.scss:313`'s `.bluetooth.off` is unreachable. | `collectors.py:1400-1414`, `:1449` |
| 2 | `eww.yuck:1`'s `:initial` literal has drifted from the Python defaults: `"battery":{"text":""` vs `common.py:9`'s `"󰂄"`, `"temperature":{"text":""` vs `state.py:31`'s `" --°C"`. Hand-synced, 18 keys against `state.py`'s 17. | `eww.yuck:1` |
| 3 | `idle_inhibited_state` is defined twice; the `collectors.py` copy is shadowed and unreachable (all three importers use `.inhibitors`). `common.py:129 idle_pidfile_path` exists only to feed it. | `collectors.py:1495-1501` |
| 4 | `scripts/listen`, `scripts/poll`, `scripts/hypr-events` have zero references in any `.nix` or `.yuck`. `default.nix:137-138` ships all of `./scripts` to `~/.config` on every generation. | `scripts/` |
| 5 | The CLI client imports the whole daemon. | `scripts/backend:10` |

## Design

### Changeset 1 — Python remediation

Lands as separate commits, each with its test. No language change. Independently
revertable.

1. **Fix the bluetooth class leak.** Map the raw `Powered:` value to the closed set
   the stylesheet expects. Test asserts `class == "off"` when powered off and that
   `.bluetooth.off` becomes reachable.
2. **Generate `eww.yuck:1`'s `:initial` from the Python defaults.** Add a
   `--print-default-state` mode to the backend, and wire it through the existing
   `pkgs.replaceVars` call at `default.nix:73` so the literal cannot drift again.
   This kills a whole class of bug rather than one instance.
3. **Split the control client out of the import graph.** New
   `eww_bar_backend/ctl.py` importing only `json` and `socket`; `scripts/backend`
   dispatches on `argv[0]` *before* importing anything heavy; empty the 97-line
   re-export block in `__init__.py`, whose first line (`from .app import main,
   run_bar`) is what pulls the whole daemon in. Verified safe: its only consumer is
   the shim's star import, and every test imports submodules directly
   (`from eww_bar_backend import collectors`, …).
4. **Delete dead code.** `collectors.py:1495-1501`, `common.py:129`, the three
   unused shell scripts, and narrow `default.nix:137-138` so `~/.config/eww/scripts`
   stops shipping them.
5. **Extract the AI-usage subsystem** (`collectors.py:168-866`) into a separate
   binary, not merely a separate module.

   The daemon currently owns this work via `periodic_refresh(state, 300,
   refresh_ai_usage)` at `app.py:70-74`, which is why it holds the only HTTP and the
   only credential handling. Extracting it to a third `argv[0]` target that prints
   the AI-usage JSON on stdout turns `refresh_ai_usage` into a `run_text(...)` plus
   parse, exactly the shape the daemon already uses for its other 34 runtime
   binaries — and it already shells out to `openusage-cli` for the sibling quotas at
   `collectors.py:442-447`, so this is consistent rather than novel.

   This is what makes (3) durable: the import-graph split only stays fixed if
   nothing pulls `urllib` back into the daemon's module graph.

After (5), `collectors.py` drops from 1501 to ~800 lines; after (3), (4) and (5)
the package drops from 3231 to roughly 2400-2500.

### Changeset 2 — Go port, gated

**Checkpoint first.** Changeset 1 removes the only user-visible defect, the only
confirmed live bug, and the drift mechanism. Re-evaluate before starting: if the
`:initial` generation stops the recurring drift and the 24 ms click latency is
imperceptible in use, the remaining case is 23 ms and a type system.

If it proceeds, port in this order, each stage keeping the daemon shippable:

1. `common`, `state`, `watchers`, `app` — the concurrency core, where Go gains most.
2. `control` and `popups` — takes the click path to ~1 ms.
3. The pure parsing functions (~66-69% of the 110 tests are pure input→output and
   transliterate to Go table tests directly).
4. The subprocess-calling collectors, using `var runText = func(...)` so the ~30
   `mock.patch` tests retrofit without call-site churn.
5. The extracted AI subsystem last, or never — it can stay Python behind its own
   binary, since `map[string]any` gains nothing over dicts.

**Equivalence gate.** The port must emit byte-compatible JSON. Run both daemons
side by side and diff emitted state. Known hazards to pin with tests before
porting each: `truncate_text` code-point vs byte slicing (`common.py:118`),
`Decimal`/`ROUND_HALF_UP` and `round()` banker's rounding (`collectors.py:936`,
`:230`, `:1195`, `:1259`), `json.dumps` separators and `ensure_ascii=False`
(`state.py:64`), and nil-vs-empty slices.

**Packaging.** `ewwBarTools` (`default.nix:17`) becomes `buildGoModule` with
`vendorHash = null`; the two `cp` at `default.nix:27-28` become `ln -s`;
`patchShebangs` and `nativeBuildInputs = [ pkgs.python3 ]` drop. Tests move into
the derivation, collapsing `flake.nix:134-166` (a 33-line `runCommand` including a
15-line guard that exists only because `unittest discover` silently skips
packageless directories). `go` needs adding to `modules/general/home/default.nix`;
`gopls` is already enabled at `lsp.nix:49-51`.

Lifting the package out of the home-manager module so `checks` can reference it
requires reconciling the two non-overlapping overlay lists (`nixpkgs/overlays.nix`
vs `overlays/default.nix`). That cost is language-neutral and unavoidable.

## Out of scope

- **Native protocol migration** (D-Bus/PulseAudio replacing `bluetoothctl`, `nmcli`,
  `pactl`, `dunstctl`). The replaceable surface is ~717 of 3231 lines. The code
  already has the seam for it: every collector splits into `X_state_from_text(text)`
  plus `X_state()`. So it can be done subsystem by subsystem, later, in whichever
  language the daemon is in by then. Doing it *with* a rewrite takes both risks at once.
- **Rust.** Decided against above.
- **Rewriting the AI subsystem's parsing to be strict.** Its liberality is correct.

## Risks

The port carries knowledge that is not visible in the code. 13 of the 39 commits to
`scripts/` are fixes (+299/-63), 9.3% of the package existing because something broke
against this specific desktop: `6dec501` (dunst stamps `CLOCK_BOOTTIME`, not
`MONOTONIC` — needs a suspend cycle to observe), `2846bb6` (hotplug events race eww's
reload, `watchers.py:110-127`), `watchers.py:58-66` (Hyprland emits *both* v1 and v2
monitor events, so matching only the v1 prefix keeps it single-fire), `watchers.py:77-79`
(GDK lags Hyprland on hotplug, so retry), `a287348` (distinguish a dead NetworkManager
from being offline).

There are 100 comment lines in 3231 and they are almost entirely *why*, not *what*.
The ones most likely to be "improved away" during a port are exactly the ugly-looking
ones. Mitigation: port comments verbatim with their code, and treat any comment
explaining a workaround as a required test case before the corresponding code moves.

Secondary risk: ~127 semantic decision points (truncation, rounding, zero values,
nil-vs-empty, JSON escaping and key ordering, `strftime` layouts, int/float coercion)
are each correct today and a coin flip after the port. Standard defect-density figures
put 3-80 latent defects on 3231 lines. Most would surface over weeks, since they need
a specific hardware, network, or suspend state.

## Verification

Definition of done for each changeset: `just fmt`, `just check`, `just test`, then
`nix build --dry-run .#nixosConfigurations.<host>.config.system.build.toplevel`.
All four pass before presenting.

Changeset 1 additionally: all 110 existing tests pass unmodified except where a test
encodes the bluetooth bug, plus a measured before/after on `eww-barctl` latency.
