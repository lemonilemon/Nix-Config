{ inputs }:
let
  inherit (inputs.nixpkgs) lib;
in
{
  # NixOS system builder
  mkSystem =
    {
      system,
      username,
      hostname,
      profile,
      isWSL ? false,
      isDarwin ? false,
      extraModules ? [ ],
      ...
    }@args:
    let
      isStandalone = !isWSL && !isDarwin;
    in
    lib.nixosSystem {
      inherit system;
      specialArgs = {
        inherit inputs username hostname;
      }
      // (builtins.removeAttrs args [
        "system"
        "extraModules"
      ]);

      modules = [
        # Core modules
        ../modules
        ../overlays

        # External modules
        inputs.catppuccin.nixosModules.catppuccin

        # Home Manager integration
        inputs.home-manager.nixosModules.home-manager
        {
          home-manager = {
            useGlobalPkgs = lib.mkDefault true;
            useUserPackages = lib.mkDefault false; # default to use system-wide packages
            extraSpecialArgs = {
              inherit
                inputs
                username
                hostname
                system
                isWSL
                ;
            };
            users.${username} = {
              imports = [
                ../modules/home.nix
                inputs.catppuccin.homeModules.catppuccin
                inputs.nix-index-database.homeModules.nix-index
                inputs.nixvim.homeModules.nixvim
              ];
            };
          };
        }

        # Profile-specific configuration
        (../profiles + "/${profile}")
      ]
      ++ lib.optionals isWSL [
        inputs.nixos-wsl.nixosModules.wsl
      ]
      ++ lib.optionals isStandalone [
        inputs.grub2-themes.nixosModules.default
      ]
      ++ extraModules;
    };

  # Standalone Home Manager configuration builder
  mkHome =
    {
      system,
      username,
      pkgs,
      isWSL ? false,
      extraModules ? [ ],
    }:
    inputs.home-manager.lib.homeManagerConfiguration {
      inherit pkgs;
      modules = [
        ../home-manager
        inputs.nixvim.homeModules.nixvim
        inputs.catppuccin.homeModules.catppuccin
        inputs.nix-index-database.homeModules.nix-index
      ]
      ++ extraModules;

      extraSpecialArgs = {
        inherit
          inputs
          username
          system
          ;
      };
    };
}
