{
  description = "A generic project template";

  inputs = {
    nixpkgs.url = "github:NixOS/nixpkgs/nixos-unstable";
    flake-utils.url = "github:numtide/flake-utils";
  };

  outputs = { self, nixpkgs, flake-utils, ... }:
    flake-utils.lib.eachDefaultSystem (system:
      let
        pkgs = import nixpkgs {
          inherit system;
          config.allowUnfree = true;
        };
      in
      {
        devShells.default = pkgs.mkShell {
          buildInputs = with pkgs; [
            # Add project-specific tools here, e.g.:
            # python3
            # nodejs
            # go
          ];
          shellHook = ''
            echo "Welcome to the generic project development environment!"
            echo "Add your tools to the buildInputs list in flake.nix as needed."
          '';
        };

        # Define a default package if needed, e.g.:
        # packages.default = pkgs.hello;
      }
    );
}
