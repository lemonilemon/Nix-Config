{
  lib,
  pkgs,
  config,
  username,
  ...
}:
{
  config = lib.mkIf config.nixos.general.nix.enable {
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
          "https://nix-community.cachix.org"
        ];
        extra-substituters = [
          "https://aseipp-nix-cache.global.ssl.fastly.net"
          "https://mirrors.ustc.edu.cn/nix-channels/store"
          "https://cache.numtide.com"
          "https://lemonilemon.cachix.org"
        ];
        trusted-public-keys = lib.mkAfter [
          "nix-community.cachix.org-1:mB9FSh9qf2dCimDSUo8Zy7bkq5CX+/rkCWyvRCYg3Fs="
          "niks3.numtide.com-1:DTx8wZduET09hRmMtKdQDxNNthLQETkc/yaX7M4qK0g="
          "lemonilemon.cachix.org-1:3JBE3d5E5WuJRgOXNz+I5BUG+HRtBecADu0RBBJV1qI="
        ];
        accept-flake-config = true;
        auto-optimise-store = true;
        show-trace = true;
      };
      extraOptions = "experimental-features = nix-command flakes";
    };
  };
}
