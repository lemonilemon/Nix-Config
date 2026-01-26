{ inputs }:
let
  inherit (inputs.nixpkgs) lib;

  # Helper for Home Manager options that mirror NixOS options
  mkHomeOpt =
    {
      osConfig ? null,
      path,
      default,
      description,
      type ? lib.types.bool,
      ...
    }:
    let
      # Automatically split the dot-string into a list
      pathList = lib.splitString "." path;
    in
    lib.mkOption {
      inherit type description;
      # Logic:
      # 1. If osConfig is available (NixOS), try to find the path in it.
      # 2. If found, use that value.
      # 3. If NOT found (or if on standalone Home Manager), fall back to 'default'.
      default = if osConfig != null then lib.attrByPath pathList default osConfig else default;
    };
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
            backupFileExtension = "backup";
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
              helpers = {
                inherit mkHomeOpt;
              };
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
        helpers = {
          inherit mkHomeOpt;
        };
      };
    };

  inherit mkHomeOpt;
}
