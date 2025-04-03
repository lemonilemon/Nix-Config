{
  pkgs,
  username,
  ...
}:
{
  programs._1password.enable = true;
  programs._1password-gui = {
    package = pkgs._1password-gui-beta;
    enable = true;
  };
  programs._1password-gui.polkitPolicyOwners = [ username ];
  # programs.zsh = {
  #   initExtraBeforeCompInit = ''
  #     eval "$(op completion zsh)"; compdef _op op
  #   '';
  # };
}
