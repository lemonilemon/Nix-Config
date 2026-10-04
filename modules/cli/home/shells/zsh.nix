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
            bindkey "''${key[Up]}" up-line-or-search # bind Up to up-line-or-search
            autoload -Uz edit-command-line
            zle -N edit-command-line
            bindkey -M viins '^G' edit-command-line
            bindkey -M vicmd '^G' edit-command-line
            KEYTIMEOUT=1 # hundredths of a second
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
