{
  lib,
  pkgs,
  config,
  username,
  ...
}:
{
  config = lib.mkIf config.nixos.general-settings.nix.enable {
    nix = {
      package = lib.mkDefault pkgs.nix;
      settings = {
        trusted-users = lib.mkDefault [ username ];
        substituters = lib.mkDeafult [
          "https://cache.nixos.org/"
        ];
        extra-substituters = lib.mkDefault [
          "https://aseipp-nix-cache.global.ssl.fastly.net"
          "https://mirrors.ustc.edu.cn/nix-channels/store"
          "https://nix-community.cachix.org"
        ];
        extra-trusted-public-keys = lib.mkDefault [
          "nix-community.cachix.org-1:mB9FSh9qf2dCimDSUo8Zy7bkq5CX+/rkCWyvRCYg3Fs="
        ];
        accept-flake-config = lib.mkDefault true;
        auto-optimise-store = lib.mkDefault true;
      };
      extraOptions = lib.mkDefault ''experimental-features = nix-command flakes'';
    };
  };
}
