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
      inputs.nixpkgs.follows = "nixpkgs";
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
  };
  outputs =
    inputs@{
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

      formatter = eachSystem (system: nixpkgs.legacyPackages.${system}.nixfmt-rfc-style);

      checks = eachSystem (system: {
        pre-commit-check = inputs.pre-commit-hooks.lib.${system}.run {
          src = ./.;
          hooks = {
            nixfmt-rfc-style.enable = true;
          };
        };
      });

      # For non-NixOS
      # homeConfigurations.${username} = home-manager.lib.homeManagerConfiguration {
      #
      #   modules = [
      #     ./overlays
      #     ./home-manager
      #     nixvim.homeModules.nixvim
      #     catppuccin.homeModules.catppuccin
      #     nix-index-database.homeModules.nix-index
      #   ];
      #
      #   extraSpecialArgs = {
      #     inherit inputs username;
      #     sys = "hm";
      #   };
      # };
      #
      # For NixOS
      nixosConfigurations = {
        NixOS-wsl = helpers.mkSystem {
          system = "x86_64-linux";
          inherit username hostname;
          profile = "wsl";
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
