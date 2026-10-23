# Desktop Module

This module provides desktop environment configurations, including window managers, display managers, and desktop-specific applications. It supports both Hyprland (Wayland compositor) and GNOME desktop environments.

## Overview

The Desktop module handles graphical desktop environment setup:
- **Hyprland**: Modern Wayland compositor with tiling window management
- **GNOME**: Full-featured desktop environment
- **Display Managers**: SDDM and GDM for login screens
- **Desktop Applications**: File managers, image viewers, status bars, and utilities
- **System Integration**: Audio, Bluetooth, network management
- **Theming**: Consistent appearance across applications

## Structure

```
desktop/
├── default.nix          # Module entry point
├── options.nix          # Feature flag definitions
├── home/                # Home Manager desktop configurations
│   ├── default.nix     # Home Manager module entry
│   ├── options.nix     # Home-specific options
│   └── hyprland/       # Hyprland configuration
│       ├── default.nix        # Hyprland entry point
│       ├── waybar.nix         # Status bar (fallback when Eww is off)
│       ├── eww/               # Maps the eww options onto programs.eww-bar (packages/eww-bar/)
│       ├── dunst.nix          # Notification daemon
│       ├── hyprlock.nix       # Screen locker
│       ├── hyprpaper/         # Static wallpaper engine (fallback)
│       ├── wallpaper/         # awww wallpaper engine and picker backend
│       ├── rofi.nix           # App launcher / window switcher
│       ├── wlogout/           # Logout menu
│       ├── theme/             # GTK/Qt theming
│       ├── network/           # Network manager applet
│       ├── bluetooth/         # Bluetooth manager
│       ├── nemo.nix           # File manager (GUI)
│       └── imv.nix            # Image viewer
└── nixos/              # NixOS system-level desktop settings
    ├── default.nix     # NixOS module entry
    ├── hyprland.nix    # Hyprland system configuration
    ├── gnome.nix       # GNOME system configuration
    └── display-manager/
        ├── default.nix # Display manager entry
        ├── sddm.nix    # SDDM configuration
        └── gdm.nix     # GDM configuration
```

## Feature Flags

Control Desktop module components:

### NixOS Options
```nix
nixos.desktop.enable              # Enable desktop environment (default: true if nixos and desktop enabled)
nixos.desktop.hyprland.enable     # Enable Hyprland (default: follows desktop.enable)
nixos.desktop.gnome.enable        # Enable GNOME (default: follows desktop.enable)
nixos.desktop.displayManager      # Display manager choice: "sddm" or "gdm" (default: "sddm")
nixos.desktop.sddmTheme           # SDDM greeter: "orrery" or "astronaut" (default: "orrery")
```

### Home Manager Options
```nix
home.desktop.enable              # Enable desktop home configuration (default: true if home and desktop enabled)
home.desktop.hyprland.enable     # Enable Hyprland user config (default: follows desktop.enable)
home.desktop.hyprland.eww.enable                    # Enable Eww bar for Hyprland
home.desktop.hyprland.eww.laptopControls.enable     # Enable laptop clamshell/headless controls
home.desktop.hyprland.eww.modules                   # Modules the bar has at all (default: every module)
home.desktop.hyprland.eww.defaultHiddenModules      # Modules hidden until shown in Settings (laptop: [ "storage" ])
home.desktop.hyprland.waybar.enable                 # Enable Waybar when Eww is disabled
```

## Desktop Environments

### Hyprland

A dynamic tiling Wayland compositor with modern features.

**System Configuration** (`nixos/hyprland.nix`):
- Hyprland package installation
- Wayland session configuration
- XWayland support
- Portal integration (file picker, screen sharing)
- Graphics driver integration

**User Configuration** (`home/hyprland/`):

#### Core Components

**Waybar** (`waybar.nix`):
- Top status bar with system information
- Workspace indicators
- System tray
- Clock and date
- Audio control
- Network status
- Battery indicator (for laptops)
- Custom modules

**Eww** (`eww/`):
- Top status bar replacement for Waybar
- Workspace, media, system, AI usage, and session controls
- Codex quota details include available manual resets and their earliest expiry
- The Claude quota card reads claude-hud's usage snapshot
  (`home.cli.programs.claudeHud.usageSnapshot`) and calls the usage API only when
  the snapshot and its last API answer are both over 5 minutes old, so a running
  Claude Code session keeps the card current. An older reading is labelled
  "as of" rather than dropped when the OAuth token has expired
- Hypridle pause backed by a systemd user inhibitor
- Optional laptop controls for external-only and headless/server display modes
- Storage module: free space on the filesystem holding `/nix/store`. Its popup
  shows the nh cleanup timer's last and next run (and what the last run freed,
  from the journal), and opens `nh-clean-now` in kitty, baobab, or BleachBit
- Settings > Bar modules shows or hides each module. Choices are written to
  `~/.config/eww-bar/modules.json`, an unmanaged file like
  `~/.config/hypr/monitors.lua`, as overrides on top of the host's
  `defaultHiddenModules`; "Reset to defaults" empties it. Hand edits take effect
  on the next `systemctl --user restart eww-bar`. The module names are listed
  once, in `packages/eww-bar/go/internal/collect/modules.json`: the Go backend
  embeds it and `bar-modules.nix` reads it
- `home.desktop.hyprland.eww.modules` sets which modules a host's bar has at all.
  A module left out has no widget, no row in Settings > Bar modules and no
  collectors: the backend starts neither its watcher process (`nmcli`, `pactl`,
  `playerctl`, `dbus-monitor`, `bluetoothctl`) nor its polls and probes. Hiding a
  module in Settings only hides it
- The bar is a self-contained package in `packages/eww-bar/`: `package.nix`
  builds the Go backend (`go/`) and renders the widgets and styles (`config/`),
  `module.nix` is its Home Manager module (`programs.eww-bar`: services, config
  files, options), and `tests/` holds the two checks the `eww-backend` flake check
  runs. `eww/default.nix` only maps the `home.desktop.hyprland.eww.*` options onto
  `programs.eww-bar`, so profiles keep setting the options they always have, and
  passes in what only this repo has: its palette, the overlay's `openusage-cli`
  and its `ccusage` wrapper. Outside this repo the module defaults to Catppuccin
  Mocha, nixpkgs' `ccusage` and no openusage-cli. Hyprland's blur rules read the
  bar's glass layers from `programs.eww-bar.blurNamespaces`
- The flake exports the bar as `packages.<system>.eww-bar` and
  `homeManagerModules.eww-bar`. `checks.eww-bar-standalone` evaluates the module
  in a bare Home Manager configuration on plain nixpkgs, and
  `checks.eww-bar-self-contained` fails if a `.nix` file under `packages/eww-bar/`
  reaches outside it

**Hyprlock** (`hyprlock.nix`):
- Screen locking utility
- Authentication integration
- Customizable lock screen
- Automatic locking after idle timeout

**Hyprpaper** (`hyprpaper/`):
- Wallpaper management
- Per-monitor wallpapers
- Dynamic wallpaper changing

**Rofi** (`rofi.nix`):
- App launcher, window switcher, and run dialog
- Third member of the hyprlock/wlogout session-overlay family: glass panel over
  compositor blur (the `rofi` layerrule in `default.nix`), hyprlock's input
  field as the search bar, selection styled with wlogout's hover formula.
  Colours come from `theme/palette.nix`; the catppuccin module's stock rofi
  theme is opted out like the rest of the family

**wlogout** (`wlogout/`):
- Logout/power menu
- Shutdown, reboot, suspend options
- Visual confirmation dialog
- Keyboard shortcuts, shown on the buttons
- Styled as hyprlock's sibling rather than as an eww bar island: fullscreen
  session overlays are a separate tier from the bar's glass chrome. Colours come
  from `theme/palette.nix`; the blurred backdrop comes from the `logout_dialog`
  layerrule in `default.nix`
- Icons are rasterised at build time from `nerd-fonts.jetbrains-mono`, one per
  accent, because a CSS `background-image` cannot inherit `color` and so could
  never take the button's accent on hover

#### Desktop Applications

**File Manager**:
- **Nemo** (`nemo.nix`): Graphical file manager (fork of Nautilus)

**Image Viewer**:
- **imv** (`imv.nix`): Lightweight Wayland image viewer
  - Fast and minimal
  - Keyboard-driven
  - Supports common image formats

#### System Integration

**Network Management** (`network/`):
- Network manager applet
- WiFi connection management
- VPN integration
- Connection status in system tray
- `nm-connection-editor` on PATH, opened by the Eww network popup's "Edit connections…"

**Bluetooth** (`bluetooth/`):
- Bluetooth manager applet
- Device pairing and connection
- Audio device management
- System tray integration

**Theming** (`theme/`):
- GTK theme configuration
- Qt theme integration
- Icon themes
- Cursor themes
- Catppuccin theming across applications

#### Hyprland Configuration

The main Hyprland config (`default.nix`) renders to the Lua config format
(`~/.config/hypr/hyprland.lua`, `configType = "lua"`) — Hyprland 0.57 removes
the legacy hyprlang `.conf` format. Monitor overrides from nwg-displays are
loaded from the unmanaged `~/.config/hypr/monitors.lua` when present. It
includes:
- Window management rules
- Workspace configuration
- Keybindings
- Animations and effects
- Monitor setup
- Input device configuration
- Startup applications
- Full session environment imported into systemd and D-Bus at startup
- Integration with waybar, hyprlock, etc.

**Key Features**:
- Automatic tiling with manual override
- Multiple workspaces
- Floating window support
- Apps cannot maximize themselves: maximize requests from every window are
  suppressed, so a new window tiles instead of covering the others
- Window gaps and borders
- Smooth animations
- Per-monitor workspace binding

### GNOME

A complete desktop environment with traditional workflow.

**System Configuration** (`nixos/gnome.nix`):
- GNOME package installation
- GNOME services
- GDM display manager (typical pairing)
- GNOME Shell extensions
- Default applications

**Features**:
- Full-featured desktop environment
- Activities overview
- Application grid
- System settings
- Built-in applications (Files, Settings, etc.)
- GNOME extensions support

**Note**: GNOME is disabled by default in laptop and desktop profiles (they use Hyprland). To enable:
```nix
nixos.desktop.gnome.enable = true;
nixos.desktop.displayManager = "gdm";  # Optional: use GDM with GNOME
```

## Display Managers

Display managers handle the login screen and session management.

### SDDM (Default)

**Configuration** (`nixos/display-manager/sddm.nix`):
- Modern Qt-based display manager
- Wayland support
- Theme: Orrery by default, from the `sddm-orrery` flake input
  ([lemonilemon/sddm-orrery](https://github.com/lemonilemon/sddm-orrery)): the
  blog's night sky and the inner planets at their real positions, in
  astronaut's layout. Its IBM Plex fonts go into `fonts.packages`, because the
  greeter only sees system fonts. `nixos.desktop.sddmTheme = "astronaut"`
  switches back to stock sddm-astronaut
- Preview the theme before a rebuild with `nix run github:lemonilemon/sddm-orrery#preview`
- Auto-login options
- Session selection (Hyprland's UWSM session entry is hidden)

**Best for**: Hyprland and KDE Plasma setups

### GDM

**Configuration** (`nixos/display-manager/gdm.nix`):
- GNOME Display Manager
- Wayland-first
- GNOME integration
- Accessibility features

**Best for**: GNOME desktop environment

### Switching Display Managers

In your profile's `config.nix`:
```nix
{
  nixos.desktop.displayManager = "gdm";  # or "sddm"
}
```

## Usage

### Enabling Desktop

In your profile's `config.nix`:
```nix
{
  desktop.enable = true;  # Enable desktop environment
}
```

### Choosing Desktop Environment

**Hyprland only** (default for laptop/desktop profiles):
```nix
{
  desktop.enable = true;
  nixos.desktop.gnome.enable = false;
  nixos.desktop.hyprland.enable = true;
  nixos.desktop.displayManager = "sddm";
}
```

**GNOME only**:
```nix
{
  desktop.enable = true;
  nixos.desktop.gnome.enable = true;
  nixos.desktop.hyprland.enable = false;
  nixos.desktop.displayManager = "gdm";
}
```

**Both** (choose at login):
```nix
{
  desktop.enable = true;
  nixos.desktop.gnome.enable = true;
  nixos.desktop.hyprland.enable = true;
}
```

### Disabling Desktop (CLI-only)

In your profile's `config.nix`:
```nix
{
  desktop.enable = false;  # Disable all desktop environments
}
```

This is the default for the WSL profile.

## Customization

### Hyprland Keybindings

Keybindings are configured in `home/hyprland/default.nix`. To modify:

1. Edit the Hyprland configuration
2. Follow Hyprland's keybinding syntax
3. Rebuild Home Manager configuration

Common bindings:
- `Super + Return`: Open terminal
- `Super + Q`: Close window
- `Super + [1-9]`: Switch workspace
- `Super + Shift + [1-9]`: Move window to workspace
- `Super + Mouse drag`: Move/resize windows

### Waybar Customization

Waybar configuration in `waybar.nix` allows customizing:
- Module order and visibility
- Styling (CSS)
- Custom modules
- Update intervals
- Click actions

### Theming

Theme configuration in `theme/` directory:
- GTK themes for GTK applications
- Qt themes for Qt applications
- Icon themes
- Cursor themes
- Color schemes (Catppuccin integration)

**Changing theme**:
```nix
# In theme configuration
gtk.theme.name = "Catppuccin-Mocha";
gtk.iconTheme.name = "Papirus-Dark";
```

### Wallpapers

Add wallpapers in `hyprpaper/` configuration:
```nix
# hyprpaper configuration
wallpapers = [
  "monitor1,/path/to/wallpaper.png"
  "monitor2,/path/to/other-wallpaper.png"
];
```

## Application Integration

### File Manager

**Nemo** (GUI):
- Default file manager for Hyprland
- Thumbnail support
- Archive integration
- Terminal integration

**Ranger** (TUI):
- Vim-like keybindings
- Image previews in terminal
- File operations
- Bulk renaming

### Image Viewer

**imv**:
- Lightweight and fast
- Wayland-native
- Keyboard navigation
- Slideshow mode

## System Services

Desktop environment services are automatically managed:
- **PipeWire**: Audio server (configured in profiles)
- **Bluetooth**: Device management (configured in profiles)
- **NetworkManager**: Network connections (configured in profiles)
- **Polkit**: Permission management
- **Portal**: Desktop integration for file pickers, screen sharing

## Best Practices

1. **Choose one DE**: Don't enable both GNOME and Hyprland unless you need to switch between them
2. **Match display manager**: Use SDDM with Hyprland, GDM with GNOME for best integration
3. **Theming**: Use consistent themes across GTK and Qt applications
4. **Wallpapers**: Store wallpapers in a consistent location (e.g., `~/Pictures/Wallpapers/`)
5. **Keybindings**: Document custom keybindings for future reference
6. **Window rules**: Use Hyprland window rules for consistent application behavior

## Troubleshooting

### Hyprland Won't Start

1. Check Hyprland logs: `journalctl --user -u hyprland`
2. Verify graphics drivers are working
3. Check Wayland support: `echo $XDG_SESSION_TYPE` (should be "wayland")
4. Try starting from TTY: `Hyprland`

### Display Manager Not Showing

1. Check display manager service: `systemctl status display-manager`
2. Verify correct display manager is enabled in configuration
3. Check for errors: `journalctl -u display-manager`

### Waybar Not Appearing

1. Check if Waybar is running: `ps aux | grep waybar`
2. Verify Waybar is in Hyprland autostart
3. Check Waybar logs for errors
4. Restart Waybar: `killall waybar && waybar &`

### Theme Not Applying

1. Rebuild Home Manager configuration
2. Log out and log back in
3. For GTK apps: Check `GTK_THEME` environment variable
4. For Qt apps: Verify Qt platform theme is set

### Screen Locking Issues

1. Check hyprlock service status
2. Verify idle timeout configuration
3. Test manual lock: `hyprlock`
4. Check PAM configuration for authentication

## Integration with Other Modules

- **CLI**: Terminal applications work in desktop environment
- **General**: Fonts from general module used in desktop
- **GUI**: GUI applications integrate with desktop environment

## Dependencies

External flake inputs:
- `hyprland`: Hyprland compositor
- `catppuccin`: Theming
- `sddm-orrery`: SDDM greeter theme
- System packages from nixpkgs for desktop applications

## Notes

- Hyprland is Wayland-only; X11 applications run via XWayland
- GNOME includes many applications by default; Hyprland is minimal by design
- Desktop module can be disabled for headless/server systems
- WSL profile disables desktop by default
