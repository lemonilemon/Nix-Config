{
  lib,
  config,
  ...
}:
{
  config = lib.mkIf config.nixos.gui.apps.enable {
    programs.localsend = {
      enable = true;

      # LocalSend is only half an app without this: it announces and receives on
      # 53317, so with the port shut it can still send but no peer can start a
      # transfer towards this machine, and it will not appear in anyone else's
      # device list.
      #
      # This does not fight modules/general/nixos/firewall.nix. That module sets
      # networking.firewall.allowed{TCP,UDP}Ports from nixos.general.firewall,
      # and those options are list types, so the definitions merge instead of
      # one overriding the other -- 53317 lands alongside 22/80/443 rather than
      # replacing them.
      #
      # It matters most on the laptop, which carries no trustedSubnets and so is
      # actually filtered. The desktop already bypasses all ports for
      # 192.168.0.0/24, so there the port is redundant on the LAN and still
      # required on any other network.
      openFirewall = true;
    };
  };
}
