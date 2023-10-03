{    
     config,
     ... 
}: {
    home.file.".p10k.zsh".source = ./.p10k.zsh;
    programs.zsh = {
        enable = true;
        shellAliases = {
            ll="ls -alF";
            la="ls -A";
            l="ls -CF";
            ls="ls --color=auto";
            ".." = "cd ..";
        };
        zplug = {
            enable = true;
            plugins = [
                { name = "nix-community/nix-zsh-completions"; }
                { name = "romkatv/powerlevel10k"; tags = [ as:theme depth:1 ]; }
                { name = "zsh-users/zsh-autosuggestions"; }
                { name = "zsh-users/zsh-history-substring-search"; }
                { name = "marlonrichert/zsh-autocomplete"; }
                { name = "b4b4r07/enhancd"; tags = [ use:init.sh ]; }
                { name = "chisui/zsh-nix-shell"; }
            ];
        };
        history = {
            path = "${config.home.homeDirectory}/.zsh_history";
            save = 10000;
            size = 10000;
        };
        initExtraFirst = ''
            # Enable Powerlevel10k instant prompt. Should stay close to the top of ~/.zshrc.
            # Initialization code that may require console input (password prompts, [y/n]
            # confirmations, etc.) must go above this block; everything else may go below.
            if [[ -r "''${XDG_CACHE_HOME:-''$HOME/.cache}/p10k-instant-prompt-''${(%):-%n}.zsh" ]]; then
              source "''${XDG_CACHE_HOME:-''$HOME/.cache}/p10k-instant-prompt-''${(%):-%n}.zsh"
            fi
        '';
        initExtra = ''
            # To customize prompt, run `p10k configure` or edit ~/.p10k.zsh.
            [[ ! -f ~/.p10k.zsh ]] || source ~/.p10k.zsh
        '';
    };
}
