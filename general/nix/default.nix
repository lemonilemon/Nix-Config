{
  username,
  ...
}:
{
  nix = {
    settings = {
      trusted-users = [ username ];
      # FIXME: use your access tokens from secrets.json here to be able to clone private repos on GitHub and GitLab
      # access-tokens = [
      #   "github.com=${secrets.github_token}"
      #   "gitlab.com=OAuth2:${secrets.gitlab_token}"
      # ];
      substituters = [
        "https://cache.nixos.org/"
      ];
      extra-substituters = [
        "https://aseipp-nix-cache.global.ssl.fastly.net"
        "https://mirrors.ustc.edu.cn/nix-channels/store"
        "https://nix-community.cachix.org"
      ];
      extra-trusted-public-keys = [
        "nix-community.cachix.org-1:mB9FSh9qf2dCimDSUo8Zy7bkq5CX+/rkCWyvRCYg3Fs="
      ];
      accept-flake-config = true;
      auto-optimise-store = true;
    };

    extraOptions = ''experimental-features = nix-command flakes'';

    gc = {
      automatic = true;
      options = "--delete-older-than 7d";
    };
  };
}
