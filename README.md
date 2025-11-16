# NixOS Configuration

A modular, flake-based NixOS and Home Manager configuration supporting multiple system profiles (WSL, laptop, desktop). This repository provides a comprehensive development environment with extensive CLI tools, Neovim configuration, and optional desktop environments.

## Features

- **Modular Architecture**: Feature flags for granular control over enabled components
- **Multiple Profiles**: Pre-configured setups for WSL, laptop, and desktop systems
- **Declarative Neovim**: Complete Neovim configuration using nixvim with LSP, completion, and extensive plugin ecosystem
- **Desktop Environments**: Support for Hyprland (Wayland) and GNOME
- **Development Tools**: Integrated support for multiple programming languages and development workflows
- **Consistent Theming**: Catppuccin theme across all applications
- **WSL Integration**: First-class support for Windows Subsystem for Linux

## Quick Start

### Prerequisites

- **NixOS**: Version 23.11 or later (for NixOS installations)
- **Nix**: With flakes enabled (for non-NixOS installations)
- **Git**: For cloning the repository

### NixOS on WSL

This setup is based on [nixos-wsl-starter](https://github.com/LGUG2Z/nixos-wsl-starter).

1. **Download NixOS-WSL**

   Get the latest release from [NixOS-WSL](https://github.com/nix-community/NixOS-WSL/releases/latest)

2. **Import into WSL**

   ```powershell
   wsl --import NixOS .\NixOS\ .\nixos-wsl.tar.gz --version 2
   ```

3. **Enter the distribution**

   ```powershell
   wsl -d NixOS
   ```

4. **Set up Nix channels**

   ```bash
   sudo nix-channel --add https://nixos.org/channels/nixos-unstable nixos
   sudo nix-channel --update
   ```

5. **Clone and apply configuration**

   ```bash
   nix-shell -p git --run "git clone https://github.com/lemonilemon/nixos-config.git /tmp/configuration && \
   sudo nixos-rebuild switch --flake /tmp/configuration#NixOS-wsl"
   ```

6. **Restart WSL**

   ```powershell
   wsl -t NixOS
   wsl -d NixOS
   ```

7. **Move configuration to home directory**

   ```bash
   mv /tmp/configuration ~/nixos-config
   ```

### Native NixOS Installation

For laptop or desktop installations:

1. **Clone the repository**

   ```bash
   git clone https://github.com/lemonilemon/nixos-config.git /etc/nixos
   cd /etc/nixos
   ```

2. **Copy and modify hardware configuration**

   ```bash
   # Generate hardware configuration
   nixos-generate-config --show-hardware-config > profiles/<your-profile>/hardware-configuration.nix
   ```

3. **Apply configuration**

   ```bash
   sudo nixos-rebuild switch --flake .#<profile>
   ```

   Available profiles: `laptop`, `desktop`

### Non-NixOS Linux (Home Manager Only)

For using Home Manager on non-NixOS distributions:

1. **Install prerequisite packages**

   ```bash
   # Debian/Ubuntu
   sudo apt update
   sudo apt install curl xz-utils git
   ```

2. **Install Nix (single-user)**

   ```bash
   sh <(curl -L https://nixos.org/nix/install) --no-daemon
   . ~/.nix-profile/etc/profile.d/nix.sh
   ```

3. **Enable flakes**

   ```bash
   mkdir -p ~/.config/nix
   echo "experimental-features = nix-command flakes" >> ~/.config/nix/nix.conf
   ```

4. **Clone and apply configuration**

   ```bash
   git clone https://github.com/lemonilemon/nixos-config.git ~/.config/home-manager
   cd ~/.config/home-manager
   nix run home-manager/master switch
   ```

5. **Configure shell**

   ```bash
   echo $(which zsh) | sudo tee -a /etc/shells
   chsh -s $(which zsh) $USER
   sudo locale-gen  # If locale not configured
   ```

6. **Restart shell or WSL**

   ```bash
   # For WSL
   wsl -t <DistroName>
   wsl -d <DistroName>

   # For native Linux
   exec zsh
   ```

## Repository Structure

```
.
├── flake.nix           # Main flake configuration and system definitions
├── flake.lock          # Locked flake dependencies
├── Justfile            # Command runner recipes for common tasks
├── CLAUDE.md           # AI assistant guidance for this repository
├── lib/                # Helper functions and builders
├── modules/            # Modular configuration components
│   ├── cli/           # CLI tools and terminal configuration
│   ├── desktop/       # Desktop environment modules
│   ├── general/       # General system settings
│   ├── gui/           # GUI applications
│   └── home.nix       # Home Manager entry point
├── profiles/          # Machine-specific configurations
│   ├── wsl/          # WSL profile
│   ├── laptop/       # Laptop profile
│   └── desktop/      # Desktop profile
├── overlays/         # Package overlays and custom packages
└── nixpkgs/          # Nixpkgs configuration
```

## Usage

This repository uses [`just`](https://github.com/casey/just) as a command runner for common operations.

### Essential Commands

```bash
# List all available commands
just

# Update all flake inputs
just update

# Update specific input
just updatep nixpkgs

# Format all Nix files
just fmt

# Build and switch to new configuration
just build

# Dry-build to preview changes
just dry-build

# View system generations
just history

# Test configuration for errors
just test
```

### Manual Rebuild

```bash
# For WSL
sudo nixos-rebuild switch --flake .#NixOS-wsl

# For laptop
sudo nixos-rebuild switch --flake .#laptop

# For desktop
sudo nixos-rebuild switch --flake .#desktop
```

## Configuration

### Module System

This configuration uses a hierarchical module system with feature flags:

**Top-level options** (defined in `modules/options.nix`):
- `cli.enable` - CLI tools and terminal configuration
- `gui.enable` - GUI applications
- `general.enable` - General system settings
- `desktop.enable` - Desktop environment

**Module categories**:
- **CLI**: Neovim, shells (zsh, starship), git, zellij, CLI utilities
- **General**: Nix settings, fonts, programming languages, utilities
- **Desktop**: Hyprland, GNOME, display managers, theming
- **GUI**: Browsers, development tools with GUI

Each module can be enabled/disabled independently in your profile configuration.

### Customization

#### Creating a Custom Profile

1. Create a new directory in `profiles/`:

   ```bash
   mkdir -p profiles/myprofile
   ```

2. Create `profiles/myprofile/default.nix`:

   ```nix
   { pkgs, username, ... }:
   {
     imports = [
       ./hardware-configuration.nix
       ./config.nix
       ../base.nix
     ];

     # Your custom configuration
     environment.sessionVariables = {
       NIXHOST = "myprofile";
     };
   }
   ```

3. Add to `flake.nix`:

   ```nix
   nixosConfigurations = {
     myprofile = helpers.mkSystem {
       system = "x86_64-linux";
       inherit username hostname;
       profile = "myprofile";
     };
   };
   ```

#### Disabling Features

In your profile's `config.nix`:

```nix
{
  # Disable desktop environment
  desktop.enable = false;

  # Disable GUI applications
  gui.enable = false;
}
```

#### Adding Packages

**System-wide packages** (in profile or module):
```nix
environment.systemPackages = with pkgs; [
  your-package
];
```

**User packages** (via Home Manager):
```nix
home.packages = with pkgs; [
  your-package
];
```

## Development

### Pre-commit Hooks

This repository includes pre-commit hooks for code formatting:

```bash
# Enter development shell
nix develop

# Hooks will run automatically on commit
git commit
```

### Formatter

```bash
# Format all Nix files using nixfmt-rfc-style
nix fmt
```

## Included Software

### CLI Tools
- **Shell**: zsh with starship prompt, zoxide directory navigation
- **Editor**: Neovim with extensive plugin configuration (LSP, completion, AI integration)
- **Terminal Multiplexer**: Zellij
- **Version Control**: Git with custom configuration
- **System Monitoring**: fastfetch, atuin (shell history)
- **AI Tools**: Integration with Claude and GitHub Copilot

### Desktop Environment (Optional)
- **Window Manager**: Hyprland (Wayland compositor)
- **Alternative DE**: GNOME
- **Display Managers**: SDDM (default), GDM
- **Status Bar**: Waybar
- **Utilities**: rofi, wlogout, hyprlock, hyprpaper

### Development
- **Languages**: gcc, python3 (configurable via `home.general.programlangs.packages`)
- **LSP Support**: Configured in Neovim for multiple languages
- **Formatters**: Language-specific formatters via conform.nvim

## Troubleshooting

### Flake Evaluation Errors

```bash
# Check for evaluation errors
nix flake check

# Show detailed trace
just test
```

### Permission Issues in WSL

Ensure your user is in the correct groups:
```bash
groups $USER
```

### Rebuilding After Changes

Always test before applying:
```bash
just dry-build  # Preview changes
just test       # Check for errors
just build      # Apply if no issues
```

## In Development

- LaTeX and Markdown preview in Neovim
- Additional desktop environment features

## References

This configuration draws inspiration from:

- [ryan4yin/nix-config](https://github.com/ryan4yin/nix-config)
- [librephoenix/nixos-config](https://gitlab.com/librephoenix/nixos-config)
- [NixOS-WSL](https://github.com/nix-community/NixOS-WSL)
- [nixvim](https://github.com/nix-community/nixvim)

## License

This project is licensed under the MIT License - see the [LICENSE](LICENSE) file for details.

## Contributing

This is a personal configuration repository, but issues and suggestions are welcome. Feel free to fork and adapt for your own use.
