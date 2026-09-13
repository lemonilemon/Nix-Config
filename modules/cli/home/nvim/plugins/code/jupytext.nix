{
  ...
}:
{
  programs.nixvim = {
    plugins.jupytext = {
      enable = true;
      settings = {
        style = "percent";
      };
    };
  };
}
