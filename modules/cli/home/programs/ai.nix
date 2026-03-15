{
  pkgs,
  ...
}:
{
  home.packages = with pkgs; [
    claude-code
    gemini-cli

    # for sandbox
    socat
    bubblewrap
  ];
  programs.opencode = {
    enable = true;
    settings = {
      theme = "catppuccin";
    };
  };

}
