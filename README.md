# Modular Cross-Platform NixOS Configuration

A modular, cross-platform NixOS configuration supporting laptop, desktop, and WSL environments with shared components and platform-specific optimizations.

## Features

- **🌐 Cross-Platform Support**: Laptop, Desktop, and WSL configurations
- **🧩 Modular Architecture**: Shared modules with platform-specific customizations  
- **⚡ Flake-Based**: Modern flakes for reproducible builds and dependency management
- **🏠 Home Manager Integration**: User environment management across platforms
- **🎨 Catppuccin Theme**: Consistent theming across all applications
- **🔧 Development Ready**: Neovim, development tools, and language support

## Platform Configurations

This flake provides three main configurations:

| Platform | Configuration | Use Case |
|----------|---------------|----------|
| **Laptop** | `laptop` | Battery optimization, TLP, mobile hardware support |
| **Desktop** | `desktop` | Performance-focused, gaming, high-end hardware |
| **WSL** | `NixOS-wsl` | Windows Subsystem for Linux, CLI-focused |

## Quick Start

### 🖥️ NixOS (Laptop/Desktop)

1. **Clone the repository:**
```bash
git clone https://github.com/lemonilemon/nixos-config.git
cd nixos-config
```

2. **Build and test the configuration:**
```bash
# For laptop
sudo nixos-rebuild build --flake .#laptop

# For desktop  
sudo nixos-rebuild build --flake .#desktop
```

3. **Apply the configuration:**
```bash
# For laptop
sudo nixos-rebuild switch --flake .#laptop

# For desktop
sudo nixos-rebuild switch --flake .#desktop
```

### 🪟 WSL (Windows Subsystem for Linux)

1. **Install NixOS-WSL:**
   - Download the latest [NixOS-WSL release](https://github.com/nix-community/NixOS-WSL/releases/latest)
   - Install in PowerShell/CMD:
```powershell
wsl --import NixOS .\NixOS\ .\nixos-wsl.tar.gz --version 2
wsl -d NixOS
```

2. **Apply the WSL configuration:**
```bash
nix-shell -p git --run "git clone https://github.com/lemonilemon/nixos-config.git /tmp/configuration &&
sudo nixos-rebuild switch --flake /tmp/configuration#NixOS-wsl"
```

3. **Restart WSL and finalize:**
```powershell
wsl -t NixOS
wsl -d NixOS
```

```bash
mv /tmp/configuration ~/nixos-config
```

### 🐧 Home Manager Only (Other Linux Distros)

For non-NixOS systems, you can use just the Home Manager configuration:

1. **Install prerequisites:**
```bash
# For Debian-based distros
sudo apt update && sudo apt install curl xz-utils git
```

2. **Install Nix:**
```bash
sh <(curl -L https://nixos.org/nix/install) --no-daemon
. ~/.nix-profile/etc/profile.d/nix.sh
```

3. **Enable flakes and install:**
```bash
mkdir -p ~/.config/nix
echo "experimental-features = nix-command flakes" >> ~/.config/nix/nix.conf
git clone https://github.com/lemonilemon/nixos-config.git ~/.config/home-manager
nix run home-manager/master -- switch --flake ~/.config/home-manager
```

4. **Set up shell:**
```bash
echo $(which zsh) | sudo tee -a /etc/shells
chsh -s $(which zsh) $USER
zsh
```

## Development Workflow

### Using Justfile Commands (After Initial Setup)

Once you have the configuration applied, you can use the included `just` commands for easier management:

```bash
# List all available commands
just

# Build and switch (set NIXHOST environment variable first)
export NIXHOST=laptop  # or desktop, NixOS-wsl
just build

# Dry build to see what would change
just dry-build

# Switch to different host configuration
just changehost desktop

# Update all flake inputs
just update

# Update specific input
just updatep nixpkgs

# Format nix files
just fmt

# View system generations
just history
```

### Manual Commands (Alternative to Justfile)

```bash
# Test build without applying
sudo nixos-rebuild build --flake .#<configuration>

# Test configuration temporarily  
sudo nixos-rebuild test --flake .#<configuration>

# Build VM for testing
sudo nixos-rebuild build-vm --flake .#<configuration>
./result/bin/run-*-vm

# Update flake inputs
nix flake update
```

### Making Changes

1. **Edit configuration files** in the appropriate module directory
2. **Test your changes:**
```bash
sudo nixos-rebuild build --flake .#<configuration>
# or with justfile:
export NIXHOST=<configuration> && just dry-build
```
3. **Apply if successful:**
```bash  
sudo nixos-rebuild switch --flake .#<configuration>
# or with justfile:
just build
```

### Adding New Platforms

To add support for a new platform:

1. Create `hosts/<hostname>/` directory with hardware configuration
2. Add platform-specific module in `modules/platforms/<platform>/`
3. Update `flake.nix` to include the new configuration

## Project Structure

Understanding the file structure helps you know where to make configuration changes:

```
nixos-config/
├── flake.nix              # 🎯 Main entry point - platform definitions
├── Justfile               # ⚡ Development commands
│
├── hosts/                 # 💻 Hardware-specific configurations
│   └── SpaceNix/          # Current system hostname
│       ├── default.nix    # Host imports and settings
│       ├── boot.nix       # Boot configuration
│       ├── nvidia.nix     # Graphics drivers
│       └── sound.nix      # Audio setup
│
├── modules/               # 🧩 Feature modules (NixOS + Home Manager)
│   ├── cli/               # Command line tools & terminal
│   │   ├── home/          # Home Manager configs (git, nvim, zsh)
│   │   └── nixos/         # System-level CLI tools
│   ├── desktop/           # Desktop environment (Hyprland)
│   │   ├── home/          # User desktop configs
│   │   └── nixos/         # System desktop services
│   ├── gui/               # GUI applications
│   │   ├── home/          # User GUI apps (browsers, kitty)
│   │   └── nixos/         # System GUI services
│   ├── general/           # Base system settings
│   │   ├── home/          # User environment basics
│   │   └── nixos/         # System fundamentals
│   │
│   ├── platforms/         # 🌐 Platform-specific optimizations
│   │   ├── laptop/        # Battery, TLP, brightness controls
│   │   ├── desktop/       # Performance, gaming optimizations
│   │   └── wsl/           # WSL-specific settings
│   │
│   └── common/            # Platform detection & validation
│
├── home-manager/          # 🏠 Home Manager entry point
├── profiles/              # 📋 Complete platform configurations
└── overlays/              # 📦 Package modifications
```

### Configuration Areas

| What to Modify | Where to Look | Examples |
|----------------|---------------|----------|
| **System Packages** | `modules/general/nixos/` | Add system-wide software |
| **User Programs** | `modules/*/home/` | Configure git, zsh, neovim |
| **Desktop Environment** | `modules/desktop/` | Hyprland, waybar, theme |
| **Hardware Settings** | `hosts/SpaceNix/` | Boot, drivers, audio |
| **Platform Features** | `modules/platforms/` | Laptop power, desktop gaming |
| **New Host** | Create `hosts/<name>/` | Hardware configuration |

## How to Modify Configurations

### Common Modifications

#### 🔧 Adding System Packages
```nix
# modules/general/nixos/default.nix
environment.systemPackages = with pkgs; [
  # Add your packages here
  htop
  git
  firefox
];
```

#### 🏠 Adding User Programs  
```nix
# modules/general/home/default.nix
home.packages = with pkgs; [
  # User-specific packages
  discord
  spotify
];
```

#### ⚙️ Configuring Applications
```nix
# modules/cli/home/git/default.nix
programs.git = {
  enable = true;
  userName = "Your Name";
  userEmail = "your.email@example.com";
};
```

#### 🖥️ Platform-Specific Settings
```nix
# modules/platforms/laptop/default.nix - Laptop optimizations
services.tlp.enable = true;

# modules/platforms/desktop/default.nix - Desktop features
programs.steam.enable = true;
```

#### 🏗️ Adding New Host
1. Create `hosts/<hostname>/`
2. Copy hardware-configuration.nix from `/etc/nixos/`
3. Add to `flake.nix` nixosConfigurations
4. Build with `sudo nixos-rebuild switch --flake .#<hostname>`

### Module System

Each module follows this pattern:
- `default.nix` - Main module imports
- `home/` - Home Manager configurations  
- `nixos/` - NixOS system configurations
- `options.nix` - Custom options and toggles

This structure separates user settings from system settings while keeping related configurations together.

## Features in Development

- Previewing Latex and markdown with neovim
- Enhanced cross-platform compatibility

## References

Other repositories that inspired me:

- [ryan4yin/nix-config](https://github.com/ryan4yin/nix-config/tree/main)
- [librephoenix/nixos-config (GitLab)](https://gitlab.com/librephoenix/nixos-config)
