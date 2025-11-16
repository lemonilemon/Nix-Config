{
  lib,
  config,
  ...
}:
{
  config = lib.mkIf config.home.cli.shells.zoxide.enable {
    programs.zoxide = {
      enable = true;
      options = [
        # replace original cd command with zoxide
        "--cmd cd"
      ];
    };
  };
}
