{
  lib,
  config,
  pkgs,
  ...
}:
{
  config = lib.mkIf config.home.gui.development.web.enable {
    home.packages = with pkgs; [
      postman # API development tool
      tcpdump # Command-line packet analyzer
      nmap # Network discovery tool
      traceroute # Track the route taken by packets
      dig # DNS lookup utility
    ];
  };
}
