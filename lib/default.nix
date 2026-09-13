{ inputs }:
let
  builders = import ./builders.nix { inherit inputs; };
in
{
  inherit (builders) mkSystem mkHome;
}
