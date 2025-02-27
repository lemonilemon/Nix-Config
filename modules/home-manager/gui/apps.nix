{
  pkgs,
  ...
}:
{
  home.packages = with pkgs; [
    webcord
    _1password-gui
  ];
  programs.ssh = {
    enable = true;
    forwardAgent = true;
    extraConfig = ''
      Host *
          IdentityAgent ~/.1password/agent.sock
      Include ~/.ssh/1Password/config
    '';
  };
  programs.zsh = {
    initExtraBeforeCompInit = ''
      eval "$(op completion zsh)"; compdef _op op
    '';
  };
}
