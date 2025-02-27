{
  ...
}:
{
  imports = [
    ./gui
    ./general
  ];
  config = {
    gui-settings.enable = true;
    virtualisation.docker = {
      enable = true;
      enableOnBoot = true;
      autoPrune.enable = true;
    };
  };
}
