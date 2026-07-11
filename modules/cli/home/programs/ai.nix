{ pkgs, ... }:
{
  home.packages = (
    with pkgs;
    [
      llm-agents.claude-code
      llm-agents.codex
      llm-agents.antigravity-cli
      llm-agents.skills
      llm-agents.ccusage

      # for sandbox
      socat
      bubblewrap
    ]
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

}
