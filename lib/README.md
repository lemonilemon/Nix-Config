# Library Functions

This directory contains helper functions and builders used throughout the NixOS configuration. These functions abstract common patterns and simplify system configuration.

## Structure

```
lib/
├── default.nix    # Main entry point, exports all helper functions
└── builders.nix   # System and Home Manager configuration builders
```

## Available Functions

### `mkSystem`

Builds a complete NixOS system configuration with Home Manager integration.

**Location**: `builders.nix`

**Parameters**:
```nix
{
  system        # Target system architecture (e.g., "x86_64-linux")
  username      # Primary user name
  hostname      # System hostname
  profile       # Profile name from profiles/ directory
  isWSL         # Enable WSL-specific modules (default: false)
  isDarwin      # Enable Darwin/macOS modules (default: false)
  extraModules  # Additional modules to include (default: [])
}
```

**Usage Example**:
```nix
# In flake.nix
nixosConfigurations = {
  my-laptop = helpers.mkSystem {
    system = "x86_64-linux";
    username = "myuser";
    hostname = "my-laptop";
    profile = "laptop";
  };

  my-wsl = helpers.mkSystem {
    system = "x86_64-linux";
    username = "myuser";
    hostname = "my-wsl";
    profile = "wsl";
    isWSL = true;
  };
};
```

**What it does**:
1. Creates a NixOS system configuration using `lib.nixosSystem`
2. Integrates Home Manager as a NixOS module
3. Configures Home Manager with:
   - Global package usage via `useGlobalPkgs`
   - System-wide package installation disabled by default
   - Backup file extension for conflicts
4. Imports core modules:
   - Main module system from `../modules`
   - Package overlays from `../overlays`
   - Catppuccin theming for NixOS and Home Manager
   - nixvim for Neovim configuration
   - nix-index-database for command-not-found
5. Loads profile-specific configuration from `profiles/<profile>/`
6. Conditionally includes:
   - NixOS-WSL module when `isWSL = true`
   - GRUB themes for standalone systems (not WSL)
   - Any additional modules from `extraModules`

**Special Args**:
The function passes the following arguments to all modules:
- `inputs` - All flake inputs
- `username` - User name
- `hostname` - System hostname
- `system` - Target architecture
- `isWSL` - WSL flag
- Plus any additional args provided (except `system` and `extraModules`)

### `mkHome`

Builds a standalone Home Manager configuration for non-NixOS systems.

**Location**: `builders.nix`

**Parameters**:
```nix
{
  system        # Target system architecture
  username      # User name
  pkgs          # Nixpkgs instance
  isWSL         # WSL flag (default: false)
  extraModules  # Additional modules (default: [])
}
```

**Usage Example**:
```nix
# For standalone Home Manager on non-NixOS
homeConfigurations = {
  "myuser@debian" = helpers.mkHome {
    system = "x86_64-linux";
    username = "myuser";
    pkgs = nixpkgs.legacyPackages.x86_64-linux;
  };
};
```

**What it does**:
1. Creates a Home Manager configuration
2. Imports:
   - Home Manager modules from `../home-manager`
   - nixvim for Neovim
   - Catppuccin theming
   - nix-index-database
   - Any additional modules from `extraModules`
3. Provides special args (`inputs`, `username`, `system`)

## Design Patterns

### Module Integration

The builders automatically integrate commonly used external modules:
- **Catppuccin**: Consistent theming across applications
- **nixvim**: Declarative Neovim configuration
- **nix-index-database**: Fast command lookup
- **Home Manager**: User environment management

This eliminates repetition across different system configurations.

### Profile-Based Configuration

The `mkSystem` function uses the profile system to load machine-specific configurations:

```nix
profile = "laptop"
# Loads: ../profiles/laptop/default.nix
```

This allows separating:
- **Common configuration**: In `modules/`
- **Machine-specific settings**: In `profiles/`

### Conditional Module Loading

The builders use conditional logic to load platform-specific modules:

```nix
# WSL-specific
++ lib.optionals isWSL [
  inputs.nixos-wsl.nixosModules.wsl
]

# Standalone (non-WSL, non-Darwin)
++ lib.optionals isStandalone [
  inputs.grub2-themes.nixosModules.default
]
```

## Extending the Library

To add new helper functions:

1. Create a new `.nix` file in `lib/` (e.g., `utils.nix`)
2. Define your functions:
   ```nix
   { inputs }:
   {
     myHelper = args: {
       # implementation
     };
   }
   ```
3. Import and export in `default.nix`:
   ```nix
   let
     builders = import ./builders.nix { inherit inputs; };
     utils = import ./utils.nix { inherit inputs; };
   in
   {
     inherit (builders) mkSystem mkHome;
     inherit (utils) myHelper;
   }
   ```

## Notes

- All builder functions receive `inputs` from the flake
- The builders assume a specific directory structure (`modules/`, `profiles/`, etc.)
- Home Manager is always integrated via `backupFileExtension = "backup"` to prevent conflicts
- `useUserPackages = false` by default means packages are installed system-wide when possible
