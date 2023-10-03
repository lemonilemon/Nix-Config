{
    username,
    ...
}: {
    nix = {
        settings = {
            trusted-users = [username];
              # FIXME: use your access tokens from secrets.json here to be able to clone private repos on GitHub and GitLab
              # access-tokens = [
              #   "github.com=${secrets.github_token}"
              #   "gitlab.com=OAuth2:${secrets.gitlab_token}"
              # ];

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
