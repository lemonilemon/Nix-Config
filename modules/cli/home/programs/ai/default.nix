{
  lib,
  config,
  pkgs,
  ...
}:
let
  # One file, six symlinks, all resolving to the same store object. Paths were
  # confirmed against each installed binary. antigravity-cli has no global
  # instruction path of its own, so ~/AGENTS.md covers it, along with anything
  # else that resolves AGENTS.md by walking up from the workspace.
  agentInstructions = ./AGENTS.md;

  # herdr runs nested inside tmux here, so both halves of its chrome are tuned to
  # stay out of tmux's way: tmux keeps the top row and ctrl+b, herdr keeps the
  # left edge and ctrl+g. Verified against `herdr config check`.
  herdrConfig = (pkgs.formats.toml { }).generate "herdr-config.toml" {
    # herdr persists the first-run notification answer back into this file, and
    # the store copy is read-only, so the prompt would return on every launch.
    onboarding = false;

    theme.name = "catppuccin";

    keys = {
      # tmux already owns ctrl+b, and nvim's cmp owns ctrl+space. ctrl+g is
      # unbound in tmux, zsh, nvim and kitty, and tmux forwards it untouched.
      prefix = "ctrl+g";
    };

    ui = {
      # tmux's status line is pinned to the top (status-position top) and
      # herdr's tab row would stack right under it. herdr cannot move that row,
      # so drop it while a workspace has a single tab; the sidebar then carries
      # herdr's whole interface down the left edge. prefix+c still makes tabs,
      # and the row reappears once a second one exists.
      hide_tab_bar_when_single_tab = true;
    };

    update = {
      # Nix owns the version; `herdr update` cannot write to the store anyway.
      version_check = false;
    };
  };

  # herdr prints its completion script rather than shipping one, so generate the
  # file at build time. zsh finds it through the fpath loop over $NIX_PROFILES
  # that home-manager writes into ~/.zshrc, which runs before zsh-autocomplete
  # calls compinit.
  herdrZshCompletions =
    pkgs.runCommand "herdr-zsh-completions"
      {
        nativeBuildInputs = [ pkgs.llm-agents.herdr ];
      }
      ''
        export HOME=$TMPDIR
        mkdir -p "$out/share/zsh/site-functions"
        herdr completion zsh > "$out/share/zsh/site-functions/_herdr"
      '';
in
{
  config = lib.mkIf config.home.cli.programs.enable {
    home.packages = (
      (with pkgs; [
        llm-agents.claude-code
        llm-agents.codex
        llm-agents.antigravity-cli
        llm-agents.pi
        llm-agents.skills
        llm-agents.ccusage
        kiro-cli

        # Runs several of the agents above side by side in one terminal, with
        # per-pane state (working / blocked / done) inferred from the process.
        llm-agents.herdr

        # for sandbox
        socat
        bubblewrap
      ])
      ++ lib.optional config.home.cli.shells.zsh.enable herdrZshCompletions
    );
    programs.opencode = {
      enable = true;
      package = pkgs.llm-agents.opencode;
      tui = {
        theme = "catppuccin";
      };
      settings = {
        agent = {
          team-lead = {
            mode = "primary";
            description = "Team Lead / Builder - Oversees the project, orchestrates tasks, delegates to other specialists when needed";
            temperature = 0.25;
            prompt = ''

              # Role
              You are the Team Lead. Your primary job is to orchestrate the software development lifecycle by delegating tasks to specialized sub-agents.

              # Guidelines
              1. Never guess requirements. If a request is vague, invoke the @product-manager to define the user story.
              2. Once requirements are clear, delegate implementation to the @developer.
              3. After implementation, immediately pass the code to the @code-reviewer agents.
              4. Synthesize their feedback and present a clear, actionable summary. Do not output bloated conversational filler.
            '';
            tools = {
              read = true;
              write = true;
              edit = true;
              bash = true;
              git = true;
              task = true;
              search = true;
            };
          };
          product-manager = {
            mode = "subagent";
            description = "Product Manager - Focuses on user stories, clarifies requirements, and ensures the product meets user needs, takes care of the edge cases";
            temperature = 0.4;
            prompt = ''

              You are the Product Manager. When delegated a task by the Team Lead, your primary responsibility is to clarify requirements and define user stories. You should ask probing questions to ensure you fully understand the user's needs and the problem at hand. Focus on defining clear, actionable user stories that the developer can implement. Consider edge cases and potential pitfalls to ensure a robust implementation. Do not output bloated conversational filler.
            '';
            tools = {
              read = true;
              write = true; # suggest file creation
              edit = false;
              bash = false;
              task = false;
            };

          };
          developer = {
            mode = "subagent";
            description = "Developer - Implement codes, refactors, add features.";
            temperature = 0.2;
            prompt = ''

              You are the developer, who implement the code based on the requirements defined by the product manager. Focuses on implementation and functionality with clean idiomatic code. Add comments to clarify complex logic. Follow project style. Do not change project structure without told.
            '';
            tools = {
              read = true;
              write = true;
              edit = true;
              bash = true;
              task = false;
            };
          };
          code-reviewer = {
            mode = "subagent";
            description = "Code Reviewer - Reviews code for quality, correctness, and adherence to best practices. Provides constructive feedback.";
            temperature = 0.1;
            prompt = ''

              You are the code reviewer, who review the code implemented by the developer. Focuses on code quality, correctness, and adherence to best practices. Provide constructive feedback with clear explanations and actionable suggestions for improvement. Do not output bloated conversational filler.
            '';
            tools = {
              read = true;
              write = false;
              edit = false;
              bash = false;
              task = false;
            };
          };

        };
      };
    };

    home.file = {
      # It is the only path that reaches `agy` (antigravity-cli), which has no
      # global instruction file of its own; tree-walking agents also pick this
      # up for any project under $HOME.
      "AGENTS.md".source = agentInstructions;
      ".claude/CLAUDE.md".source = agentInstructions;
      ".codex/AGENTS.md".source = agentInstructions;
      # pi walks up from the cwd too, so ~/AGENTS.md would cover projects under
      # $HOME; this is its documented global path, which also applies outside
      # $HOME and is loaded before any project-trust decision.
      ".pi/agent/AGENTS.md".source = agentInstructions;
      # Inferred path: kiro-cli has never run here, so ~/.kiro does not exist.
      # Its binary documents "global/workspace steering" as a default resource
      # and its global home is ~/.kiro. Re-verify after the first kiro-cli run.
      ".kiro/steering/00-global.md".source = agentInstructions;
    };

    xdg.configFile = {
      "opencode/AGENTS.md".source = agentInstructions;
      # Only the file, not the directory: herdr writes herdr.log and its session
      # state next to it and needs the directory itself to stay writable.
      "herdr/config.toml".source = herdrConfig;
    };
  };

}
