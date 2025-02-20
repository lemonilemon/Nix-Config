{
  pkgs,
  ...
}:
{
  options = {
  };
  config = {
    programs.kitty = {
      enable = true;
      themeFile = "Catppuccin-Mocha";
      font = {
        name = "fira-code";
        package = pkgs.nerd-fonts.fira-code;
        size = 12;
      };
      keybindings = {
        "ctrl+c" = "copy_or_interrupt";
      };
      shellIntegration.enableZshIntegration = true;
    };
  };
}
