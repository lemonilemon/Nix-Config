{
  pkgs,
  lib,
  ...
}:
{
  networking.firewall = {
    enable = lib.mkDefault true;
    allowedTCPPorts = lib.mkDefault [
      22
      80
      443
    ];
    allowedUDPPorts = lib.mkDefault [ ];
    extraCommands = lib.mkDefault ''
      nft add rule inet filter input ip saddr 192.168.1.0/24 accept
    '';
  };
  environment.systemPackages = with pkgs; [
    nftables
  ];
}
