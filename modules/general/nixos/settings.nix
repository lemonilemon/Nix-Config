{
  lib,
  pkgs,
  config,
  username,
  ...
}:
{
  config = lib.mkIf config.nixos.general-settings.nix.enable {
    # Nix settings needed (mkDefault has lower priority in this case)
    nix = {
      package = pkgs.nix;
      settings = {
        max-jobs = "auto";
        trusted-users = lib.mkAfter [
          "@wheel"
          username
        ];
        substituters = lib.mkAfter [
          "https://nix-community.cachix.org?priority"
        ];
        extra-substituters = [
          "https://aseipp-nix-cache.global.ssl.fastly.net"
          "https://mirrors.ustc.edu.cn/nix-channels/store"
        ];
        trusted-public-keys = lib.mkAfter [
          "nix-community.cachix.org-1:mB9FSh9qf2dCimDSUo8Zy7bkq5CX+/rkCWyvRCYg3Fs="
        ];
        accept-flake-config = true;
        auto-optimise-store = true;
        show-trace = true;
      };
      extraOptions = ''experimental-features = nix-command flakes'';
    };
  };
}
