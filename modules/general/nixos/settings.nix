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
        # 16 cores against 15 GiB of RAM: "auto" resolved max-jobs to 16, and
        # cores = 0 hands every one of those jobs all 16 cores, which permits up
        # to 256 concurrent compilers on a machine that cannot feed a tenth of
        # them. 4 x 4 keeps 16-way total parallelism but bounds how many heavy
        # C++ translation units can be resident at once.
        #
        # Cache hits are untouched: max-substitution-jobs is separate and stays
        # at 16, so the ordinary rebuild -- almost all substitution -- is exactly
        # as fast as before. Only genuine local builds are throttled.
        max-jobs = 4;
        cores = 4;
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

    # Builders run inside nix-daemon's cgroup, so a ceiling here bounds a build
    # without bounding evaluation, which happens in the nixos-rebuild client
    # process instead and is left alone.
    #
    # MemoryHigh throttles and reclaims rather than killing: a greedy build gets
    # pushed into swap and slows down instead of driving the machine to a global
    # OOM. That distinction matters here because systemd-oomd watches user.slice
    # while nix-daemon lives in system.slice -- so a build-driven pressure event
    # picks its victim from the desktop session rather than from the build.
    #
    # OOMScoreAdjust settles it the other way if the kernel does have to choose:
    # builders become the preferred victim, so a rebuild sacrifices itself rather
    # than the session. Children inherit the score.
    systemd.services.nix-daemon.serviceConfig = {
      MemoryHigh = "8G";
      OOMScoreAdjust = 500;
    };
  };
}
