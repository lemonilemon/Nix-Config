{
  lib,
  config,
  ...
}:
{
  config = lib.mkIf config.home.cli.shells.zsh.enable {
    programs.zsh = {
      enable = true;
      shellAliases = {
        lg = "lazygit";
        ".." = "cd ..";
      };
      enableCompletion = false;
      zplug = {
        enable = true;
        plugins = [
          { name = "nix-community/nix-zsh-completions"; }
          {
            name = "marlonrichert/zsh-autocomplete";
            tags = [ "at:main" ];
          }
          { name = "loiccoyle/zsh-github-copilot"; }
          { name = "chisui/zsh-nix-shell"; }
        ];
      };
      history = {
        path = "${config.home.homeDirectory}/.zsh_history";
        save = 10000;
        size = 10000;
      };
      initContent =
        let
          zshConfigEarlyInit = lib.mkBefore ''
            # For non-NixOS distros.
            if [ -e ~/.nix-profile/etc/profile.d/nix.sh ]; then . ~/.nix-profile/etc/profile.d/nix.sh; fi # added by Nix installer
          '';
          zshConfig = ''
            # Bindkeys for zsh-github-copilot
            bindkey '^[|' zsh_gh_copilot_explain  # bind Alt+shift+\ to explain
            bindkey '^[\' zsh_gh_copilot_suggest  # bind Alt+\ to suggest
            bindkey "''${key[Up]}" up-line-or-search # bind Up to up-line-or-search
            export LANGUAGE=en_US.UTF-8
            export LC_ALL=en_US.UTF-8
            export LANG=en_US.UTF-8
            export LC_CTYPE=en_US.UTF-8
            # For direnv to work properly.
            eval "$(direnv hook zsh)"
          '';
        in
        lib.mkMerge [
          zshConfigEarlyInit
          zshConfig
        ];
    };
  };
}
