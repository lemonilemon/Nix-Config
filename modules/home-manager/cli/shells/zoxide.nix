{
  lib,
  config,
  ...
}:
{
  options = {
    cli-settings.shells.zoxide.enable = lib.mkOption {
      type = lib.types.bool;
      default = config.cli-settings.shells.enable;
      description = "Enable zoxide";
    };
  };
  config = lib.mkIf config.cli-settings.shells.zoxide.enable {
    programs.zoxide = {
      enable = true;
      options = [
        # replace original cd command with zoxide
        "--cmd cd"
      ];
    };
  };
}
