{
  programs.nixvim = {
    plugins = {
      snacks = {
        enable = true;
        settings = {
          bigfile = {
            enabled = true;
          };
          quickfile = {
            enabled = true;
          };
          indent = {
            enabled = true;
          };
          input = {
            enabled = true;
          };
          notifier = {
            enabled = true;
          };
          scope = {
            enabled = true;
          };
          scroll = {
            enabled = true;
          };
          statuscolumn = {
            enabled = true;
          };
          words = {
            enabled = true;
          };
          image = {
            # Image viewer using Kitty Graphics Protocol
            enabled = false; # Not supported by zellij
          };
          rename = {
            # file renaming
            enabled = true;
          };
          bufdelete = {
            enabled = true;
          };
        };
      };
    };
  };
}
