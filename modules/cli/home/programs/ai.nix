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

}
