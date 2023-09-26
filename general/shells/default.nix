{
    pkgs,
    username, 
    ...
}: {
    users.defaultUserShell = pkgs.zsh;
    programs.zsh.enable = true;
    environment.shells = with pkgs; [
        zsh 
	bash
    ];
}
