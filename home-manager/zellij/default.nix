{
    ...
}: {
    programs.zellij = {
        enable = true;
        enableZshIntegration = true;
        settings = {
            theme = "catppuccin-mocha";
            default_shell = "zsh";
        };
    };
}
