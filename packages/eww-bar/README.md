# eww-bar

A status bar for Hyprland built on [eww](https://github.com/elkowar/eww): workspaces,
window title, media, notifications, network, Bluetooth, AI usage, temperature, CPU,
memory, storage, volume and battery, each with its own popup, plus a Settings popup
that shows or hides modules while the bar runs. The system tray sits inline up to six
visible icons; past that it folds to `⋯`, which opens the whole tray in a popup that
grows with it. A Go backend collects the state and
streams it to eww; `eww-barctl` and `eww-popup` are the commands the widgets call.

Everything it needs is in this directory. The rest of the repository uses it through
`modules/desktop/home/hyprland/eww/default.nix`, which only maps the repository's
own options onto the ones below.

## Use

The flake exports it as `homeManagerModules.eww-bar`:

```nix
home-manager.users.me = {
  imports = [ inputs.nixos-config.homeManagerModules.eww-bar ];
  programs.eww-bar = {
    enable = true;
    modules = [ "window" "notifications" "network" "cpu" "memory" "volume" "battery" ];
    battery.enable = true;
  };
};
```

It assumes Hyprland with Home Manager's systemd integration: the bar's service is
`WantedBy` and `PartOf` `hyprland-session.target`, and the backend talks to
Hyprland through `hyprctl` and its event socket.

## Options

All under `programs.eww-bar`.

| Option | Default | Effect |
| --- | --- | --- |
| `enable` | `false` | Install the bar and its user services |
| `modules` | every module | Modules the bar has at all. One left out has no widget, no Settings row and no collectors |
| `defaultHidden` | `[ ]` | Modules hidden until shown from Settings > Bar modules |
| `battery.enable` | `false` | The battery module and battery state collection |
| `laptopControls.enable` | `false` | Clamshell and headless display modes, and the lid inhibitor service |
| `wifi.enable` | `false` | The Wi-Fi toggle in the network popup |
| `colors` | Catppuccin Mocha (`palette.nix`) | Colours the stylesheet reads, one SCSS variable each |
| `ai.ccusage.package` | `pkgs.ccusage` | Token and cost totals for the AI usage module; `null` leaves them out |
| `ai.openusage.package` | `null` | `openusage-cli`, for the Codex and Antigravity quota cards; `null` leaves them out |
| `ai.codexHome` | `~/.codex` | `CODEX_HOME` for the openusage-cli probes |
| `ai.claude.usageSnapshot` | `null` | claude-hud's `display.externalUsageWritePath`. The Claude card then shows the newest of that file and the last usage API answer, asks the API only when both are over 5 minutes old, and labels a reading over 35 minutes old "as of" |
| `blurNamespaces` | read-only | The bar's glass layer-shell namespaces, for the compositor's blur rules |

The module names are listed in `go/internal/collect/modules.json`, in bar order.
The bar keeps that order whatever the order in `modules`.

## Package

`package.nix` builds the bar without the module: the Go backend in `bin/` and the
rendered eww config in `share/eww-bar/`. It takes `colors`, `modules`,
`defaultHidden`, `enableBattery`, `enableLaptopControls` and `enableWifi` as
`callPackage` arguments, with the same defaults as the options above. The backend is
its own derivation (`passthru.backend`), so changing a setting never recompiles Go.
The flake exports it with its defaults as `packages.<system>.eww-bar`.

## Layout

| Path | Contents |
| --- | --- |
| `module.nix` | The Home Manager module: options, user services, config files |
| `eww-systray-uncached.patch` | Applied to the eww the module runs: tray items are read without zbus's property cache, so an item that registers Passive and turns Active later, as libayatana-appindicator apps such as Spotify do, still shows, and icons follow `NewIcon` |
| `package.nix` | The package: Go backend plus rendered config |
| `bar-modules.nix` | The module list for Nix, read from `go/internal/collect/modules.json` |
| `palette.nix` | The default colours |
| `go/` | The backend, `eww-barctl` and `eww-popup` |
| `config/` | `eww.yuck`, `eww.scss` and the icons |
| `tests/` | The two checks `checks.eww-backend` runs besides `go test` |

## Checks

- `checks.eww-backend` runs `go test ./...`, checks that `eww.yuck`'s initial
  snapshot matches the backend's, and that `eww.scss` only uses pseudo-classes GTK
  implements.
- `checks.eww-bar-standalone` evaluates the module in a bare Home Manager
  configuration on plain nixpkgs, without this repository's overlays.
- `checks.eww-bar-self-contained` fails if a `.nix` file here refers to `../`.

`go test ./...` in `go/` is the fast loop while working on the backend.
