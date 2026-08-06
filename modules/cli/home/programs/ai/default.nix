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

  # Skills versioned in the repo instead of installed by the `skills` CLI, so
  # their rules and sample excerpts evolve through the repo the way AGENTS.md
  # does. One attrset line per skill; skillLinks expands each entry into every
  # agent's skill root.
  nixManagedSkills = {
    my-voice = ./skills/my-voice;
  };

  # ~/.agents/skills is pi's native global skill directory and the shared home
  # of the `skills` CLI's installs; ~/.claude/skills is claude-code's. Links
  # are per-name (the CLI's own layout), so CLI-installed skills keep living
  # alongside, and both names resolve to the same store path, so a skill
  # cannot drift between agents.
  skillLinks = lib.concatMapAttrs (name: src: {
    ".agents/skills/${name}".source = src;
    ".claude/skills/${name}".source = src;
  }) nixManagedSkills;

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

  # Catppuccin Mocha for pi's TUI. pi ships only `dark` and `light`, so the
  # theme is authored here to sit alongside tmux, herdr and opencode. The
  # palette is inlined rather than imported from
  # modules/desktop/home/hyprland/theme/palette.nix because nothing under
  # modules/cli reaches across into the desktop tree; starship.nix keeps its own
  # copy of the same mocha values for the same reason.
  mochaVars = {
    mauve = "#cba6f7";
    pink = "#f5c2e7";
    red = "#f38ba8";
    peach = "#fab387";
    yellow = "#f9e2af";
    green = "#a6e3a1";
    teal = "#94e2d5";
    sapphire = "#74c7ec";
    blue = "#89b4fa";
    lavender = "#b4befe";
    text = "#cdd6f4";
    subtext0 = "#a6adc8";
    overlay1 = "#7f849c";
    overlay0 = "#6c7086";
    surface1 = "#45475a";
    surface0 = "#313244";
    mantle = "#181825";
    base = "#1e1e2e";
    crust = "#11111b";

    # Mocha has no tinted surfaces, and pi wants a background per tool outcome.
    # These are the base blended toward green/red far enough to read as a status
    # at a glance without competing with the diff colors on top of them.
    toolSuccessBg = "#1d2b21";
    toolErrorBg = "#2b1d24";
  };

  piTheme = (pkgs.formats.json { }).generate "pi-catppuccin-mocha.json" {
    name = "catppuccin-mocha";
    vars = mochaVars;
    # Every key here is required by pi's theme schema except thinkingMax, and
    # the schema rejects unknown keys, so this list tracks the schema exactly.
    colors = {
      accent = "mauve";
      border = "blue";
      borderAccent = "mauve";
      borderMuted = "surface1";
      success = "green";
      error = "red";
      warning = "yellow";
      muted = "overlay1";
      dim = "overlay0";
      text = "text";
      thinkingText = "overlay1";

      selectedBg = "surface1";
      userMessageBg = "surface0";
      userMessageText = "text";
      customMessageBg = "mantle";
      customMessageText = "text";
      customMessageLabel = "mauve";
      toolPendingBg = "mantle";
      toolSuccessBg = "toolSuccessBg";
      toolErrorBg = "toolErrorBg";
      toolTitle = "text";
      toolOutput = "subtext0";

      mdHeading = "yellow";
      mdLink = "blue";
      mdLinkUrl = "overlay0";
      mdCode = "peach";
      mdCodeBlock = "green";
      mdCodeBlockBorder = "surface1";
      mdQuote = "subtext0";
      mdQuoteBorder = "overlay0";
      mdHr = "surface1";
      mdListBullet = "mauve";

      toolDiffAdded = "green";
      toolDiffRemoved = "red";
      toolDiffContext = "overlay1";

      # Catppuccin's own syntax mapping, so code in pi matches nvim.
      syntaxComment = "overlay1";
      syntaxKeyword = "mauve";
      syntaxFunction = "blue";
      syntaxVariable = "text";
      syntaxString = "green";
      syntaxNumber = "peach";
      syntaxType = "yellow";
      syntaxOperator = "teal";
      syntaxPunctuation = "subtext0";

      # A ladder that climbs in saturation, so the current thinking level is
      # legible from the footer colour alone.
      thinkingOff = "surface1";
      thinkingMinimal = "overlay0";
      thinkingLow = "sapphire";
      thinkingMedium = "blue";
      thinkingHigh = "lavender";
      thinkingXhigh = "mauve";
      thinkingMax = "pink";

      bashMode = "green";
    };
    # Only used by `pi --export <file>`, which renders a session to HTML.
    export = {
      pageBg = mochaVars.crust;
      cardBg = mochaVars.base;
      infoBg = mochaVars.surface0;
    };
  };

  piSettings = (pkgs.formats.json { }).generate "pi-settings.json" {
    # pi speaks the Codex subscription's own API, so this reuses the ChatGPT
    # plan `codex` already bills against instead of metering tokens. Run
    # `/login` once and pick ChatGPT Plus/Pro; the OAuth tokens land in
    # ~/.pi/agent/auth.json, which stays writable because only settings.json is
    # symlinked into the store. Anthropic is deliberately not the default here:
    # a Claude Pro/Max login through a third-party harness draws from extra
    # usage and is billed per token rather than against the plan.
    defaultProvider = "openai-codex";
    # Newest top-tier model in the openai-codex catalog pi 0.82.0 ships.
    # gpt-5.6-terra and gpt-5.6-luna are the cheaper rungs, worth dropping to if
    # this burns through the plan's limits too quickly. Which ids the account is
    # actually entitled to depends on the ChatGPT tier, and pi refreshes the
    # catalog into ~/.pi/agent/models-store.json, so confirm with
    # `pi --list-models` after the first login.
    defaultModel = "gpt-5.6-sol";
    # Ctrl+P cycles everything the subscription exposes.
    enabledModels = [ "openai-codex/*" ];

    defaultThinkingLevel = "medium";
    theme = "catppuccin-mocha";

    # Nix owns the version, so the install/update ping has nothing to report.
    # It does not cover the version check; PI_SKIP_VERSION_CHECK does that.
    enableInstallTelemetry = false;

    # No `skills` entry on purpose. ~/.agents/skills is already one of pi's
    # native global skill directories and the `skills` CLI populates it, so the
    # set is shared with claude-code for free. ~/.claude/skills is left out
    # because it symlinks into ~/.agents/skills, and naming both would make
    # every skill collide with itself.

    # Read straight out of the pi package, so they track the pinned version
    # instead of being vendored into this repo. plan-mode is a directory whose
    # index.ts is the entry point; its utils.ts is imported, not loaded twice.
    extensions = [
      "${pkgs.llm-agents.pi}/libexec/pi/examples/extensions/plan-mode"
      "${pkgs.llm-agents.pi}/libexec/pi/examples/extensions/permission-gate.ts"
    ];
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

    # Nix owns pi's version, and `enableInstallTelemetry` does not cover this
    # check, so it would only ever point at an update that cannot be applied.
    home.sessionVariables.PI_SKIP_VERSION_CHECK = "1";
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
      # pi merges this with a project-local .pi/settings.json, which stays
      # writable, and keeps auth.json, trust.json and models-store.json as
      # separate files, so `/login` and `/trust` still work. The store copy is
      # read-only, so `/settings` and `pi install` cannot write here; declare
      # models, extensions and packages above instead.
      ".pi/agent/settings.json".source = piSettings;
      # One of pi's native global theme directories, matched by the `name` field
      # in the file rather than by filename.
      ".pi/agent/themes/catppuccin-mocha.json".source = piTheme;
      # Inferred path: kiro-cli has never run here, so ~/.kiro does not exist.
      # Its binary documents "global/workspace steering" as a default resource
      # and its global home is ~/.kiro. Re-verify after the first kiro-cli run.
      ".kiro/steering/00-global.md".source = agentInstructions;
    }
    // skillLinks;

    xdg.configFile = {
      "opencode/AGENTS.md".source = agentInstructions;
      # Only the file, not the directory: herdr writes herdr.log and its session
      # state next to it and needs the directory itself to stay writable.
      "herdr/config.toml".source = herdrConfig;
    };
  };

}
