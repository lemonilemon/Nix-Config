# General Module

This module provides fundamental system settings and user environment configurations that are common across all profiles. It handles core NixOS settings, Nix configuration, programming language toolchains, fonts, and various utilities.

## Overview

The General module serves as the foundation for system functionality:
- **NixOS Settings**: Core system configuration, Nix daemon settings, and experimental features
- **Programming Languages**: Development toolchains for multiple languages
- **Fonts**: System and user font configuration
- **Utilities**: Common tools and helpers
- **nix-ld**: Compatibility layer for running non-NixOS binaries
- **Secrets Management**: Configuration for handling sensitive data
- **PDF Tools**: PDF viewers and utilities

## Structure

```
general/
├── default.nix          # Module entry point
├── options.nix          # Feature flag definitions
├── home/                # Home Manager configurations
│   ├── default.nix      # Home Manager module entry
│   ├── options.nix      # Home-specific options
│   ├── fonts/           # Font configuration
│   ├── pdf/             # PDF tools
│   ├── programlangs/    # Programming language packages
│   ├── secrets/         # Secrets management
│   └── utils/           # Utility programs
└── nixos/               # NixOS system-level settings
    ├── default.nix      # NixOS module entry
    ├── base.nix         # Base system settings
    ├── settings.nix     # Nix daemon and flake settings
    ├── nixld.nix        # nix-ld configuration
    ├── network.nix      # NetworkManager, wireless and firmware
    ├── firewall.nix     # Firewall ports and trusted subnets
    ├── secrets.nix      # System-level sops-nix key source
    ├── syncthing.nix    # Obsidian vault sync mesh
    └── power.nix        # Governor, auto-cpufreq, powertop, thermald, UPower
```

## Feature Flags

Control General module components via options:

### Home Manager Options
```nix
home.general.enable             # Enable all general home settings (default: true if home and general enabled)
home.general.fonts.enable       # Enable font configuration (default: follows general.enable)
home.general.pdf.enable         # Enable PDF tools (default: follows general.enable)
home.general.programlangs.enable # Enable programming languages (default: follows general.enable)
home.general.secrets.enable     # Enable secrets management (default: follows general.enable)
home.general.utils.enable       # Enable utilities (default: follows general.enable)
home.general.obsidian.enable    # Enable the Obsidian vault (default: follows general.enable, off on wsl)
home.general.obsidian.vaultPath # Absolute path of the vault (default: ~/Documents/notes)

# Programming language packages (customizable list)
home.general.programlangs.packages = [ pkgs.gcc pkgs.python3 ... ];
```

### NixOS Options
```nix
nixos.general.enable            # Enable general NixOS settings (default: true if nixos and general enabled)
nixos.general.nixld.enable      # Enable nix-ld (default: follows general.enable)
nixos.general.nix.enable        # Enable Nix settings (default: follows general.enable)
```

Form-factor-derived options. Each defaults from `formFactor` (see
`profiles/README.md`) and can be overridden on its own by a profile:

```nix
nixos.general.power.enable              # off on wsl
nixos.general.power.governor            # "powersave" on laptop, "performance" otherwise
nixos.general.power.autoCpufreq.enable  # laptop only; owns the governor when on
nixos.general.power.autoCpufreq.settings
nixos.general.power.powertop.enable     # laptop only
nixos.general.power.thermald.enable     # laptop only; intended for Intel mobile platforms
nixos.general.power.upower.enable       # laptop only

nixos.general.network.enable            # off on wsl
nixos.general.network.manager.enable    # NetworkManager plus the VPN plugins
nixos.general.network.wifi.enable       # laptop only; also seeds the bar's Wi-Fi toggle
nixos.general.network.firmware.enable   # redistributable firmware (wifi, GPU, microcode)

nixos.general.firewall.enable           # off on wsl
nixos.general.firewall.allowedTCPPorts  # [ 22 80 443 ]
nixos.general.firewall.allowedUDPPorts  # [ ]
nixos.general.firewall.trustedSubnets   # IPv4 subnets accepted wholesale

nixos.general.syncthing.enable          # follows home.general.obsidian.enable; off on wsl
nixos.general.syncthing.deviceName      # "laptop" / "desktop"; picks this host's identity

home.general.obsidian.enable            # off on wsl
```

`home.general.obsidian.vaultPath` is the single source of truth for where the
vault is. The Syncthing folder, obsidian.nvim's workspace and the Obsidian
package all read it, so there is no per-consumer copy to drift. It has no
form-factor component -- only `enable` does.

`auto-cpufreq` and the static governor are mutually exclusive by construction:
`power.nix` only sets `powerManagement.cpuFreqGovernor` when
`autoCpufreq.enable` is false, because auto-cpufreq rewrites the governor at
runtime and two owners of one knob is a bug waiting to happen.

## Components

### NixOS Settings (`nixos/`)

#### `base.nix`
Core NixOS system configuration:
- Boot loader settings
- Kernel parameters
- System-wide packages
- Service configurations
- Security settings
- User account management

#### `settings.nix`
Nix daemon and package manager configuration:
- Experimental features (flakes, nix-command)
- Binary cache settings
- Trusted users and substituters
- Garbage collection settings
- Auto-optimization
- Build settings (cores, max-jobs)

**Key features**:
```nix
nix.settings = {
  experimental-features = [ "nix-command" "flakes" ];
  auto-optimise-store = true;
  # ... additional settings
};
```

#### `nixld.nix`
Configures [nix-ld](https://github.com/Mic92/nix-ld) for running non-NixOS binaries:
- Dynamic linker compatibility
- Library path configuration
- Enables running pre-compiled binaries from other distros
- Useful for proprietary software and binary distributions

**Use case**: Running AppImages, downloaded binaries, or software that expects FHS filesystem layout.

#### `network.nix`
Network-related system settings, driven by `nixos.general.network.*`:
- NetworkManager with the OpenVPN and OpenConnect plugins
- Standalone `networking.wireless` explicitly off, since NetworkManager runs
  its own supplicant and the two conflict
- Redistributable firmware
- A wpa_supplicant OpenSSL workaround that applies on every host

#### `firewall.nix`
Firewall configuration, driven by `nixos.general.firewall.*`: the allowed TCP
and UDP port lists, and `trustedSubnets`, which are accepted wholesale on the
input chain. The rules are emitted as `iptables` commands against the
`nixos-fw` chain because that is NixOS' default firewall backend; they are IPv4
only.

#### `power.nix`
Power and thermal management, driven by `nixos.general.power.*`: CPU governor,
auto-cpufreq, powertop tunings, thermald and UPower. `power-profiles-daemon` is
held off here because it conflicts with both auto-cpufreq and a static
governor.

#### `secrets.nix`
Points sops-nix at the age key for **system** secrets
(`~/.config/sops/age/keys.txt`, the same file the Home Manager side uses). It
has no feature flag of its own on purpose: it is the key source for the whole
category rather than a feature, and sops-nix's config block is gated on
`sops.secrets != {}`, so it is free on a host that declares no secrets.

#### `syncthing.nix`
Syncs the Obsidian vault between hosts, driven by `nixos.general.syncthing.*`.
The folder it syncs is `home.general.obsidian.vaultPath` -- the same option
obsidian.nvim reads -- rather than a path of its own, and
`nixos.general.syncthing.enable` follows `home.general.obsidian.enable`, since
there is nothing to sync on a host with no vault.

The device mesh — which machines hold the vault — is a literal in the module
rather than an option, because it is one global fact rather than a per-host
knob. Each host derives its peer list by removing itself from that mesh, using
`deviceName`.

The split that makes this fully declarative: **device IDs are public** (each is
a hash of the public half of a certificate) so they are committed in the
module, while the **private keys live in `secrets/syncthing.yaml`** and each
host decrypts only its own. A reinstalled host therefore restores its key from
sops and is immediately the same device to every peer — no re-pairing.

Two things are deliberate and easy to undo by accident:

- **The vault is also a git working tree** (it has a GitHub remote and
  auto-commits). Syncthing's ignore list may therefore only name paths the
  vault's own ignore rules already cover — ignoring a *tracked* file would
  reach the other machine as a deletion and get committed as one. The current
  list is per-machine UI state and regenerable caches only.
- **Version snapshots are stored outside the vault** via `versioning.fsPath`
  (`~/.local/state/syncthing/versions/`). Syncthing's default location is
  `<folder>/.stversions`, which would put a copy of every past revision inside
  an auto-committing repo.

**Sync is LAN-only.** `globalAnnounceEnabled` and `relaysEnabled` are off, so
the hosts announce their addresses to no external server and Syncthing contacts
nothing outside the local network (the nixpkgs build is tagged `noupgrade`, and
`urAccepted = -1` declines usage reporting). Local discovery, a UDP broadcast on
21027 that never leaves the network segment, is what finds peers.

That is also what makes committing the device IDs safe in a public repo. A
device ID is only a privacy problem when there is an address record to resolve
it against: global discovery's lookups are unauthenticated, so with announcing
on, anyone holding an ID could resolve it to the machine's public IP and online
status. With announcing off there is nothing to resolve.

The cost is that devices sync when they share a network. For off-LAN sync,
either turn the two options back on, or give each device a static address
alongside `"dynamic"` -- Syncthing resolves the address list per entry, so
local discovery and a fixed address (a VPN name, say) compose rather than
conflict.

The web UI stays bound to `127.0.0.1:8384`. That matters on the desktop, which
trusts `192.168.0.0/24` wholesale in its firewall: binding loopback is what
keeps that LAN bypass from also exposing the admin interface.

Adding a device: put its ID in the `mesh` attribute set. Non-NixOS devices (the
Android phone) are entered the same way, with `null` as a placeholder until the
device reports its ID; null rows are dropped rather than emitted.

### Home Manager Settings (`home/`)

#### Programming Languages (`programlangs/`)

Configures development toolchains for multiple programming languages.

**Default packages** (defined in `home/default.nix`):
```nix
home.general.programlangs.packages = [
  gcc              # C/C++ compiler
  python3          # Python with numpy
  uv               # Python package manager
  nodejs           # Node.js runtime
  bun              # JavaScript runtime
  jre8             # Java Runtime Environment
  rustc            # Rust compiler
  cargo            # Rust package manager
];
```

**Customization**:
```nix
# In your profile or module
home.general.programlangs.packages = with pkgs; [
  # Override with your preferred languages
  go
  zig
  julia
  # Or extend the defaults
] ++ config.home.general.programlangs.packages;
```

**Language-specific configuration**:
- Python includes numpy by default via `python3.withPackages`
- Can be extended with virtualenv managers (micromamba commented out)
- Package managers included where applicable (cargo, npm via nodejs, uv)

#### Fonts (`fonts/`)

System and user font configuration:
- Font packages installation
- Font discovery and caching
- Font rendering settings
- Fallback font configuration

**Common fonts included**:
- Monospace fonts for terminal/coding
- System fonts
- Icon fonts (Nerd Fonts)
- CJK fonts (if needed)

**Usage**:
```nix
fonts.packages = with pkgs; [
  nerd-fonts.jetbrains-mono
  noto-fonts
  noto-fonts-cjk
];
```

#### Obsidian (`options.nix` only)

`home.general.obsidian` has no module directory of its own: it is a fact about
the machine that three existing modules consume, rather than configuration in
its own right.

| Consumer | Reads |
|----------|-------|
| `general/nixos/syncthing.nix` | `vaultPath` as the synced folder, `enable` via the syncthing flag |
| `cli/home/nvim/plugins/code/markdown.nix` | `vaultPath` as obsidian.nvim's workspace, `enable` as the plugin flag |
| `gui/home/apps/default.nix` | `enable`, to decide whether to install the Obsidian package |

`tests/nix/host-options.nix` asserts that the first two resolve to the same
path. That is not ceremony: obsidian.nvim previously pointed at
`~/obsidian/school`, a directory that did not exist, and nothing reported it.

#### PDF Tools (`pdf/`)

PDF viewer and manipulation tools:
- PDF readers (Zathura, evince, etc.)
- PDF utilities
- Document viewing configuration

#### Secrets Management (`secrets/`)

Configuration for handling sensitive data:
- Secret encryption/decryption tools
- Credential management
- SSH/GPG integration
- Secure storage configuration

**Note**: Actual secrets should never be committed to the repository. This module configures the tools and frameworks for managing secrets.

#### Utilities (`utils/`)

Common utility programs and tools:
- System utilities
- File management tools
- Archive handlers
- Text processing tools
- Miscellaneous helpers

## Usage

### Basic Enablement

In your profile's `config.nix`:
```nix
{
  general.enable = true;  # Enable all general settings
}
```

This is enabled by default in most profiles.

### Selective Component Control

```nix
{
  general.enable = true;

  # Disable specific components
  home.general.pdf.enable = false;      # Don't install PDF tools
  nixos.general.nixld.enable = false;   # Disable nix-ld
}
```

### Customizing Programming Languages

**Replace defaults**:
```nix
{
  home.general.programlangs.packages = with pkgs; [
    # Only install these languages
    go
    lua
    ruby
  ];
}
```

**Extend defaults**:
```nix
{
  home.general.programlangs.packages = with pkgs;
    config.home.general.programlangs.packages ++ [
      # Add to existing defaults
      haskellPackages.ghc
      dotnet-sdk
    ];
}
```

**Add Python packages**:
```nix
{
  home.general.programlangs.packages = with pkgs; [
    (python3.withPackages (ps: with ps; [
      numpy
      pandas
      matplotlib
      requests
      # your packages here
    ]))
  ];
}
```

### Adding System Packages

For system-wide packages not managed by Home Manager:
```nix
# In a profile or module
environment.systemPackages = with pkgs; [
  your-package
];
```

### Adding User Packages

For user-specific packages:
```nix
# In home manager configuration
home.packages = with pkgs; [
  your-package
];
```

## Nix Configuration

### Experimental Features

This configuration enables modern Nix features:
- **Flakes**: New package and configuration format
- **nix-command**: New CLI interface (`nix build`, `nix run`, etc.)

### Binary Caches

Configured to use:
- Official NixOS cache
- Community caches (if any)
- Custom caches (can be added)

### Optimizations

- **Auto-optimization**: Automatic hard-linking of identical files in `/nix/store`
- **Garbage collection**: Regular cleanup of unused packages
- **Build optimization**: Parallel builds based on CPU cores

## Best Practices

1. **Language toolchains**: Install language-specific tools via the programming languages module rather than adding them individually
2. **Secrets**: Never commit secrets; use proper secrets management
3. **System vs User packages**: Use `environment.systemPackages` for system services, `home.packages` for user tools
4. **Font configuration**: Add fonts to Home Manager for user-specific needs, to system for display managers
5. **nix-ld usage**: Only enable if you need to run non-NixOS binaries; it's not needed for most pure Nix setups

## Troubleshooting

### Binary Not Working

If a downloaded binary doesn't work:
1. Enable `nixos.general.nixld.enable = true`
2. Rebuild the system
3. Check library dependencies with `ldd <binary>`

### Python Package Issues

For Python packages not available in nixpkgs:
```bash
# Use uv or pip in a virtual environment
uv venv
source .venv/bin/activate
uv pip install <package>
```

### Font Not Showing

1. Verify font is in `fonts.packages`
2. Rebuild Home Manager configuration
3. Clear font cache: `fc-cache -f`
4. Verify with: `fc-list | grep <font-name>`

### Nix Store Growing Large

```bash
# Manual garbage collection
nix-collect-garbage -d

# Remove old generations
nix-collect-garbage --delete-older-than 30d

# Check disk usage
du -sh /nix/store
```

## Integration with Other Modules

The General module is foundational and is typically enabled across all profiles:

- **CLI module**: Depends on general settings for shell environment
- **Desktop module**: Uses font configuration from general
- **GUI module**: Relies on nix-ld for some proprietary applications
- **All modules**: Benefit from Nix settings and optimizations

## Configuration Files

### Important Files to Customize

1. **`home/default.nix`**: Modify default programming language packages
2. **`nixos/settings.nix`**: Adjust Nix daemon settings for your needs
3. **`home/fonts/default.nix`**: Add or remove font packages
4. **`home/utils/default.nix`**: Include utility programs you commonly use

## Dependencies

- **nix-ld**: Optional, for binary compatibility
- **nixpkgs**: All packages come from nixpkgs
- **home-manager**: User environment management

## Notes

- This module is designed to be enabled by default
- Most settings are sensible defaults that work across all profiles
- Customizations should typically happen at the profile level
- The module is structured to allow granular control via feature flags while maintaining good defaults
