{
  programs.zoxide = {
    enable = true;
    options = [
      # replace original cd command with zoxide
      "--cmd cd"
    ];
  };
}
