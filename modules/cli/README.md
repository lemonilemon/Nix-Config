# CLI Module

This module provides comprehensive command-line interface tools and configurations, including shell setup, terminal multiplexer, version control, and an extensively configured Neovim editor.

## Overview

The CLI module is designed to provide a complete terminal-based development environment with:
- **Neovim**: Full IDE-like experience with LSP, completion, AI integration, and extensive plugins
- **Shells**: zsh with starship prompt and zoxide navigation
- **Git**: Version control configuration
- **Zellij**: Terminal multiplexer for workspace management
- **CLI Programs**: System utilities, AI tools, and productivity enhancers

## Structure

```
cli/
├── default.nix      # Module entry point
├── options.nix      # Feature flag definitions
├── home/            # Home Manager configurations
│   ├── default.nix  # Home Manager module entry
│   ├── options.nix  # Home-specific option definitions
│   ├── git/         # Git configuration
│   ├── nvim/        # Neovim configuration (nixvim)
│   │   ├── options/ # Editor settings and keymaps
│   │   └── plugins/ # Plugin configurations
│   ├── programs/    # CLI utilities
│   │   └── ai/      # AI coding agents (claude-code, codex, opencode, ...) + AGENTS.md
│   ├── shells/      # Shell configurations (zsh, starship, zoxide)
│   └── zellij/      # Terminal multiplexer config
└── nixos/           # NixOS system-level CLI settings
```

## Feature Flags

Control CLI components via options defined in `options.nix`:

```nix
# Top-level
home.cli.enable          # Enable entire CLI module (default: true if home and cli enabled)

# Git
home.cli.git.enable      # Enable git configuration (default: follows cli.enable)

# Neovim
home.cli.nvim.enable     # Enable Neovim (default: follows cli.enable)

# Programs
home.cli.programs.enable # Enable CLI utilities (default: follows cli.enable)

# Shells
home.cli.shells.enable   # Enable shell configuration (default: follows cli.enable)
home.cli.shells.zsh.enable      # Enable zsh
home.cli.shells.starship.enable # Enable starship prompt
home.cli.shells.zoxide.enable   # Enable zoxide directory jumper

# Zellij
home.cli.zellij.enable   # Enable zellij terminal multiplexer (default: follows cli.enable)
```

## Components

### Git (`home/git/`)

Version control configuration including:
- User identity and email
- Default branch settings
- Merge and diff tools
- Aliases and shortcuts
- Git LFS support (if needed)

**Usage**: Configuration is automatically applied when `home.cli.git.enable = true`

### Neovim (`home/nvim/`)

Extensive Neovim configuration using [nixvim](https://github.com/nix-community/nixvim).

#### Structure

**`options/`**: Core editor configuration
- `opts.nix`: Editor options (line numbers, tabs, search, etc.)
- `keymaps.nix`: Custom key bindings
- `colorschemes.nix`: Color scheme configuration (Catppuccin)

**`plugins/`**: Organized by functionality

```
plugins/
├── ai/                 # AI assistants
│   ├── avante.nix      # Avante AI
│   └── CopilotChat.nix # GitHub Copilot Chat
├── code/               # Code editing tools
│   ├── competitest.nix # Competitive programming
│   ├── coderunner.nix  # Code execution
│   ├── jupytext.nix    # Jupyter notebooks as plain text
│   ├── markdown.nix    # Markdown support
│   ├── template.nix    # File templates
│   ├── typst.nix       # Typst document system
│   └── vimtex.nix      # LaTeX editing
├── completion/         # Code completion
│   ├── cmp.nix         # nvim-cmp completion engine
│   └── lspkind.nix     # LSP kind icons
├── files/              # File navigation
│   ├── neotree.nix     # File tree explorer
│   └── telescope.nix   # Fuzzy finder
├── formatting/         # Code formatting
│   └── conform.nix     # Formatter integration
├── git/                # Git integration
│   ├── gitsigns.nix    # Git decorations
│   └── lazygit.nix     # Git TUI
├── lsp/                # Language Server Protocol
│   ├── lsp.nix         # LSP configuration
│   └── trouble.nix     # Diagnostics UI
├── snippets/           # Code snippets
│   └── luasnip.nix     # Snippet engine
├── treesitter/         # Syntax parsing
│   └── treesitter.nix  # Treesitter config
├── ui/                 # User interface
│   ├── bufferline.nix  # Buffer/tab line
│   ├── lualine.nix     # Status line
│   ├── noice.nix       # UI improvements
│   └── notify.nix      # Notification system
└── utils/              # Utilities
    ├── colorizer.nix   # Color highlighting
    ├── comment.nix     # Commenting
    ├── comment-box.nix # Comment boxes
    ├── edgy.nix        # Window layout
    ├── flash.nix       # Navigation
    ├── hardtime.nix    # Vim training
    ├── snacks.nix      # Utility collection
    ├── transparent.nix # Transparent background
    ├── whichkey.nix    # Keybinding hints
    └── zellij-nav.nix  # Zellij integration
```

#### Key Features

**LSP Support**: Language Server Protocol integration for:
- Code completion
- Go-to-definition
- Hover documentation
- Diagnostics
- Code actions

**AI Integration**:
- GitHub Copilot for code suggestions
- Avante AI for advanced assistance
- CopilotChat for conversational coding help

**Competitive Programming**:
- CompetiTest for running test cases
- Code Runner for quick execution

**Document Editing**:
- VimTeX for LaTeX
- Typst support
- Markdown with preview capabilities
- Jupyter notebooks (`.ipynb`) edited as plain text via jupytext

**Navigation**:
- Telescope fuzzy finder (files, grep, buffers, etc.)
- Neo-tree file explorer
- Flash for quick movement
- Zoxide integration

#### Customization

To add a new plugin:

1. Create `plugins/<category>/<plugin-name>.nix`:
   ```nix
   { config, lib, ... }:
   {
     config = lib.mkIf config.home.cli.nvim.enable {
       programs.nixvim.plugins.<plugin> = {
         enable = true;
         # plugin configuration
       };
     };
   }
   ```

2. Import in `nvim/default.nix`

To modify keymaps, edit `options/keymaps.nix`.

### Shells (`home/shells/`)

Shell environment configuration:

**zsh** (`zsh.nix`):
- Interactive shell with extensive customization
- Plugin management (syntax highlighting, autosuggestions)
- History configuration
- Completion system
- Shell aliases and functions

**starship** (`starship.nix`):
- Fast, customizable prompt
- Git status integration
- Language version indicators
- Directory truncation
- Execution time display

**zoxide** (`zoxide.nix`):
- Smart directory jumping
- Learning-based navigation
- Replaces `cd` with smarter alternatives

### Programs (`home/programs/`)

CLI utility configurations:

- **ai/**: AI tools — `claude-code`, `codex`, `antigravity-cli` (`agy`), `pi`, `skills`, `ccusage`, `kiro-cli`, and `opencode` with a custom multi-agent setup (team-lead, product-manager, developer, code-reviewer). Also installs `socat` and `bubblewrap` for sandboxing.
- **ai/ herdr**: A terminal multiplexer that runs several agents side by side and shows each pane's state. `~/.config/herdr/config.toml` is generated here so herdr nests cleanly inside tmux: prefix `ctrl+g` (tmux keeps `ctrl+b`, nvim's cmp keeps `ctrl+space`), and the tab row is hidden while a workspace has a single tab so tmux keeps the top row and herdr's chrome stays on the left edge. Only the file is symlinked, not the directory, since herdr writes its log and session state alongside it. Its zsh completion is generated at build time too, since herdr prints the script instead of shipping one.
- **ai/AGENTS.md**: The single global agent instruction file, symlinked to `~/AGENTS.md`, `~/.claude/CLAUDE.md`, `~/.codex/AGENTS.md`, `~/.pi/agent/AGENTS.md`, `~/.kiro/steering/00-global.md`, and `~/.config/opencode/AGENTS.md`, so every agent reads identical content and cannot drift. The link targets the Nix store, so it is read-only — agents cannot append to it in place.
- **ai/skills/my-voice/**: Personal writing-voice skill (tone rules, zh-TW conventions, AI-tell ban list, sample excerpts) for text published under lemonilemon's name. Versioned in the repo rather than installed by the `skills` CLI, and symlinked into both `~/.agents/skills/my-voice` (pi's native skill root, shared home of CLI-installed skills) and `~/.claude/skills/my-voice` (claude-code). Repo-managed skills are declared in the `nixManagedSkills` attrset in `ai/default.nix`; one line per skill generates both links. Its `references/samples.md` grows over time: agents propose excerpt additions as repo edits and a rebuild installs them.
- **atuin.nix**: Shell history sync and search (alternative to ctrl+r)
- **fastfetch.nix**: System information display
- **yazi.nix**: Terminal file manager
- **utils.nix**: General CLI utilities

### Zellij (`home/zellij/`)

Terminal multiplexer configuration:

- Workspace management
- Tab and pane layouts
- Plugin integration
- Neovim integration for seamless navigation
- Session management

**Key features**:
- Alternative to tmux with modern defaults
- Built-in layouts
- Plugin system
- Floating panes
- Session resurrection

## Usage

### Enabling CLI Module

In your profile's `config.nix`:

```nix
{
  cli.enable = true;  # Enable all CLI components
}
```

### Selective Component Control

```nix
{
  cli.enable = true;

  # Disable specific components
  home.cli.zellij.enable = false;  # Use tmux instead
  home.cli.nvim.enable = false;    # Use different editor
}
```

### Adding Custom Packages

Add CLI tools in your module or profile:

```nix
{
  home.packages = with pkgs; [
    ripgrep
    fd
    bat
    # your custom CLI tools
  ];
}
```

## NixOS Integration

The `nixos/` directory contains system-level CLI configurations that require root privileges or system-wide settings.

## Neovim Quick Reference

### Default Keymaps

Key bindings are defined in `nvim/options/keymaps.nix`. Common patterns:
- Leader key for custom commands
- File navigation via Telescope
- LSP bindings for code intelligence
- Git integration via lazygit

### Plugin Management

All plugins are managed declaratively through nixvim:
- No manual `:PackerSync` or similar commands needed
- Configuration changes require rebuilding the system/home-manager
- Plugins are downloaded and built by Nix

### LSP Configuration

LSP servers are configured in `plugins/lsp/lsp.nix`. To add a language:

1. Add the LSP server to `programs.nixvim.plugins.lsp.servers`
2. Optionally add formatters to `plugins/formatting/conform.nix`
3. Rebuild to apply changes

## Best Practices

1. **Keep plugins organized**: Place plugins in appropriate category subdirectories
2. **Use feature flags**: Leverage options for conditional configuration
3. **Document custom keymaps**: Comment complex key bindings in `keymaps.nix`
4. **Test before committing**: Always test Neovim config changes before committing
5. **Modular configuration**: Keep plugin configs in separate files for maintainability

## Troubleshooting

### Neovim Issues

```bash
# Check nixvim configuration
nix eval .#nixosConfigurations.<profile>.config.home-manager.users.<username>.programs.nixvim

# Test Neovim in isolation
nix run .#nixosConfigurations.<profile>.config.home-manager.users.<username>.programs.nixvim.package
```

### Shell Not Loading

Ensure zsh is set as default shell:
```bash
echo $SHELL  # Should show path to zsh
chsh -s $(which zsh)
```

### Plugin Not Loading

1. Check the plugin is enabled via feature flag
2. Verify import in `nvim/default.nix`
3. Check for syntax errors in plugin configuration
4. Rebuild and restart Neovim

## Dependencies

External flake inputs used by CLI module:
- `nixvim`: Neovim configuration framework
- `catppuccin`: Color scheme
- Custom plugin sources (template.nvim, coderunner.nvim, etc.) defined in `flake.nix`
