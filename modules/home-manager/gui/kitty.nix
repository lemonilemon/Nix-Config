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
        size = 16;
        name = "FiraCode Nerd Font";
      };
      keybindings = {
        "ctrl+c" = "copy_or_interrupt";
      };
      settings = {
        confirm_os_window_close = 0;
        dynamic_background_opacity = true;
        enable_audio_bell = false;
        mouse_hide_wait = "-1.0";
        window_padding_width = 10;
        background_opacity = "0.7";
        strip_trailing_spaces = "smart";
        paste_actions = "confirm-if-large";
      };
      shellIntegration.enableZshIntegration = true;
    };
  };
}
