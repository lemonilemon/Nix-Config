{
    pkgs,
    username,
    hostname,
    ...
}: {
    wsl = {
        enable = true;
        defaultUser = "${username}";
    };
    networking.hostName = "${hostname}";
    programs.zsh.enable = true;
    environment.shells = [pkgs.zsh];
    environment.systemPackages = with pkgs; [
        wsl-open
    ];
    security.sudo.wheelNeedsPassword = false;

    users.users.${username} = {
    isNormalUser = true;
    # FIXME: change your shell here if you don't want zsh
    shell = pkgs.zsh;
        extraGroups = [
          "wheel"
        ];
        # FIXME: add your own hashed password
        # hashedPassword = "";
        # FIXME: add your own ssh public key
        # openssh.authorizedKeys.keys = [
        #   "ssh-rsa ..."
        # ];
    };

    services.automatic-timezoned.enable = true;
    time.timeZone = "Asia/Taipei";
    system.stateVersion = "23.05";
}
