{
  description = "Lemonilemon's Nix Flake";

  inputs = {
    nixpkgs.url = "github:NixOS/nixpkgs/nixos-unstable";
    nixos-hardware.url = "github:NixOS/nixos-hardware/master";
    nixos-wsl = {
      url = "github:nix-community/NixOS-WSL";
      inputs.nixpkgs.follows = "nixpkgs";
    };
    nixvim = {
      url = "github:nix-community/nixvim";
    };
    home-manager = {
      url = "github:nix-community/home-manager";
      inputs.nixpkgs.follows = "nixpkgs";
    };
    nix-index-database = {
      url = "github:nix-community/nix-index-database";
      inputs.nixpkgs.follows = "nixpkgs";
    };
    systems.url = "github:nix-systems/default";
    flake-utils = {
      url = "github:numtide/flake-utils";
      inputs.systems.follows = "systems";
    };

    pre-commit-hooks.url = "github:cachix/git-hooks.nix";
    hyprland.url = "github:hyprwm/Hyprland";
    catppuccin.url = "github:catppuccin/nix";
    grub2-themes.url = "github:vinceliuice/grub2-themes";
    rose-pine-hyprcursor = {
      url = "github:ndom91/rose-pine-hyprcursor";
      inputs.nixpkgs.follows = "nixpkgs";
      inputs.hyprlang.follows = "hyprland/hyprlang";
    };
    zen-browser = {
      url = "github:0xc000022070/zen-browser-flake";
      inputs.nixpkgs.follows = "nixpkgs";
    };
    claude-desktop = {
      url = "github:k3d3/claude-desktop-linux-flake";
      inputs.nixpkgs.follows = "nixpkgs";
      inputs.flake-utils.follows = "flake-utils";
    };
    template-nvim = {
      url = "github:nvimdev/template.nvim/308f6f8f0bf98cb7c71855ffa8a3019a5642d1cd";
      flake = false;
    };
    coderunner-nvim = {
      url = "github:CRAG666/code_runner.nvim/45dfea066a6110abcbc3cd361457ac3cbaefd68b";
      flake = false;
    };
    copilot-lualine-nvim = {
      url = "github:AndreM222/copilot-lualine/222e90bd8dcdf16ca1efc4e784416afb5f011c31";
      flake = false;
    };
  };
  outputs =
    inputs@{
      self,
      systems,
      nixpkgs,
      ...
    }:
    let
      eachSystem = nixpkgs.lib.genAttrs (import systems);
      helpers = (import ./lib) { inherit inputs; };

      username = "lemonilemon";
      hostname = "SpaceNix";

    in
    {
      legacyPackages = eachSystem (
        system:
        (import ./nixpkgs {
          inherit inputs system;
        })
      );

      formatter = eachSystem (
        system:
        let
          pkgs = nixpkgs.legacyPackages.${system};
        in
        pkgs.writeShellScriptBin "parallel-nixfmt" ''
          find . \( -name "*.nix" -not -path "./result*" -not -path "./.direnv/*" \) \
            -print0 | \
            ${pkgs.parallel}/bin/parallel --no-notice -0 -j$(nproc) ${pkgs.nixfmt}/bin/nixfmt
        ''
      );

      checks = eachSystem (system: {
        pre-commit-check = inputs.pre-commit-hooks.lib.${system}.run {
          src = ./.;
          hooks = {
            nixfmt.enable = true;
          };
        };
      });

      devShells = eachSystem (system: {
        default = nixpkgs.legacyPackages.${system}.mkShell {
          inherit (self.checks.${system}.pre-commit-check) shellHook;

          buildInputs = with nixpkgs.legacyPackages.${system}; [
            nixfmt
          ];
        };
      });

      # For NixOS
      nixosConfigurations = {
        NixOS-wsl = helpers.mkSystem {
          system = "x86_64-linux";
          inherit username hostname;
          profile = "wsl";
          isWSL = true;
        };

        laptop = helpers.mkSystem {
          system = "x86_64-linux";
          inherit username hostname;
          profile = "laptop";
        };

        desktop = helpers.mkSystem {
          system = "x86_64-linux";
          inherit username hostname;
          profile = "desktop";
        };
      };
    };
}
