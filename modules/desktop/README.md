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
│       ├── waybar.nix         # Status bar
│       ├── hyprlock.nix       # Screen locker
│       ├── hyprpaper/         # Wallpaper manager
│       ├── wlogout/           # Logout menu
│       ├── theme/             # GTK/Qt theming
│       ├── network/           # Network manager applet
│       ├── bluetooth/         # Bluetooth manager
│       ├── nemo.nix           # File manager (GUI)
│       ├── ranger.nix         # File manager (TUI)
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
```

### Home Manager Options
```nix
home.desktop.enable              # Enable desktop home configuration (default: true if home and desktop enabled)
home.desktop.hyprland.enable     # Enable Hyprland user config (default: follows desktop.enable)
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

**Hyprlock** (`hyprlock.nix`):
- Screen locking utility
- Authentication integration
- Customizable lock screen
- Automatic locking after idle timeout

**Hyprpaper** (`hyprpaper/`):
- Wallpaper management
- Per-monitor wallpapers
- Dynamic wallpaper changing

**wlogout** (`wlogout/`):
- Logout/power menu
- Shutdown, reboot, suspend options
- Visual confirmation dialog
- Keyboard shortcuts

#### Desktop Applications

**File Managers**:
- **Nemo** (`nemo.nix`): Graphical file manager (fork of Nautilus)
- **Ranger** (`ranger.nix`): Terminal-based file manager with vim keybindings

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

The main Hyprland config (`default.nix`) includes:
- Window management rules
- Workspace configuration
- Keybindings
- Animations and effects
- Monitor setup
- Input device configuration
- Startup applications
- Integration with waybar, hyprlock, etc.

**Key Features**:
- Automatic tiling with manual override
- Multiple workspaces
- Floating window support
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
- Theme customization
- Auto-login options
- Session selection

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
- System packages from nixpkgs for desktop applications

## Notes

- Hyprland is Wayland-only; X11 applications run via XWayland
- GNOME includes many applications by default; Hyprland is minimal by design
- Desktop module can be disabled for headless/server systems
- WSL profile disables desktop by default
