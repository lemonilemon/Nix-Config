{ inputs, system }:
import inputs.nixpkgs {
  inherit system;
  config = import ./nixpkgs.nix;
  overlays = import ./overlays.nix { inherit inputs; };
}
