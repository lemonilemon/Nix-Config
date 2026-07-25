{
  pkgs,
  lib,
  config,
  ...
}:
let
  cfg = config.nixos.general.firewall;
in
{
  config = lib.mkIf cfg.enable {
    networking.firewall = {
      enable = true;
      inherit (cfg) allowedTCPPorts allowedUDPPorts;

      # NixOS' default firewall backend is iptables and `nixos-fw` is its input
      # chain, so these go through iptables rather than nft. IPv4 only, which is
      # how these subnets have always been written.
      extraCommands = lib.concatMapStringsSep "\n" (
        subnet: "iptables -A nixos-fw -s ${subnet} -j nixos-fw-accept"
      ) cfg.trustedSubnets;
    };

    # The rules above are iptables rules on the iptables backend, so ship the
    # tool that can actually inspect them. `nft list ruleset` shows nothing here
    # and reads as "no firewall" to anyone debugging.
    environment.systemPackages = with pkgs; [
      iptables
    ];
  };
}
