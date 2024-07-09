{
    GUI,
    lib,
    ...
}: with lib; mkIf GUI {
    programs.kitty = {
        enable = true;
        theme = "Catppuccin-Mocha";
        shellIntegration.enableZshIntegration = true; 
        font = {
            name = "FiraCode";
            size = 12;
        };
    };
}
