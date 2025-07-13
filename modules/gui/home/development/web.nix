{
  lib,
  config,
  pkgs,
  ...
}:
{
  config = lib.mkIf config.home.gui-settings.development.web.enable {
    home.packages = with pkgs; [
      postman # API development tool
    ];
  };
}
