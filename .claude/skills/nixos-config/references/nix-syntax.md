# Nix Language Syntax Reference

A quick reference for patterns commonly needed when editing lemonilemon/Nix-Config.

---

## Nix Expressions

### Let-in
```nix
let
  x = 1;
  y = x + 2;
in y  # evaluates to 3
```

### Attribute sets (records)
```nix
{
  key = "value";
  nested.key = "also valid";
  "key with spaces" = true;
}
```

### With expression
```nix
with pkgs; [ git vim curl ]   # equivalent to [ pkgs.git pkgs.vim pkgs.curl ]
```

### Inherit
```nix
let name = "alice"; in
{ inherit name; }   # same as { name = name; }

# Inherit from attrset
{ inherit (config.my) option; }  # same as { option = config.my.option; }
```

### Function syntax
```nix
# Single argument
x: x + 1

# Attribute set destructuring (most common in modules)
{ pkgs, lib, config, ... }: {
  # module body
}

# With default
{ name ? "default", ... }: name
```

### String interpolation
```nix
"Hello ${name}!"
''
  Multi-line
  string with ${interpolation}
''
```

### Lists
```nix
[ 1 2 3 ]
[ pkgs.git pkgs.vim ]
list1 ++ list2   # concatenate
```

### Conditionals
```nix
if condition then valueA else valueB
```

### Recursive attribute sets
```nix
rec {
  x = 1;
  y = x + 1;  # can reference x
}
```

---

## NixOS Module System

### Option types
```nix
lib.types.bool
lib.types.str
lib.types.int
lib.types.float
lib.types.path
lib.types.package
lib.types.listOf lib.types.str
lib.types.listOf lib.types.package
lib.types.attrsOf lib.types.str
lib.types.nullOr lib.types.str
lib.types.enum [ "a" "b" "c" ]
lib.types.submodule { options = { ... }; }
```

### mkOption patterns
```nix
lib.mkOption {
  type = lib.types.bool;
  default = false;
  description = "Whether to enable X.";
  example = true;
}

# Shorthand for boolean enables:
lib.mkEnableOption "X"
# Equivalent to mkOption with type = bool, default = false
```

### Priority / override
```nix
lib.mkDefault value    # lower priority (can be overridden by profile)
lib.mkForce value      # higher priority (overrides everything)
lib.mkOverride 900 value  # explicit priority number (1000 = default)
```

### mkMerge and mkIf
```nix
config = lib.mkMerge [
  (lib.mkIf cfg.featureA.enable {
    environment.systemPackages = [ pkgs.a ];
  })
  (lib.mkIf cfg.featureB.enable {
    services.b.enable = true;
  })
];
```

---

## Home Manager Specifics

### Common top-level options
```nix
home.username = "lemonilemon";
home.homeDirectory = "/home/lemonilemon";
home.stateVersion = "24.05";  # do not change after initial setup
home.packages = with pkgs; [ ... ];

# Session variables
home.sessionVariables = {
  EDITOR = "nvim";
  FOO = "bar";
};

# Shell aliases (also available per-shell under programs.zsh.shellAliases)
home.shellAliases = {
  ll = "ls -la";
};
```

### XDG config files
```nix
# Write an arbitrary config file
xdg.configFile."myapp/config.toml".text = ''
  [section]
  key = "value"
'';

# Or source from a file in the repo
xdg.configFile."myapp/config.toml".source = ./myapp-config.toml;
```

### programs.* pattern
```nix
programs.zsh = {
  enable = true;
  autosuggestion.enable = true;
  syntaxHighlighting.enable = true;
  initExtra = ''
    # extra zsh config
  '';
  shellAliases = {
    ll = "ls -la";
  };
};
```

---

## Flake Patterns

### follows — pin a transitive input
```nix
inputs.my-tool = {
  url = "github:owner/my-tool";
  inputs.nixpkgs.follows = "nixpkgs";  # use our nixpkgs, not theirs
};
```

### Non-flake source (flake = false)
```nix
inputs.some-plugin = {
  url = "github:owner/repo/abc123";
  flake = false;  # raw source, no outputs evaluated
};
# Use as: pkgs.vimUtils.buildVimPlugin { src = inputs.some-plugin; ... }
```

### Accessing a flake's packages in a module
```nix
{ inputs, pkgs, ... }:
{
  environment.systemPackages = [
    inputs.some-tool.packages.${pkgs.system}.default
  ];
}
```

---

## Overlay Patterns

### In overlays/
```nix
# overlays/my-overlay.nix
final: prev: {
  my-package = prev.my-package.override {
    someFlag = true;
  };

  # Or a completely custom package:
  my-custom = prev.callPackage ./my-custom-pkg { };
}
```

### Applying overlays (nixpkgs/ handles this)
The `nixpkgs/` directory applies overlays from `overlays/` — follow whatever pattern
is already set up there when adding a new overlay file.

---

## Nixvim (Neovim config)

Nixvim options live under `programs.nixvim.*` in Home Manager modules.
Full option reference: https://nix-community.github.io/nixvim/

```nix
programs.nixvim = {
  enable = true;
  plugins.telescope.enable = true;
  plugins.lsp = {
    enable = true;
    servers.ts_ls.enable = true;
  };
  extraConfigLua = ''
    -- raw lua config
  '';
};
```
