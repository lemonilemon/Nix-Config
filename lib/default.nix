{ inputs }:
let
  # Import helper modules
  builders = import ./builders.nix { inherit inputs; };
  # utils = import ./utils.nix { inherit inputs; };
in
{
  # Export all helper functions
  inherit (builders) mkSystem mkHome;
  # inherit (utils) mkUser enableFeatures ifEnabled;
}
