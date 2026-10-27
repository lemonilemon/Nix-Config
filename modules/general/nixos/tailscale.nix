{
  lib,
  config,
  username,
  ...
}:
{
  config = lib.mkIf config.nixos.general.tailscale.enable {
    services.tailscale = {
      enable = true;
      openFirewall = true;
      extraSetFlags = [ "--operator=${username}" ];
    };
  };
}
