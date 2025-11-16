{
  lib,
  config,
  pkgs,
  username,
  ...
}:
{
  config = lib.mkIf config.nixos.gui.apps.enable {
    programs._1password.enable = true;
    programs._1password-gui = {
      package = pkgs._1password-gui-beta;
      enable = true;
    };
    programs._1password-gui.polkitPolicyOwners = [ username ];

    # set allowed browsers for 1 password
    environment.etc = {
      "1password/custom_allowed_browsers" = {
        text = ''
          zen
        '';
        mode = "0755";
      };
    };
    # programs.zsh = {
    #   initExtraBeforeCompInit = ''
    #     eval "$(op completion zsh)"; compdef _op op
    #   '';
    # };
  };
}
