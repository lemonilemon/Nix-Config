{
  programs.nixvim = {
    plugins = {
      #https://nix-community.github.io/nixvim/plugins/competitest/index.html?highlight=competities#competitest
      competitest = {
        enable = true;
        settings = {
          compile_command = {
            cpp = {
              args = [
                "-DLOCAL"
                "$(FNAME)"
                "-o"
                "$(FNOEXT)"
                "-Wall"
                "-Wextra"
              ];
              exec = "g++";
            };
          };
          run_command = {
            cpp = {
              exec = "./$(FNOEXT)";
            };
          };
          evaluate_template_modifiers = true;
          received_problems_path = "$(HOME)/cp/$(JUDGE)/$(CONTEST)/$(PROBLEM)/main.$(FEXT)";
          template_file = "$(HOME)/cp/templates/template.$(FEXT)";
        };
      };
    };
  };
}
