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
        size = 14;
        name = "FiraCode Nerd Font";
      };
      keybindings = {
        "ctrl+c" = "copy_or_interrupt";
      };
      shellIntegration.enableZshIntegration = true;
    };
  };
}
