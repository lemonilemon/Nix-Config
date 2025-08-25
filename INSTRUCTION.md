# Project Instructions for AI Assistants

## Overview

This is a **modular cross-platform NixOS configuration** project that demonstrates advanced Nix patterns for managing multiple computing environments (laptop, desktop, WSL) from a single, maintainable codebase.

## Project Philosophy

### Core Principles
1. **Cross-Platform Uniformity**: Same user experience across laptop, desktop, and WSL environments
2. **Modular Architecture**: Feature-centric organization separating concerns while maintaining cohesion  
3. **Declarative Configuration**: Everything as code - reproducible, version-controlled, and testable
4. **Safety-First Approach**: Always test changes before applying, use Nix's hash validation when possible
5. **Progressive Enhancement**: Build on working configurations rather than breaking changes

### Why This Approach Matters
- **Consistency**: Identical development environment across all devices
- **Maintainability**: Changes in one place affect all relevant platforms
- **Scalability**: Easy to add new hosts, users, or platform types
- **Knowledge Sharing**: Configuration serves as documentation of system setup

## AI Assistant Guidelines

### Available Tools and Resources

#### NixOS MCP Server
You have access to a specialized NixOS MCP server with these capabilities:
- `mcp__mcp-nixos__nixos_search` - Search NixOS packages and options
- `mcp__mcp-nixos__nixos_info` - Get detailed package/option information  
- `mcp__mcp-nixos__home_manager_search` - Search Home Manager options
- `mcp__mcp-nixos__darwin_search` - Search nix-darwin options (macOS)
- `mcp__mcp-nixos__nixos_flakes_search` - Search community flakes

**Always use these tools** when:
- User asks about NixOS packages or options
- Helping configure services or programs
- Looking up Home Manager settings
- Exploring available flakes or packages

Example usage:
```
mcp__mcp-nixos__nixos_search query:"hyprland" 
mcp__mcp-nixos__home_manager_search query:"programs.git"
```

### Working Principles

#### 1. Test Before Apply
```bash
# Always build first
sudo nixos-rebuild build --flake .#<config>

# Use justfile for convenience after initial setup
just dry-build
```

#### 2. Understand Before Modifying
- Read existing configurations to understand patterns
- Check how similar features are implemented
- Follow the established module structure

#### 3. Platform Awareness
- Use `platformConfig` parameters to make platform-specific decisions
- WSL should exclude GUI components
- Laptops need power management
- Desktops can include performance/gaming features

#### 4. Module Organization
Follow this pattern:
```
modules/<feature>/
├── default.nix      # Main imports and module definition
├── home/           # Home Manager configurations
├── nixos/          # NixOS system configurations  
└── options.nix     # Custom options and toggles
```

### Common Tasks

#### Adding Packages
1. **System packages**: `modules/general/nixos/default.nix`
2. **User packages**: `modules/general/home/default.nix`  
3. **Platform-specific**: `modules/platforms/<platform>/`

#### Configuring Programs
1. Check existing patterns in relevant module
2. Use MCP server to find correct options:
   ```
   mcp__mcp-nixos__home_manager_search query:"programs.<program>"
   ```
3. Add configuration following established structure

#### Adding New Hosts
1. Create `hosts/<hostname>/` directory
2. Include hardware-configuration.nix
3. Add host-specific settings
4. Update flake.nix nixosConfigurations

### Safety Guidelines

#### Always Do
- ✅ Test configurations with `build` before `switch`
- ✅ Use the MCP server for NixOS/Home Manager queries
- ✅ Follow established patterns and file organization  
- ✅ Consider platform differences (`platformConfig`)
- ✅ Document any new patterns or complex configurations

#### Never Do
- ❌ Apply untested configurations directly
- ❌ Break working configurations without backup approach
- ❌ Add system packages to Home Manager or vice versa
- ❌ Ignore platform-specific requirements
- ❌ Create new organizational patterns without discussion

### Project Status Reference

For current project status, implementation details, and specific architectural decisions, refer to `CLAUDE.md`. This file contains:
- Current implementation status
- Architectural patterns in use  
- Specific configuration details
- Development history and decisions

### Getting Help

When you need assistance with NixOS-specific questions:

1. **First**: Use the MCP server tools to search for packages/options
2. **Then**: Check existing configurations for patterns
3. **Finally**: Refer to official NixOS/Home Manager documentation

The MCP server provides access to:
- 132,409+ packages in nixpkgs
- 22,405+ NixOS configuration options
- Complete Home Manager option database
- Community flakes and overlays

This comprehensive toolset should handle most configuration needs without external searches.