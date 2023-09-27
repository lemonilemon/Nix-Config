{username, ...}:
{
    nix = {
        extraOptions = ''
            experimental-features = nix-command flakes
            trusted-users = [ ${username} ];
        '';
    };
}
