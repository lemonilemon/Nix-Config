{
  config,
  lib,
  ...
}:
let
  inherit (config.catppuccin) flavor accent sources;
in
{
  config = lib.mkIf config.home.gui.apps.thunderbird.enable {
    programs.thunderbird = {
      enable = true;
      policies = lib.mkIf config.catppuccin.enable {
        Extensions.Install = [ "${sources.thunderbird}/${flavor}/${flavor}-${accent}.xpi" ];
      };
      profiles.default = {
        isDefault = true;
        settings = {
          "mailnews.start_page.enabled" = false;
          "datareporting.healthreport.uploadEnabled" = false;
          # 0: adaptive junk filter off, leaving spam to the mail provider
          "mail.server.default.spamLevel" = 0;
        };
      };
    };

    catppuccin.thunderbird.enable = false;
  };
}
