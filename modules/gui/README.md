# GUI Module

This module provides graphical user interface applications including web browsers, development tools, terminal emulators, and various GUI utilities. It complements the Desktop module by providing applications that run within the desktop environment.

## Overview

The GUI module handles installation and configuration of:
- **Web Browsers**: Firefox, Zen Browser
- **Development Tools**: Web development environments, IDEs
- **Terminal Emulator**: Kitty terminal
- **Applications**: General GUI applications and utilities

**Note**: The GUI module requires a desktop environment to be enabled. It is automatically disabled in CLI-only profiles like WSL.

## Structure

```
gui/
├── default.nix          # Module entry point
├── options.nix          # Feature flag definitions
├── home/                # Home Manager GUI configurations
│   ├── default.nix     # Home Manager module entry
│   ├── options.nix     # Home-specific option definitions
│   ├── browsers/       # Web browser configurations
│   │   ├── default.nix        # Browser module entry
│   │   ├── firefox.nix        # Firefox configuration
│   │   └── zen.nix            # Zen Browser configuration
│   ├── development/    # Development tool configurations
│   │   ├── default.nix        # Development module entry
│   │   └── web.nix            # Web development tools
│   ├── apps/           # General GUI applications
│   │   └── default.nix        # Apps module entry
│   └── kitty.nix       # Kitty terminal emulator
└── nixos/              # NixOS system-level GUI settings
    ├── default.nix     # NixOS module entry
    ├── apps/           # System-level applications
    │   ├── default.nix        # Apps module entry
    │   └── 1password.nix      # 1Password integration
    └── development/    # System-level development tools
        ├── default.nix        # Development module entry
        └── web.nix            # Web development system packages
```

## Feature Flags

Control GUI module components via options:

### Home Manager Options
```nix
home.gui.enable                    # Enable all GUI applications (default: true if home and gui enabled)

# Browsers
home.gui.browsers.enable           # Enable browsers (default: follows gui.enable)
home.gui.browsers.firefox.enable   # Enable Firefox (default: follows browsers.enable)
home.gui.browsers.zen.enable       # Enable Zen Browser (default: follows browsers.enable)

# Development
home.gui.development.enable        # Enable dev tools (default: follows gui.enable)
home.gui.development.web.enable    # Enable web dev tools (default: follows development.enable)

# Applications
home.gui.apps.enable               # Enable general apps (default: follows gui.enable)

# Terminal
home.gui.kitty.enable              # Enable Kitty terminal (default: follows gui.enable)
```

### NixOS Options
```nix
nixos.gui.enable                   # Enable system-level GUI settings (default: true if nixos and gui enabled)
nixos.gui.development.enable       # Enable system dev tools (default: follows gui.enable)
nixos.gui.development.web.enable   # Enable system web dev tools (default: follows development.enable)
nixos.gui.apps.enable              # Enable system apps (default: follows gui.enable)
```

## Components

### Web Browsers (`home/browsers/`)

#### Firefox (`firefox.nix`)

Mozilla Firefox web browser with customization support.

**Features**:
- Privacy-focused default settings
- Extension management via Home Manager
- Profile configuration
- Search engine customization
- Bookmark synchronization
- Performance optimizations

**Customization**:
```nix
programs.firefox = {
  enable = true;
  profiles.default = {
    # Add extensions
    extensions = with pkgs.nur.repos.rycee.firefox-addons; [
      ublock-origin
      privacy-badger
    ];

    # Custom settings
    settings = {
      "browser.startup.homepage" = "https://nixos.org";
      "privacy.trackingprotection.enabled" = true;
    };
  };
};
```

#### Zen Browser (`zen.nix`)

A Firefox-based browser with enhanced privacy and customization features.

**Features**:
- Based on Firefox
- Privacy-focused defaults
- Custom UI modifications
- Performance enhancements
- Modern design

**Source**: Installed via the `zen-browser` flake input

**Note**: Zen Browser is relatively new and may have fewer extensions compared to Firefox.

### Development Tools (`home/development/`, `nixos/development/`)

#### Web Development (`web.nix`)

Tools and utilities for web development.

**Typical packages** (examples):
- **IDEs/Editors**: VSCode, VSCodium
- **API Testing**: Postman, Insomnia
- **Database Tools**: DBeaver, pgAdmin
- **Version Control GUIs**: GitKraken, GitHub Desktop
- **Design Tools**: Figma (web), design utilities

**Browser Developer Tools**:
- Firefox Developer Edition (optional)
- Chrome/Chromium DevTools
- Browser extension development tools

**Build Tools**:
- Node.js (from general module)
- Package managers (npm, yarn, pnpm)
- Bundlers and build systems

### Terminal Emulator (`kitty.nix`)

Kitty is a fast, feature-rich, GPU-based terminal emulator.

**Features**:
- GPU acceleration for rendering
- Image display in terminal
- Multiple windows and tabs
- Customizable keybindings
- Rich configuration options
- Font ligatures support
- True color support
- Tmux/Zellij integration

**Configuration**:
```nix
programs.kitty = {
  enable = true;
  theme = "Catppuccin-Mocha";  # Consistent with system theme
  settings = {
    font_family = "JetBrainsMono Nerd Font";
    font_size = 12;
    # ... additional settings
  };
};
```

**Key Features**:
- Integrates well with Hyprland (Wayland-native)
- Fast performance with large scrollback
- Scriptable via kitty remote control
- Split windows without tmux

### Applications (`home/apps/`, `nixos/apps/`)

General GUI applications and utilities.

**Home Manager Apps** (`home/apps/default.nix`):
- User-specific applications
- Personal productivity tools
- Media applications
- Communication apps

**System Apps** (`nixos/apps/`):
- System-wide applications
- **1Password** (`1password.nix`): Password manager with system integration
  - Browser integration
  - SSH agent
  - CLI tool
  - System authentication

## Usage

### Enabling GUI Module

In your profile's `config.nix`:
```nix
{
  gui.enable = true;  # Enable all GUI applications
}
```

**Note**: Automatically enabled for desktop and laptop profiles, disabled for WSL.

### Selective Component Control

```nix
{
  gui.enable = true;

  # Use only Firefox, not Zen
  home.gui.browsers.zen.enable = false;

  # Disable development tools
  home.gui.development.enable = false;

  # Disable Kitty if using another terminal
  home.gui.kitty.enable = false;
}
```

### Disabling GUI (CLI-only)

In your profile's `config.nix`:
```nix
{
  gui.enable = false;  # Disable all GUI applications
}
```

This is the default for WSL profile.

## Browser Selection

### Using Multiple Browsers

Both Firefox and Zen can be enabled simultaneously:
```nix
{
  home.gui.browsers.firefox.enable = true;
  home.gui.browsers.zen.enable = true;
}
```

Set your default browser via system settings or XDG defaults:
```nix
xdg.mimeApps.defaultApplications = {
  "text/html" = "firefox.desktop";
  "x-scheme-handler/http" = "firefox.desktop";
  "x-scheme-handler/https" = "firefox.desktop";
};
```

### Browser-Specific Use Cases

**Firefox**: Best for:
- Mature extension ecosystem
- Long-term stability
- Enterprise deployments
- Web development (DevTools)

**Zen Browser**: Best for:
- Privacy-focused browsing
- Modern UI preferences
- Experimental features
- Customization enthusiasts

## Development Tools

### Web Development Setup

For a complete web development environment:

```nix
{
  home.gui.development.web.enable = true;

  # Additional tools from general module
  home.general.programlangs.packages = with pkgs; [
    nodejs
    bun
    # ... other languages
  ];

  # Editor (from CLI module or add here)
  # Neovim is already configured in CLI module
  # Or add VSCode/VSCodium in development/web.nix
}
```

### IDE Configuration

Add IDE packages in `development/web.nix` or via Home Manager:

```nix
home.packages = with pkgs; [
  vscode
  # or
  vscodium
  # or
  jetbrains.webstorm
];
```

## Terminal Configuration

### Kitty vs Other Terminals

**Kitty** (default in this config):
- GPU-accelerated
- Wayland-native
- Image support
- Best for: Hyprland, modern systems

**Alternatives** (can be configured):
- **Alacritty**: Minimal, fast, GPU-accelerated
- **WezTerm**: Rust-based, feature-rich
- **GNOME Terminal**: Integrated with GNOME
- **Konsole**: KDE's terminal

To use a different terminal:
```nix
{
  home.gui.kitty.enable = false;
  home.packages = [ pkgs.alacritty ];  # or other terminal
}
```

## Application Integration

### Password Manager (1Password)

System-level integration (`nixos/apps/1password.nix`):

**Features**:
- Browser extension integration
- SSH agent support
- CLI tool (`op`)
- System authentication
- Biometric unlock (if supported)

**Setup**:
1. Install: Enabled via `nixos.gui.apps.enable`
2. Sign in to 1Password account
3. Enable browser integration
4. Configure SSH agent (optional)

**SSH Integration**:
```bash
# Use 1Password as SSH agent
export SSH_AUTH_SOCK=~/.1password/agent.sock
```

## Best Practices

1. **Choose one primary browser**: Configure it thoroughly rather than switching frequently
2. **Terminal consistency**: Stick with one terminal emulator for consistency
3. **Development tools**: Install project-specific tools via `nix-shell` or `flake.nix` rather than globally
4. **Application data**: Keep important data in version-controlled dotfiles where appropriate
5. **Extensions/plugins**: Manage browser extensions via Home Manager for reproducibility

## Troubleshooting

### Browser Not Starting

1. Check if desktop environment is running
2. Verify browser package is installed: `which firefox` or `which zen`
3. Check for errors: Run browser from terminal to see output
4. Clear browser cache/profile if corrupted

### Kitty Not Rendering Correctly

1. Verify GPU drivers are properly installed
2. Check Wayland session: `echo $WAYLAND_DISPLAY`
3. Try software rendering: `kitty --config /dev/null`
4. Check font installation: `fc-list | grep <font-name>`

### 1Password Integration Issues

1. Verify 1Password service is running: `systemctl --user status 1password`
2. Check browser extension installation
3. For SSH: Ensure SSH_AUTH_SOCK is set correctly
4. Restart 1Password: `systemctl --user restart 1password`

### Development Tools Missing

1. Check feature flags are enabled
2. Verify package installation: `nix-store -q --tree ~/.nix-profile | grep <package>`
3. Rebuild Home Manager: `home-manager switch`
4. Check PATH: `echo $PATH`

## Integration with Other Modules

- **Desktop**: GUI apps run within desktop environment (Hyprland/GNOME)
- **CLI**: Terminal apps (Neovim) work in Kitty
- **General**: Development tools use language toolchains from general module

## Customization Examples

### Adding a New Browser

1. Create `browsers/chrome.nix`:
   ```nix
   { config, lib, pkgs, ... }:
   {
     config = lib.mkIf config.home.gui.browsers.chrome.enable {
       home.packages = [ pkgs.google-chrome ];
     };
   }
   ```

2. Add option to `options.nix`
3. Import in `browsers/default.nix`

### Adding GUI Development Tools

Edit `development/web.nix`:
```nix
home.packages = with pkgs; [
  vscode
  postman
  dbeaver
  gitkraken
  # ... your tools
];
```

### Custom Kitty Theme

In `kitty.nix`:
```nix
programs.kitty = {
  theme = "Custom";
  extraConfig = ''
    # Custom colors
    foreground #FFFFFF
    background #000000
    # ... more settings
  '';
};
```

## Dependencies

External flake inputs:
- `zen-browser`: Zen Browser package
- `catppuccin`: Consistent theming
- System packages from nixpkgs

## Notes

- GUI module requires `desktop.enable = true` to be useful
- WSL profile disables GUI by default (no display server)
- Browser configurations can be extended with profiles and extensions
- Development tools are best installed per-project via `nix-shell` or direnv when possible
- System-level apps (like 1Password) provide better integration than user-level installations
