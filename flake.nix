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
      pkgs = import ./config {
        inherit inputs;
        inherit system;
      };

      hostname = "SpaceNix";

      # Platform configuration factory
      mkSystem =
        {
          platform,
          hostDir,
          hostOverride ? null,
          system ? "x86_64-linux",
        }:
        let
          hostName = if hostOverride != null then hostOverride else hostname;
          platformConfig = {
            type = platform;
            isWSL = platform == "wsl";
            isLaptop = platform == "laptop";
            isDesktop = platform == "desktop";
          };
        in
        nixpkgs.lib.nixosSystem {
          inherit system;
          specialArgs = {
            inherit inputs username platformConfig;
            hostname = hostName;
          };
          modules = [
            # Core module structure
            ./modules
            ./overlays

            # Host-specific configuration
            ./hosts/${hostDir}

            # Platform detection and common settings
            ./modules/common

            # Platform-specific modules
            ./modules/platforms/${platform}

            # Third-party modules
            catppuccin.nixosModules.catppuccin
            grub2-themes.nixosModules.default
          ]
          ++ nixpkgs.lib.optionals (platform == "wsl") [
            nixos-wsl.nixosModules.wsl
          ]
          ++ [
            # Home Manager integration - use original working approach
            home-manager.nixosModules.home-manager
            {
              home-manager = {
                useGlobalPkgs = true;
                useUserPackages = true;
                extraSpecialArgs = {
                  inherit
                    inputs
                    username
                    platformConfig
                    system
                    ;
                  sys = if platform == "wsl" then "NixOS-wsl" else platform; # Map platform to sys for backward compatibility
                  hostname = hostName;
                };
                users.${username} = {
                  imports = [
                    ./modules/home.nix
                    catppuccin.homeModules.catppuccin
                    nix-index-database.homeModules.nix-index
                  ];
                };
                sharedModules = [ nixvim.homeModules.nixvim ];
              };
            }
          ];
        };
    in
    {
      legacyPackages.${system} = pkgs;

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
          nixvim.homeModules.nixvim
          catppuccin.homeModules.catppuccin
          nix-index-database.homeModules.nix-index
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
                      nix-index-database.homeModules.nix-index
                    ];
                  };

                  home-manager.sharedModules = [ nixvim.homeModules.nixvim ];
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
