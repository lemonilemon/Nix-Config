{
  lib,
  config,
  pkgs,
  ...
}:
let
  herdr = pkgs.llm-agents.herdr;
  tmux = config.programs.tmux.package;

  tmuxHerdr = pkgs.writeShellApplication {
    name = "tmux-herdr";
    runtimeInputs = [ tmux ];
    text = ''
      session=$1
      start_dir=$2
      herdr_session=''${session//[^A-Za-z0-9._-]/_}

      read -r window active pane_command < <(
        tmux list-windows -t "=$session" -f '#{==:#{window_name},herdr}' \
          -F '#{window_id} #{window_active} #{pane_current_command}'
      ) || true

      if [[ -z ''${window:-} ]]; then
        tmux new-window -t "=$session:" -n herdr -c "$start_dir" \
          ${herdr}/bin/herdr --session "$herdr_session"
      elif [[ $pane_command != herdr ]]; then
        tmux respawn-window -k -t "$window" -c "$start_dir" \
          ${herdr}/bin/herdr --session "$herdr_session"
        tmux select-window -t "$window"
      elif [[ $active == 1 ]]; then
        tmux last-window -t "=$session"
      else
        tmux select-window -t "$window"
      fi
    '';
  };

  herdrAttention = pkgs.writeShellApplication {
    name = "herdr-attention";
    runtimeInputs = [
      herdr
      pkgs.jq
      pkgs.coreutils
    ];
    text = ''
      red=$1
      green=$2
      fg=$3
      output=""

      while read -r name; do
        read -r blocked finished < <(
          HERDR_SESSION=$name timeout 1 herdr agent list 2>/dev/null |
            jq -r '.result.agents | map(.agent_status)
              | "\(map(select(. == "blocked")) | length) \(map(select(. == "done")) | length)"'
        ) || continue
        if (( blocked + finished == 0 )); then
          continue
        fi

        segment=$name
        if (( blocked > 0 )); then
          segment+=" #[fg=$red]●$blocked#[fg=$fg]"
        fi
        if (( finished > 0 )); then
          segment+=" #[fg=$green]●$finished#[fg=$fg]"
        fi
        output+="''${output:+  }$segment"
      done < <(herdr session list --json | jq -r '.sessions[] | select(.running) | .name')

      printf '%s' "$output"
    '';
  };

  attention = "#(${herdrAttention}/bin/herdr-attention '#{E:@thm_red}' '#{E:@thm_green}' '#{E:@thm_fg}')";

  statusModule = pkgs.writeText "herdr-status-module.conf" ''
    %hidden MODULE_NAME="herdr"
    set -g @catppuccin_herdr_icon "󰚩 "
    set -gF @catppuccin_herdr_color "#{E:@thm_mauve}"
    set -g @catppuccin_herdr_text " ${attention}"
    source "${pkgs.tmuxPlugins.catppuccin}/share/tmux-plugins/catppuccin/utils/status_module.conf"
  '';
in
{
  config = lib.mkIf (config.home.cli.multiplexer.tmux.enable && config.home.cli.programs.enable) {
    programs.tmux.extraConfig = ''
      bind a run-shell "${tmuxHerdr}/bin/tmux-herdr #{q:session_name} #{q:session_path}"

      source ${statusModule}
      set -ag status-right "#{?${attention},#{E:@catppuccin_status_herdr},}"
    '';
  };
}
