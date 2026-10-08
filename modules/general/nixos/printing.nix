{
  lib,
  config,
  ...
}:
{
  config = lib.mkIf config.nixos.general.printing.enable {
    services.printing.enable = true;

    services.avahi = {
      enable = true;
      nssmdns4 = true;
    };
  };
}
