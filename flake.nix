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
    systems.url = "github:nix-systems/default";
    pre-commit-hooks.url = "github:cachix/git-hooks.nix";
  };
  outputs =
    inputs@{
      self,
      nixpkgs,
      nixos-wsl,
      nixos-hardware,
      nixvim,
      home-manager,
      systems,
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
          ./home-manager
          nixvim.homeManagerModules.nixvim
        ];

        extraSpecialArgs = {
          inherit inputs username;
          sys = "hm";
        };
      };

      # For NixOS
      nixosConfigurations =
        let
          hostname = "SpaceNix";
        in
        {
          NixOS-wsl =
            let
              sys = "wsl";
            in
            nixpkgs.lib.nixosSystem {
              system = "x86_64-linux";
              specialArgs = {
                inherit inputs username hostname;
              };
              modules = [
                ./modules/nixos
                # nixos-wsl
                nixos-wsl.nixosModules.wsl

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
                      ;
                  };
                  home-manager.users.${username} = import ./modules/home-manager;
                  home-manager.sharedModules = [ nixvim.homeManagerModules.nixvim ];
                }
                # profile settings
                ./profiles/wsl
                ./profiles/base.nix
              ];
            };
        };
    };
}
