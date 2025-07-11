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
      # IMPORTANT: we're using "libgbm" and is only available in unstable so ensure
      # to have it up to date or simply don't specify the nixpkgs input
      inputs.nixpkgs.follows = "nixpkgs";
    };

  };
  outputs =
    inputs@{
      self,
      systems,
      nixpkgs,
      nixos-wsl,
      nixos-hardware,
      nix-index-database,
      home-manager,
      grub2-themes,
      nixvim,
      catppuccin,
      ...
    }:
    let
      eachSystem = nixpkgs.lib.genAttrs (import systems);
      username = "lemonilemon";
      system = "x86_64-linux";
      pkgs = nixpkgs.legacyPackages.${system};
    in
    {
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
      homeConfigurations.${username} = home-manager.lib.homeManagerConfiguration {
        inherit pkgs;

        modules = [
          ./overlays
          ./home-manager
          nixvim.homeManagerModules.nixvim
          catppuccin.homeModules.catppuccin
          nix-index-database.hmModules.nix-index
        ];

        extraSpecialArgs = {
          inherit inputs username system;
          sys = "hm";
        };
      };

      # For NixOS
      nixosConfigurations =
        let
          hostname = "SpaceNix";
          make_nixosConfig =
            {
              sys ? "default",
              profile,
              specialArgs ? { },
            }:
            nixpkgs.lib.nixosSystem {
              system = "x86_64-linux";
              specialArgs = {
                inherit
                  inputs
                  username
                  hostname
                  specialArgs
                  system
                  ;
              };
              modules = [
                ./modules
                ./overlays

                # nixos-wsl
                nixos-wsl.nixosModules.wsl
                grub2-themes.nixosModules.default

                catppuccin.nixosModules.catppuccin

                # home-manager
                home-manager.nixosModules.home-manager
                {
                  home-manager.useGlobalPkgs = true;
                  home-manager.useUserPackages = true;
                  home-manager.extraSpecialArgs = {
                    inherit
                      inputs
                      username
                      hostname
                      sys
                      specialArgs
                      system
                      ;
                  };
                  home-manager.users.${username} = {
                    imports = [
                      ./modules/home.nix
                      catppuccin.homeModules.catppuccin
                      nix-index-database.hmModules.nix-index
                    ];
                  };

                  home-manager.sharedModules = [ nixvim.homeManagerModules.nixvim ];
                }
                # profile settings
                profile
              ];

            };
        in
        {
          NixOS-wsl = make_nixosConfig {
            sys = "NixOS-wsl";
            profile = ./profiles/wsl;
          };

          laptop = make_nixosConfig {
            sys = "laptop";
            profile = ./profiles/laptop;
          };

          desktop = make_nixosConfig {
            sys = "desktop";
            profile = ./profiles/desktop;
          };
        };
    };
}
